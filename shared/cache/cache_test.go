package cache

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const ttl = time.Minute

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

func setup(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: time.Second})
	t.Cleanup(func() { rc.Close() })
	return New(rc, ttl), mr
}

func TestSecondCallIsServedFromRedis(t *testing.T) {
	c, mr := setup(t)
	l := &loader{val: payload{Name: "a", Price: 1234.56}}
	ctx := context.Background()

	first, err := Fetch(ctx, c, "k", l.load)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fetch(ctx, c, "k", l.load)
	if err != nil {
		t.Fatal(err)
	}

	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1", n)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cached %+v differs from fresh %+v", second, first)
	}
	if got := mr.TTL("k"); got != ttl {
		t.Fatalf("entry TTL = %v, want %v", got, ttl)
	}
}

func TestEntryExpiresAfterTTL(t *testing.T) {
	c, mr := setup(t)
	l := &loader{}
	ctx := context.Background()

	Fetch(ctx, c, "k", l.load)
	mr.FastForward(ttl - time.Second)
	Fetch(ctx, c, "k", l.load)
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times before expiry, want 1", n)
	}

	mr.FastForward(2 * time.Second)
	Fetch(ctx, c, "k", l.load)
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times after expiry, want 2", n)
	}
}

func TestDifferentKeysAreSeparateEntries(t *testing.T) {
	c, _ := setup(t)
	l := &loader{}
	ctx := context.Background()

	for _, k := range []string{"a", "b", "a", "b"} {
		Fetch(ctx, c, k, l.load)
	}
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times for 2 keys, want 2", n)
	}
}

func TestSlicesKeepNilAndEmpty(t *testing.T) {
	c, _ := setup(t)
	ctx := context.Background()

	for name, want := range map[string][]int{"empty": {}, "nil": nil} {
		load := func(context.Context) ([]int, error) { return want, nil }
		Fetch(ctx, c, name, load)
		got, _ := Fetch(ctx, c, name, load)
		if (got == nil) != (want == nil) || len(got) != 0 {
			t.Fatalf("%s slice came back as %#v", name, got)
		}
	}
}

func TestConcurrentColdRequestsLoadOnce(t *testing.T) {
	c, _ := setup(t)
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
			got[i], _ = Fetch(context.Background(), c, "k", l.load)
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

func TestRedisDownStillServesData(t *testing.T) {
	c, mr := setup(t)
	l := &loader{val: payload{Name: "live"}}
	mr.Close()

	for range 2 {
		got, err := Fetch(context.Background(), c, "k", l.load)
		if err != nil {
			t.Fatalf("with Redis down: %v", err)
		}
		if got.Name != "live" {
			t.Fatalf("wrong data with Redis down: %+v", got)
		}
	}
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times with Redis down, want 2 (nothing cached)", n)
	}
}

func TestErrorsAreReturnedAndNotCached(t *testing.T) {
	c, _ := setup(t)
	l := &loader{}
	ctx := context.Background()
	boom := errors.New("boom")

	l.err = boom
	if _, err := Fetch(ctx, c, "k", l.load); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}

	l.err = nil
	if _, err := Fetch(ctx, c, "k", l.load); err != nil {
		t.Fatalf("after recovery: %v", err)
	}
	if n := l.calls.Load(); n != 2 {
		t.Fatalf("loaded %d times, want 2 (the error must not be cached)", n)
	}
}

func TestUnreadableEntryIsReplaced(t *testing.T) {
	c, mr := setup(t)
	l := &loader{val: payload{Name: "fresh"}}
	ctx := context.Background()
	if err := mr.Set("k", "not json"); err != nil {
		t.Fatal(err)
	}

	got, err := Fetch(ctx, c, "k", l.load)
	if err != nil || got.Name != "fresh" {
		t.Fatalf("got %+v, %v", got, err)
	}
	Fetch(ctx, c, "k", l.load)
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1 (the bad entry should be overwritten)", n)
	}
}

func TestCallerHangingUpDoesNotLoseTheResult(t *testing.T) {
	c, mr := setup(t)
	l := &loader{delay: 200 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := Fetch(ctx, c, "k", l.load); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for !mr.Exists("k") {
		if time.Now().After(deadline) {
			t.Fatal("the shared load never reached the cache")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := Fetch(context.Background(), c, "k", l.load); err != nil {
		t.Fatal(err)
	}
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("loaded %d times, want 1", n)
	}
}
