package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/api/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/api/internal/platform/testutil"
)

// These run against a real Redis, which miniredis cannot vouch for: real TTLs
// and the wire format of what is stored.

func TestRealRedisStoresJSONWithTTL(t *testing.T) {
	rc := testutil.Redis(t)
	key := testutil.Prefix(t, rc) + "k"
	c := cache.New(rc, time.Minute)
	ctx := context.Background()

	type row struct {
		A int
		B []string
	}
	want := row{A: 7, B: []string{"x"}}
	var loads atomic.Int32
	load := func(context.Context) (row, error) { loads.Add(1); return want, nil }

	for i := 0; i < 3; i++ {
		got, err := cache.Fetch(ctx, c, key, load)
		if err != nil || got.A != 7 || len(got.B) != 1 {
			t.Fatalf("call %d: %+v, %v", i, got, err)
		}
	}
	if loads.Load() != 1 {
		t.Errorf("loaded %d times, want 1", loads.Load())
	}
	raw, err := rc.Get(ctx, key).Result()
	if err != nil || raw != `{"A":7,"B":["x"]}` {
		t.Errorf("stored %q, %v", raw, err)
	}
	if ttl := rc.PTTL(ctx, key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("ttl = %s, want within (0, 1m]", ttl)
	}
}

func TestRealRedisExpiry(t *testing.T) {
	rc := testutil.Redis(t)
	key := testutil.Prefix(t, rc) + "k"
	c := cache.New(rc, 200*time.Millisecond)
	var loads atomic.Int32
	load := func(context.Context) (int, error) { return int(loads.Add(1)), nil }

	if v, _ := cache.Fetch(context.Background(), c, key, load); v != 1 {
		t.Fatalf("first = %d", v)
	}
	time.Sleep(400 * time.Millisecond)
	if v, _ := cache.Fetch(context.Background(), c, key, load); v != 2 {
		t.Errorf("after expiry = %d, want a fresh load (2)", v)
	}
}

func TestRealRedisCoalescesAndDoesNotCacheErrors(t *testing.T) {
	rc := testutil.Redis(t)
	prefix := testutil.Prefix(t, rc)
	c := cache.New(rc, time.Minute)
	ctx := context.Background()

	var loads atomic.Int32
	slow := func(context.Context) (int, error) { loads.Add(1); time.Sleep(100 * time.Millisecond); return 5, nil }
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = cache.Fetch(ctx, c, prefix+"slow", slow) }()
	}
	wg.Wait()
	if loads.Load() != 1 {
		t.Errorf("10 concurrent callers ran %d loads, want 1", loads.Load())
	}

	boom := errors.New("boom")
	if _, err := cache.Fetch(ctx, c, prefix+"err", func(context.Context) (int, error) { return 0, boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if n, _ := rc.Exists(ctx, prefix+"err").Result(); n != 0 {
		t.Error("an error was cached")
	}
}
