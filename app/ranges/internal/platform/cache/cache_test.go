package cache

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const ttl = time.Minute

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// clock is a manually advanced time source shared by the cache and the fake service.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// fakeService speaks the redis service's HTTP contract over a map.
type fakeService struct {
	clk *clock
	mu  sync.Mutex
	m   map[string]fakeEntry
	// status, when set, answers every request with it instead (an outage).
	status  atomic.Int32
	gets    atomic.Int32
	puts    atomic.Int32
	badTTL  string
	lastTTL string
}

type fakeEntry struct {
	val     []byte
	expires time.Time // zero: never
}

func (f *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s := f.status.Load(); s != 0 {
		w.WriteHeader(int(s))
		return
	}
	key := strings.TrimPrefix(r.URL.EscapedPath(), "/v1/cache/")
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		f.gets.Add(1)
		e, ok := f.m[key]
		if !ok || (!e.expires.IsZero() && !e.expires.After(f.clk.now())) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch {
		case f.badTTL != "":
			w.Header().Set(ttlHeader, f.badTTL)
		case !e.expires.IsZero():
			w.Header().Set(ttlHeader, strconv.Itoa(int(e.expires.Sub(f.clk.now()).Seconds())))
		}
		_, _ = w.Write(e.val)
	case http.MethodPut:
		f.puts.Add(1)
		f.lastTTL = r.URL.Query().Get("ttl")
		d, err := time.ParseDuration(f.lastTTL)
		body, _ := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.m[key] = fakeEntry{val: body, expires: f.clk.now().Add(d)}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (f *fakeService) has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.m[key]
	return ok
}

func setup(t *testing.T) (*Cache, *fakeService, *clock) {
	t.Helper()
	clk := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	fake := &fakeService{clk: clk, m: map[string]fakeEntry{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	r, err := newRemote(srv.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return newCache(r, clk.now), fake, clk
}

func put(t *testing.T, c *Cache, key, val string) {
	t.Helper()
	c.Put(context.Background(), key, []byte(val), ttl)
}

func get(t *testing.T, c *Cache, key string) (string, Outcome) {
	t.Helper()
	v, out := c.Get(context.Background(), key)
	return string(v), out
}

func TestAStoredValueIsServedByTheRedisService(t *testing.T) {
	c, fake, _ := setup(t)
	put(t, c, "k", "v")
	if v, out := get(t, c, "k"); v != "v" || out != Hit {
		t.Fatalf("got %q, %v", v, out)
	}
	if fake.lastTTL != "1m0s" {
		t.Fatalf("stored with ttl %q, want 1m0s", fake.lastTTL)
	}
}

func TestEntryExpiresAfterTTL(t *testing.T) {
	c, _, clk := setup(t)
	put(t, c, "k", "v")
	clk.advance(ttl - time.Second)
	if _, out := get(t, c, "k"); out != Hit {
		t.Fatalf("before expiry: %v, want Hit", out)
	}
	clk.advance(2 * time.Second)
	if _, out := get(t, c, "k"); out != Miss {
		t.Fatalf("after expiry: %v, want Miss", out)
	}
}

func TestKeysAreEscapedInThePath(t *testing.T) {
	c, fake, _ := setup(t)
	put(t, c, "analytics:v1:a b/c", "v")
	if !fake.has("analytics:v1:a%20b%2Fc") {
		t.Fatalf("the key was not escaped; the service holds %v", fake.m)
	}
}

// The outage cases: the redis service answers 503.

func TestOutageServesTheInMemoryCopyUntilItExpires(t *testing.T) {
	c, fake, clk := setup(t)
	put(t, c, "k", "v") // stored remotely and in memory

	fake.status.Store(http.StatusServiceUnavailable)
	clk.advance(30 * time.Second)
	if v, out := get(t, c, "k"); v != "v" || out != Fallback {
		t.Fatalf("got %q, %v", v, out)
	}
	clk.advance(31 * time.Second) // 61 s after the entry was stored
	if _, out := get(t, c, "k"); out != Unavailable {
		t.Fatalf("after the copy expired: %v, want Unavailable", out)
	}
}

func TestAnUnreachableServiceIsLeftAloneForAWhile(t *testing.T) {
	c, fake, clk := setup(t)
	fake.status.Store(http.StatusServiceUnavailable)

	get(t, c, "a")
	before := fake.gets.Load() + fake.puts.Load()
	get(t, c, "b")
	put(t, c, "c", "v")
	if after := fake.gets.Load() + fake.puts.Load(); after != before {
		t.Fatalf("the service was called %d more times while marked down", after-before)
	}

	fake.status.Store(0)
	clk.advance(retryAfter + time.Second)
	put(t, c, "d", "v")
	if !fake.has("d") {
		t.Fatal("the service was not used again after the retry delay")
	}
}

func TestAServiceMissDropsTheStaleCopy(t *testing.T) {
	c, fake, _ := setup(t)
	put(t, c, "k", "v")

	fake.mu.Lock()
	delete(fake.m, "k") // evicted remotely
	fake.mu.Unlock()
	if _, out := get(t, c, "k"); out != Miss {
		t.Fatalf("outcome = %v, want Miss (a healthy service saying 'gone' wins over memory)", out)
	}
}

func TestBadResponsesCountAsAnOutage(t *testing.T) {
	for name, bad := range map[string]string{"non-numeric ttl header": "soon", "negative ttl header": "-5"} {
		t.Run(name, func(t *testing.T) {
			c, fake, _ := setup(t)
			put(t, c, "k", "v")
			fake.badTTL = bad
			if v, out := get(t, c, "k"); v != "v" || out != Fallback {
				t.Fatalf("got %q, %v, want the copy", v, out)
			}
			if c.available() {
				t.Fatal("a malformed answer should mark the service down")
			}
		})
	}
}

func TestTransportErrorsAreHandled(t *testing.T) {
	r, _ := newRemote("http://127.0.0.1:1", 200*time.Millisecond) // nothing listens
	c := newCache(r, time.Now)
	if _, out := get(t, c, "k"); out != Unavailable {
		t.Fatalf("outcome = %v, want Unavailable", out)
	}
	put(t, c, "k", "v")
	if v, out := get(t, c, "k"); v != "v" || out != Fallback {
		t.Fatalf("got %q, %v", v, out)
	}
}

func TestEntriesWithoutAnExpiryGetADefault(t *testing.T) {
	c, fake, clk := setup(t)
	fake.m["config:x"] = fakeEntry{val: []byte(`"v"`)}
	if _, out := c.Get(context.Background(), "config:x"); out != Hit {
		t.Fatalf("outcome = %v, want Hit", out)
	}
	fake.status.Store(http.StatusServiceUnavailable)
	clk.advance(defaultTTL - time.Second)
	if _, out := c.Get(context.Background(), "config:x"); out != Fallback {
		t.Fatalf("inside the default: outcome = %v, want Fallback", out)
	}
	clk.advance(2 * time.Second)
	if _, out := c.Get(context.Background(), "config:x"); out != Unavailable {
		t.Fatalf("past the default expiry: outcome = %v, want Unavailable", out)
	}
}

func TestNewRejectsBadURLs(t *testing.T) {
	for _, u := range []string{"", "localhost:8082", "redis://localhost:6379", "http://"} {
		if _, err := New(u); err == nil {
			t.Errorf("New(%q) should fail", u)
		}
	}
	if _, err := New("http://redis-service:8082/"); err != nil {
		t.Errorf("valid URL: %v", err)
	}
}

func TestPutAgainstAnUnavailableServiceKeepsTheCopy(t *testing.T) {
	c, fake, _ := setup(t)
	fake.status.Store(http.StatusInternalServerError)
	c.Put(context.Background(), "k", []byte("v"), ttl)
	if v, ok := c.mem.get("k"); !ok || string(v) != "v" {
		t.Fatalf("memory = %q, %v", v, ok)
	}
}

func TestGetReportsAMissOnAHealthyService(t *testing.T) {
	c, _, _ := setup(t)
	if _, out := c.Get(context.Background(), "nope"); out != Miss {
		t.Fatalf("outcome = %v, want Miss", out)
	}
}

func TestAMissOnAHealthyServiceDropsTheCopyEvenIfTheServiceFailsLater(t *testing.T) {
	c, fake, _ := setup(t)
	ctx := context.Background()
	c.Put(ctx, "k", []byte("v"), ttl)

	fake.mu.Lock()
	delete(fake.m, "k") // gone from the service
	fake.mu.Unlock()
	if _, out := c.Get(ctx, "k"); out != Miss {
		t.Fatalf("outcome = %v, want Miss", out)
	}

	fake.status.Store(http.StatusServiceUnavailable)
	if v, out := c.Get(ctx, "k"); out != Unavailable {
		t.Fatalf("after the outage: outcome = %v (%q), want Unavailable: the copy must not outlive a miss", out, v)
	}
}
