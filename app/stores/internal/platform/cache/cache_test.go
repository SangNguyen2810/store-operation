package cache

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
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

type payload struct {
	Name  string
	Price float64
	Tags  []string
}

// loader stands in for the real query and counts how often it runs.
type loader struct {
	calls atomic.Int32
	delay time.Duration
	err   error
	val   payload
}

func (l *loader) load(context.Context) (payload, error) {
	l.calls.Add(1)
	time.Sleep(l.delay)
	return l.val, l.err
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

func TestSecondCallIsServedFromTheRedisService(t *testing.T) {
	c, fake, _ := setup(t)
	l := &loader{val: payload{Name: "a", Price: 1234.56}}
	ctx := context.Background()

	first, err := Fetch(ctx, c, "k", ttl, l.load)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fetch(ctx, c, "k", ttl, l.load)
	if err != nil {
		t.Fatal(err)
	}

	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1", n)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cached %+v differs from fresh %+v", second, first)
	}
	if fake.lastTTL != "1m0s" {
		t.Fatalf("stored with ttl %q, want 1m0s", fake.lastTTL)
	}
}

func TestEntryExpiresAfterTTL(t *testing.T) {
	c, _, clk := setup(t)
	l := &loader{}
	ctx := context.Background()

	Fetch(ctx, c, "k", ttl, l.load)
	clk.advance(ttl - time.Second)
	Fetch(ctx, c, "k", ttl, l.load)
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times before expiry, want 1", n)
	}

	clk.advance(2 * time.Second)
	Fetch(ctx, c, "k", ttl, l.load)
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times after expiry, want 2", n)
	}
}

func TestDifferentKeysAreSeparateEntries(t *testing.T) {
	c, _, _ := setup(t)
	l := &loader{}
	ctx := context.Background()

	for _, k := range []string{"a", "b", "a", "b"} {
		Fetch(ctx, c, k, ttl, l.load)
	}
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times for 2 keys, want 2", n)
	}
}

func TestKeysAreEscapedInThePath(t *testing.T) {
	c, fake, _ := setup(t)
	Fetch(context.Background(), c, "analytics:v1:a b/c", ttl, (&loader{}).load)
	if !fake.has("analytics:v1:a%20b%2Fc") {
		t.Fatalf("the key was not escaped; the service holds %v", fake.m)
	}
}

func TestSlicesKeepNilAndEmpty(t *testing.T) {
	c, _, _ := setup(t)
	ctx := context.Background()

	for name, want := range map[string][]int{"empty": {}, "nil": nil} {
		load := func(context.Context) ([]int, error) { return want, nil }
		Fetch(ctx, c, name, ttl, load)
		got, _ := Fetch(ctx, c, name, ttl, load)
		if (got == nil) != (want == nil) || len(got) != 0 {
			t.Fatalf("%s slice came back as %#v", name, got)
		}
	}
}

func TestConcurrentColdRequestsLoadOnce(t *testing.T) {
	c, _, _ := setup(t)
	l := &loader{delay: 150 * time.Millisecond, val: payload{Name: "x"}}

	const callers = 20
	var wg sync.WaitGroup
	got := make([]payload, callers)
	start := make(chan struct{})
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got[i], _ = Fetch(context.Background(), c, "k", ttl, l.load)
		}()
	}
	close(start)
	wg.Wait()

	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times for %d concurrent callers, want 1", n, callers)
	}
	for _, g := range got {
		if g.Name != "x" {
			t.Fatalf("a caller saw %+v", g)
		}
	}
}

func TestErrorsAreReturnedAndNotCached(t *testing.T) {
	c, _, _ := setup(t)
	l := &loader{}
	ctx := context.Background()
	boom := errors.New("boom")

	l.err = boom
	if _, err := Fetch(ctx, c, "k", ttl, l.load); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}

	l.err = nil
	if _, err := Fetch(ctx, c, "k", ttl, l.load); err != nil {
		t.Fatalf("after recovery: %v", err)
	}
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times, want 2 (the error must not be cached)", n)
	}
}

func TestUnreadableEntryIsReplaced(t *testing.T) {
	c, fake, _ := setup(t)
	l := &loader{val: payload{Name: "fresh"}}
	ctx := context.Background()
	fake.m["k"] = fakeEntry{val: []byte("not json")}

	got, err := Fetch(ctx, c, "k", ttl, l.load)
	if err != nil || got.Name != "fresh" {
		t.Fatalf("got %+v, %v", got, err)
	}
	Fetch(ctx, c, "k", ttl, l.load)
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1 (the bad entry should be overwritten)", n)
	}
}

func TestCallerHangingUpDoesNotLoseTheResult(t *testing.T) {
	c, fake, _ := setup(t)
	l := &loader{delay: 200 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := Fetch(ctx, c, "k", ttl, l.load); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for !fake.has("k") {
		if time.Now().After(deadline) {
			t.Fatal("the shared load never reached the cache")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := Fetch(context.Background(), c, "k", ttl, l.load); err != nil {
		t.Fatal(err)
	}
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1", n)
	}
}

// The outage cases: the redis service answers 503.

func TestOutageServesTheInMemoryCopyUntilItExpires(t *testing.T) {
	c, fake, clk := setup(t)
	l := &loader{val: payload{Name: "live"}}
	ctx := context.Background()
	Fetch(ctx, c, "k", ttl, l.load) // stored remotely and in memory

	fake.status.Store(http.StatusServiceUnavailable)
	clk.advance(30 * time.Second)
	got, err := Fetch(ctx, c, "k", ttl, l.load)
	if err != nil || got.Name != "live" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times during the outage inside the TTL, want 1", n)
	}

	clk.advance(31 * time.Second) // 61 s after the entry was stored
	Fetch(ctx, c, "k", ttl, l.load)
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times after the copy expired, want 2", n)
	}
}

func TestOutageStillServesDataWhenNothingIsCached(t *testing.T) {
	c, fake, _ := setup(t)
	l := &loader{val: payload{Name: "live"}}
	fake.status.Store(http.StatusInternalServerError)

	for range 2 {
		got, err := Fetch(context.Background(), c, "k", ttl, l.load)
		if err != nil || got.Name != "live" {
			t.Fatalf("got %+v, %v", got, err)
		}
	}
	// The first call loaded and kept its answer in memory, so the second needs no query.
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1", n)
	}
}

func TestAnUnreachableServiceIsLeftAloneForAWhile(t *testing.T) {
	c, fake, clk := setup(t)
	fake.status.Store(http.StatusServiceUnavailable)
	l := &loader{}
	ctx := context.Background()

	Fetch(ctx, c, "a", ttl, l.load)
	before := fake.gets.Load() + fake.puts.Load()
	Fetch(ctx, c, "b", ttl, l.load)
	Fetch(ctx, c, "c", ttl, l.load)
	if after := fake.gets.Load() + fake.puts.Load(); after != before {
		t.Fatalf("the service was called %d more times while marked down", after-before)
	}

	fake.status.Store(0)
	clk.advance(retryAfter + time.Second)
	Fetch(ctx, c, "d", ttl, l.load)
	if !fake.has("d") {
		t.Fatal("the service was not used again after the retry delay")
	}
}

func TestRecoveryReturnsToTheService(t *testing.T) {
	c, fake, clk := setup(t)
	l := &loader{val: payload{Name: "v"}}
	ctx := context.Background()
	fake.status.Store(http.StatusServiceUnavailable)
	Fetch(ctx, c, "k", ttl, l.load)

	fake.status.Store(0)
	clk.advance(retryAfter + time.Second)
	Fetch(ctx, c, "k", ttl, l.load)
	if !fake.has("k") {
		t.Fatal("after recovery the answer must be stored in the service again")
	}
}

func TestAServiceMissDropsTheStaleCopy(t *testing.T) {
	c, fake, _ := setup(t)
	l := &loader{val: payload{Name: "v"}}
	ctx := context.Background()
	Fetch(ctx, c, "k", ttl, l.load)

	fake.mu.Lock()
	delete(fake.m, "k") // evicted remotely
	fake.mu.Unlock()
	Fetch(ctx, c, "k", ttl, l.load)
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times, want 2 (a healthy service saying 'gone' wins over memory)", n)
	}
}

func TestBadResponsesCountAsAnOutage(t *testing.T) {
	for name, set := range map[string]func(*fakeService){
		"non-numeric ttl header": func(f *fakeService) { f.badTTL = "soon" },
		"negative ttl header":    func(f *fakeService) { f.badTTL = "-5" },
	} {
		t.Run(name, func(t *testing.T) {
			c, fake, _ := setup(t)
			l := &loader{val: payload{Name: "v"}}
			Fetch(context.Background(), c, "k", ttl, l.load)
			set(fake)
			if _, err := Fetch(context.Background(), c, "k", ttl, l.load); err != nil {
				t.Fatal(err)
			}
			if c.available() {
				t.Fatal("a malformed answer should mark the service down")
			}
		})
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

func TestTransportErrorsAreHandled(t *testing.T) {
	r, _ := newRemote("http://127.0.0.1:1", 200*time.Millisecond) // nothing listens
	clk := &clock{t: time.Now()}
	c := newCache(r, clk.now)
	l := &loader{val: payload{Name: "v"}}
	got, err := Fetch(context.Background(), c, "k", ttl, l.load)
	if err != nil || got.Name != "v" {
		t.Fatalf("got %+v, %v", got, err)
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
