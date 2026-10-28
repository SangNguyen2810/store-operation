package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/cache"
	"github.com/SangNguyen2810/store-operation/api/internal/analytics/adapters/postgres"
	platformcache "github.com/SangNguyen2810/store-operation/api/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/testutil"
)

// These drive the real router, use cases and SQL over HTTP against Pagila (and
// Redis for the cached variants). Only the network listener is left out.

var asOf = time.Date(2022, 7, 27, 0, 0, 0, 0, time.UTC)

func plainHandler(t *testing.T) http.Handler {
	t.Helper()
	pool := testutil.Pool(t)
	return newHandler(pool, postgres.New(pool), asOf)
}

func cachedHandler(t *testing.T, rc *redis.Client) http.Handler {
	t.Helper()
	pool := testutil.Pool(t)
	repo := cache.New(platformcache.New(rc, time.Minute), postgres.New(pool))
	return newHandler(pool, repo, asOf)
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("bad json %q: %v", rec.Body, err)
	}
}

func TestHealthAndMeta(t *testing.T) {
	h := plainHandler(t)
	var health, meta map[string]string
	decode(t, get(h, "/healthz"), &health)
	decode(t, get(h, "/api/meta"), &meta)
	if health["status"] != "ok" || meta["asOf"] != "2022-07-27" {
		t.Errorf("health %v meta %v", health, meta)
	}
}

func TestStoresEndpoint(t *testing.T) {
	var stores []struct {
		ID    int    `json:"id"`
		Label string `json:"label"`
	}
	decode(t, get(plainHandler(t), "/api/stores"), &stores)
	if len(stores) != 2 || stores[0].ID != 1 || stores[1].ID != 2 || stores[0].Label == "" {
		t.Errorf("stores = %+v", stores)
	}
}

type metric struct {
	Value    float64  `json:"value"`
	Previous float64  `json:"previous"`
	DeltaPct *float64 `json:"deltaPct"`
}

type kpis struct {
	Revenue         metric `json:"revenue"`
	Rentals         metric `json:"rentals"`
	ActiveCustomers metric `json:"activeCustomers"`
	AvgPayment      metric `json:"avgPayment"`
}

func TestKPIsComparePreviousWindowOverHTTP(t *testing.T) {
	h := plainHandler(t)
	var k kpis
	decode(t, get(h, "/api/kpis?from=2022-06-28&to=2022-07-27"), &k)
	if k.Revenue.Value <= 0 || k.Rentals.Value <= 0 || k.ActiveCustomers.Value <= 0 {
		t.Fatalf("expected real figures, got %+v", k)
	}
	// The previous figures must equal the same query asked for the preceding 30 days.
	var prev kpis
	decode(t, get(h, "/api/kpis?from=2022-05-29&to=2022-06-27"), &prev)
	if k.Revenue.Previous != prev.Revenue.Value || k.Rentals.Previous != prev.Rentals.Value {
		t.Errorf("previous %+v does not match the preceding window %+v", k, prev)
	}
	if k.Revenue.Previous > 0 && k.Revenue.DeltaPct == nil {
		t.Error("deltaPct missing though there is a previous value")
	}
	// The default window is the 30 days ending at as-of.
	var def kpis
	decode(t, get(h, "/api/kpis"), &def)
	if def.Revenue.Value != k.Revenue.Value || def.Rentals.Value != k.Rentals.Value {
		t.Errorf("default window %+v differs from explicit %+v", def, k)
	}
}

func TestStoreFilterSplitsTheTotal(t *testing.T) {
	h := plainHandler(t)
	var all, one, two kpis
	decode(t, get(h, "/api/kpis?from=2022-01-01&to=2022-12-31"), &all)
	decode(t, get(h, "/api/kpis?from=2022-01-01&to=2022-12-31&store=1"), &one)
	decode(t, get(h, "/api/kpis?from=2022-01-01&to=2022-12-31&store=2"), &two)
	if d := all.Revenue.Value - one.Revenue.Value - two.Revenue.Value; d > 0.01 || d < -0.01 {
		t.Errorf("store 1 + store 2 revenue = %.2f, all = %.2f", one.Revenue.Value+two.Revenue.Value, all.Revenue.Value)
	}
	if one.Revenue.Value == 0 || two.Revenue.Value == 0 {
		t.Errorf("each store should have sales: %v %v", one.Revenue.Value, two.Revenue.Value)
	}
}

func TestUnknownStoreIsEmptyNotAnError(t *testing.T) {
	h := plainHandler(t)
	var k kpis
	decode(t, get(h, "/api/kpis?store=999"), &k)
	if k.Revenue.Value != 0 || k.Rentals.Value != 0 {
		t.Errorf("unknown store should have no figures: %+v", k)
	}
	var films []map[string]any
	decode(t, get(h, "/api/top/films?store=999"), &films)
	if films == nil || len(films) != 0 {
		t.Errorf("films = %v, want an empty (non-null) list", films)
	}
}

func TestRevenueAndFilmsOverHTTP(t *testing.T) {
	h := plainHandler(t)
	var pts []struct {
		Bucket  string  `json:"bucket"`
		StoreID int     `json:"storeId"`
		Revenue float64 `json:"revenue"`
	}
	decode(t, get(h, "/api/revenue?from=2022-03-01&to=2022-03-31&granularity=day"), &pts)
	if len(pts) != 62 || pts[0].Bucket != "2022-03-01" {
		t.Errorf("got %d points, first %+v", len(pts), pts[0])
	}
	decode(t, get(h, "/api/revenue?from=2022-03-01&to=2022-03-31&granularity=month"), &pts)
	if len(pts) != 2 {
		t.Errorf("monthly series has %d points, want 2 (one per store)", len(pts))
	}

	var films []struct {
		Title   string  `json:"title"`
		Rentals int     `json:"rentals"`
		Revenue float64 `json:"revenue"`
	}
	decode(t, get(h, "/api/top/films?limit=5"), &films)
	if len(films) != 5 {
		t.Fatalf("got %d films, want 5", len(films))
	}
	for i := 1; i < len(films); i++ {
		if films[i].Revenue > films[i-1].Revenue {
			t.Errorf("films not sorted by revenue: %+v", films)
		}
	}
}

func TestErrorsOverHTTP(t *testing.T) {
	h := plainHandler(t)
	for target, status := range map[string]int{
		"/api/kpis?from=bad":                      http.StatusBadRequest,
		"/api/revenue?granularity=year":           http.StatusBadRequest,
		"/api/top/films?limit=999":                http.StatusBadRequest,
		"/api/nope":                               http.StatusNotFound,
		"/api/kpis?from=2022-07-02&to=2022-07-01": http.StatusBadRequest,
	} {
		rec := get(h, target)
		if rec.Code != status {
			t.Errorf("%s: status %d, want %d", target, rec.Code, status)
		}
	}
	rec := get(h, "/api/kpis?from=bad")
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content-type %q", ct)
	}
}

// The cache must be invisible to clients: same bytes cold, warm, and uncached.
var cachedURLs = []string{
	"/api/kpis",
	"/api/kpis?store=1",
	"/api/revenue?from=2022-03-01&to=2022-03-31&granularity=week",
	"/api/top/films?limit=7&store=2",
	"/api/stores",
}

func clearKeys(t *testing.T, rc *redis.Client) {
	t.Helper()
	clean := func() {
		keys, _ := rc.Keys(context.Background(), "analytics:v1:*").Result()
		if len(keys) > 0 {
			rc.Del(context.Background(), keys...)
		}
	}
	clean()
	t.Cleanup(clean)
}

func TestCachedResponsesAreByteIdentical(t *testing.T) {
	rc := testutil.Redis(t)
	clearKeys(t, rc)
	plain, cached := plainHandler(t), cachedHandler(t, rc)

	for _, u := range cachedURLs {
		want := get(plain, u)
		cold := get(cached, u)
		warm := get(cached, u)
		if want.Code != 200 || cold.Body.String() != want.Body.String() || warm.Body.String() != want.Body.String() {
			t.Errorf("%s differs\nplain: %s\ncold:  %s\nwarm:  %s", u, want.Body, cold.Body, warm.Body)
		}
	}
	keys, _ := rc.Keys(context.Background(), "analytics:v1:*").Result()
	// kpis: 2 windows x 2 calls (all, store 1), revenue 1, films 1; /api/stores is not cached.
	if len(keys) != 6 {
		t.Errorf("redis holds %d analytics keys, want 6: %v", len(keys), keys)
	}
	for _, k := range keys {
		if !strings.HasPrefix(k, "analytics:v1:") {
			t.Errorf("unexpected key %q", k)
		}
	}
}

func TestCacheServesFromRedisNotTheDatabase(t *testing.T) {
	rc := testutil.Redis(t)
	clearKeys(t, rc)
	h := cachedHandler(t, rc)
	var first kpis
	decode(t, get(h, "/api/kpis"), &first)

	// Overwrite the cached current window with a marker: the next request must return it.
	keys, _ := rc.Keys(context.Background(), "analytics:v1:totals:2022-06-28*").Result()
	if len(keys) != 1 {
		t.Fatalf("totals keys = %v", keys)
	}
	rc.Set(context.Background(), keys[0], `{"Revenue":12345.67,"Payments":1,"Rentals":1,"ActiveCustomers":1}`, time.Minute)
	var second kpis
	decode(t, get(h, "/api/kpis"), &second)
	if second.Revenue.Value != 12345.67 {
		t.Errorf("revenue = %v; the entry in Redis was not used", second.Revenue.Value)
	}
}

func TestDeadRedisStillAnswersCorrectly(t *testing.T) {
	pool := testutil.Pool(t)
	dead := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 100 * time.Millisecond, MaxRetries: -1})
	t.Cleanup(func() { _ = dead.Close() })
	h := newHandler(pool, cache.New(platformcache.New(dead, time.Minute), postgres.New(pool)), asOf)

	for _, u := range cachedURLs {
		want, got := get(plainHandler(t), u), get(h, u)
		if got.Code != 200 || got.Body.String() != want.Body.String() {
			t.Errorf("%s with Redis down: status %d\n got %s\nwant %s", u, got.Code, got.Body, want.Body)
		}
	}
}
