package cache

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/app"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
	platformcache "github.com/SangNguyen2810/store-operation/app/api/internal/platform/cache"
)

const ttl = time.Minute

// fakeInner stands in for the Postgres adapter and counts how often it is asked.
type fakeInner struct {
	app.Repo
	totals, revenue, films atomic.Int32
	revenueOut             []domain.RevenuePoint
}

func (f *fakeInner) Totals(_ context.Context, flt domain.Filter) (domain.Totals, error) {
	f.totals.Add(1)
	// The window start makes every window's answer different.
	return domain.Totals{Revenue: float64(flt.From.Day()) + 0.25, Payments: 3, Rentals: 2, ActiveCustomers: 1}, nil
}

func (f *fakeInner) Revenue(context.Context, domain.Filter, domain.Granularity) ([]domain.RevenuePoint, error) {
	f.revenue.Add(1)
	return f.revenueOut, nil
}

func (f *fakeInner) TopFilms(_ context.Context, _ domain.Filter, limit int) ([]domain.FilmStat, error) {
	f.films.Add(1)
	return []domain.FilmStat{{FilmID: limit, Title: "T", Rentals: 1, Revenue: 9.99}}, nil
}

// fakeService is the redis service's HTTP contract over a map.
type fakeService struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (f *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.EscapedPath(), "/v1/cache/")
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		v, ok := f.m[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(v)
	case http.MethodPut:
		f.m[key], _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (f *fakeService) exists(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.m[key]
	return ok
}

func (f *fakeService) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for k := range f.m {
		out = append(out, k)
	}
	return out
}

func newService(t *testing.T) (*platformcache.Cache, *fakeService) {
	t.Helper()
	fake := &fakeService{m: map[string][]byte{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	c, err := platformcache.New(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return c, fake
}

func setup(t *testing.T) (*Repo, *fakeInner, *fakeService) {
	t.Helper()
	c, fake := newService(t)
	inner := &fakeInner{}
	return New(c, inner, ttl), inner, fake
}

func day(d int) time.Time { return time.Date(2022, 7, d, 0, 0, 0, 0, time.UTC) }

func window(from, to int) domain.Filter { return domain.Filter{From: day(from), To: day(to)} }

func ptr(i int) *int { return &i }

func TestRepeatedQueriesAreServedFromTheCache(t *testing.T) {
	repo, inner, svc := setup(t)
	ctx := context.Background()
	f := window(1, 10)

	first, err := repo.Totals(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Totals(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if n := inner.totals.Load(); n != 1 || first != second {
		t.Fatalf("inner queried %d times, fresh %+v, cached %+v", n, first, second)
	}
	if !svc.exists(key("totals", f)) {
		t.Fatalf("expected key %q in the redis service, have %v", key("totals", f), svc.keys())
	}
}

func TestEachKPIWindowIsCachedOnItsOwn(t *testing.T) {
	repo, inner, _ := setup(t)
	svc := app.NewService(repo, nil, day(1))
	ctx := context.Background()
	f := window(10, 19)

	for range 3 {
		if _, err := svc.KPIs(ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	// One call asks for the current and the previous window; repeats ask for neither again.
	if n := inner.totals.Load(); n != 2 {
		t.Fatalf("inner queried %d times for 3 KPI calls, want 2", n)
	}
}

func TestDifferentQueriesUseDifferentKeys(t *testing.T) {
	repo, inner, _ := setup(t)
	ctx := context.Background()
	base := window(1, 10)

	queries := []domain.Filter{
		base,
		window(1, 11),
		window(2, 10),
		{From: base.From, To: base.To, StoreID: ptr(1)},
		{From: base.From, To: base.To, StoreID: ptr(2)},
	}
	for _, q := range queries {
		repo.Totals(ctx, q)
	}
	if n := inner.totals.Load(); int(n) != len(queries) {
		t.Fatalf("inner queried %d times for %d distinct filters", n, len(queries))
	}

	repo.Revenue(ctx, base, domain.Day)
	repo.Revenue(ctx, base, domain.Week)
	repo.Revenue(ctx, base, domain.Day)
	if n := inner.revenue.Load(); n != 2 {
		t.Fatalf("revenue queried %d times for 2 granularities, want 2", n)
	}

	a, _ := repo.TopFilms(ctx, base, 5)
	b, _ := repo.TopFilms(ctx, base, 10)
	if inner.films.Load() != 2 || a[0].FilmID == b[0].FilmID {
		t.Fatalf("limits 5 and 10 must not share an entry: %+v %+v", a, b)
	}
}

func TestDomainValuesSurviveTheRoundTrip(t *testing.T) {
	repo, inner, _ := setup(t)
	ctx := context.Background()
	inner.revenueOut = []domain.RevenuePoint{{Bucket: "2022-07-01", StoreID: 2, Revenue: 1234.56}, {Bucket: "2022-07-02", StoreID: 1, Revenue: 0.1}}

	fresh, _ := repo.Revenue(ctx, window(1, 2), domain.Day)
	cached, _ := repo.Revenue(ctx, window(1, 2), domain.Day)
	if !reflect.DeepEqual(fresh, cached) {
		t.Fatalf("fresh %+v, cached %+v", fresh, cached)
	}

	// An empty result must stay an empty list, not become null in the JSON response.
	inner.revenueOut = []domain.RevenuePoint{}
	repo.Revenue(ctx, window(3, 4), domain.Day)
	empty, _ := repo.Revenue(ctx, window(3, 4), domain.Day)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty slice came back as %#v", empty)
	}
}

func TestStoresAreCachedForAnHour(t *testing.T) {
	c, svc := newService(t)
	inner := &storesOnly{}
	repo := New(c, inner, ttl)

	first, err := repo.Stores(context.Background())
	if err != nil || len(first) != 1 {
		t.Fatalf("got %+v, %v", first, err)
	}
	second, _ := repo.Stores(context.Background())
	if inner.calls.Load() != 1 || !reflect.DeepEqual(first, second) {
		t.Fatalf("queried %d times; fresh %+v, cached %+v", inner.calls.Load(), first, second)
	}
	if !svc.exists("analytics:v1:stores") {
		t.Fatalf("expected the stores key, have %v", svc.keys())
	}
	if storesTTL != time.Hour {
		t.Errorf("storesTTL = %v, want 1h", storesTTL)
	}
}

func TestStoreDetailsSurviveTheRoundTrip(t *testing.T) {
	c, _ := newService(t)
	want := domain.Store{
		ID: 2, Label: "Hamilton, New Zealand", Manager: "Lavone O'Reilly", Phone: "657282285970",
		Address: "478 Joliet Way", District: "Hamilton", PostalCode: "77948", City: "Hamilton", Country: "New Zealand",
		InventoryItems: 2311, ActiveCustomers: 273,
	}
	repo := New(c, &storesOnly{stores: []domain.Store{want}}, ttl)
	repo.Stores(context.Background())
	got, _ := repo.Stores(context.Background())
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

type storesOnly struct {
	app.Repo
	calls  atomic.Int32
	stores []domain.Store
}

func (s *storesOnly) Stores(context.Context) ([]domain.Store, error) {
	s.calls.Add(1)
	if s.stores != nil {
		return s.stores, nil
	}
	return []domain.Store{{ID: 1, Label: "A"}}, nil
}
