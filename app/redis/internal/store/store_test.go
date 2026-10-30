package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func newStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb, err := Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Close() })
	return New(rdb), mr
}

func TestSetThenGet(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	if err := s.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	val, ttl, found, err := s.Get(ctx, "k")
	if err != nil || !found || string(val) != "v" {
		t.Fatalf("val %q found %v err %v", val, found, err)
	}
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("ttl = %v, want up to a minute", ttl)
	}
}

func TestKeyWithoutExpiryHasZeroTTL(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	if err := s.Set(ctx, "config:x", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	if _, ttl, found, _ := s.Get(ctx, "config:x"); !found || ttl != 0 {
		t.Errorf("found %v ttl %v, want a key that never expires", found, ttl)
	}
}

func TestMissAndExpiry(t *testing.T) {
	s, mr := newStore(t)
	ctx := context.Background()
	if _, _, found, err := s.Get(ctx, "nope"); found || err != nil {
		t.Errorf("found %v err %v for a missing key", found, err)
	}
	_ = s.Set(ctx, "k", []byte("v"), time.Second)
	mr.FastForward(2 * time.Second)
	if _, _, found, _ := s.Get(ctx, "k"); found {
		t.Error("an expired key was returned")
	}
}

func TestStatsCountEachOutcome(t *testing.T) {
	s, mr := newStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "k", []byte("v"), time.Minute)
	s.Get(ctx, "k")
	s.Get(ctx, "k")
	s.Get(ctx, "other")
	mr.Close()
	s.Get(ctx, "k")
	_ = s.Set(ctx, "k", []byte("v"), time.Minute)
	if got, want := s.Stats(), (Stats{Hits: 2, Misses: 1, Sets: 1, Errors: 2}); got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

func TestPing(t *testing.T) {
	s, mr := newStore(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("ping: %v", err)
	}
	mr.Close()
	if err := s.Ping(context.Background()); err == nil {
		t.Error("ping should fail when Redis is down")
	}
}

func TestOpenRejectsABadURL(t *testing.T) {
	if _, err := Open(context.Background(), "http://not-redis"); err == nil {
		t.Error("want an error for a URL that is not redis://")
	}
}

func TestOpenDoesNotRequireRedisToBeUp(t *testing.T) {
	rdb, err := Open(context.Background(), "redis://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer rdb.Close()
	if err := New(rdb).Ping(context.Background()); err == nil {
		t.Error("ping to a closed port should fail")
	}
}

func TestRemaining(t *testing.T) {
	for _, c := range []struct {
		name    string
		pttl    time.Duration
		ttl     time.Duration
		expired bool
	}{
		{"time left", 1500 * time.Millisecond, 1500 * time.Millisecond, false},
		{"never expires", time.Duration(-1), 0, false},
		{"expired in between", time.Duration(-2), 0, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if ttl, expired := remaining(c.pttl); ttl != c.ttl || expired != c.expired {
				t.Errorf("remaining(%v) = %v, %v; want %v, %v", c.pttl, ttl, expired, c.ttl, c.expired)
			}
		})
	}
}

func TestOpenUsesShortTimeoutsAndNoRetries(t *testing.T) {
	rdb, err := Open(context.Background(), "redis://127.0.0.1:6379")
	if err != nil {
		t.Fatal(err)
	}
	defer rdb.Close()
	o := rdb.Options()
	// Open asks for -1; go-redis turns that into 0 once the client is built.
	if o.MaxRetries != 0 {
		t.Errorf("MaxRetries = %d, want 0 (a dead Redis must not be retried)", o.MaxRetries)
	}
	for name, d := range map[string]time.Duration{"dial": o.DialTimeout, "read": o.ReadTimeout, "write": o.WriteTimeout} {
		if d != opTimeout {
			t.Errorf("%s timeout = %v, want %v", name, d, opTimeout)
		}
	}
}
