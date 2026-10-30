package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *clock { return &clock{t: time.Date(2022, 7, 27, 12, 0, 0, 0, time.UTC)} }

func TestLimiterBurstThenRefill(t *testing.T) {
	c := newClock()
	l := newLimiter(2, 3, c.now) // 2 tokens/s, burst 3
	for i := 0; i < 3; i++ {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("burst request %d denied", i+1)
		}
	}
	ok, wait := l.allow("a")
	if ok || wait != 500*time.Millisecond {
		t.Fatalf("ok=%v wait=%v, want denied for 500ms", ok, wait)
	}
	c.advance(500 * time.Millisecond)
	if ok, _ := l.allow("a"); !ok {
		t.Error("one token should have come back after 500ms")
	}
	if ok, _ := l.allow("a"); ok {
		t.Error("only one token came back")
	}
}

func TestLimiterNeverRefillsPastBurst(t *testing.T) {
	c := newClock()
	l := newLimiter(10, 2, c.now)
	l.allow("a")
	c.advance(time.Hour)
	allowed := 0
	for i := 0; i < 5; i++ {
		if ok, _ := l.allow("a"); ok {
			allowed++
		}
	}
	if allowed != 2 {
		t.Errorf("allowed %d after a long idle, want the burst of 2", allowed)
	}
}

func TestLimiterKeepsClientsApart(t *testing.T) {
	l := newLimiter(1, 1, newClock().now)
	if ok, _ := l.allow("a"); !ok {
		t.Fatal("a denied")
	}
	if ok, _ := l.allow("a"); ok {
		t.Error("a should be out of tokens")
	}
	if ok, _ := l.allow("b"); !ok {
		t.Error("b must not share a's bucket")
	}
}

func TestLimiterEvictsIdleClients(t *testing.T) {
	c := newClock()
	l := newLimiter(1, 10, c.now) // a bucket is full again after 10 s
	for _, ip := range []string{"a", "b", "c"} {
		l.allow(ip)
	}
	if l.size() != 3 {
		t.Fatalf("size %d", l.size())
	}

	// Before the sweep interval nothing is dropped, even if buckets are full.
	c.advance(30 * time.Second)
	l.allow("d")
	if l.size() != 4 {
		t.Errorf("swept too early: size %d", l.size())
	}

	// After it, the clients that refilled are gone and the fresh one stays.
	c.advance(31 * time.Second)
	l.allow("d")
	if l.size() != 1 {
		t.Errorf("size %d after the sweep, want only d", l.size())
	}
}

func TestLimiterKeepsClientsStillOverTheLimit(t *testing.T) {
	c := newClock()
	l := newLimiter(0.01, 1, c.now) // refills in 100 s
	l.allow("slow")
	c.advance(sweepEvery)
	l.allow("other")
	if _, found := l.buckets["slow"]; !found {
		t.Error("a bucket that has not refilled must survive the sweep")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.9:5555"
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	if got := clientIP(r); got != "203.0.113.9" {
		t.Errorf("clientIP = %q; X-Forwarded-For must be ignored", got)
	}
	r.RemoteAddr = "no-port"
	if got := clientIP(r); got != "no-port" {
		t.Errorf("clientIP = %q", got)
	}
}
