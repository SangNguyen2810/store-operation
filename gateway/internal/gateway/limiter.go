package gateway

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// clientIP is the connection's remote address. X-Forwarded-For is ignored on
// purpose: nothing trusted sits in front of the gateway, so the header would
// only let a client pick its own bucket.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type bucket struct {
	tokens float64
	last   time.Time
}

// limiter is a token bucket per client IP.
type limiter struct {
	rate  float64 // tokens added per second
	burst float64
	now   func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

func newLimiter(ratePerSec float64, burst int, now func() time.Time) *limiter {
	return &limiter{rate: ratePerSec, burst: float64(burst), now: now, buckets: map[string]*bucket{}, lastSweep: now()}
}

// sweepEvery is how often idle buckets are dropped.
const sweepEvery = time.Minute

// allow takes a token for ip. When none is left it reports how long until one is.
func (l *limiter) allow(ip string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) >= sweepEvery {
		l.sweep(now)
	}
	b, found := l.buckets[ip]
	if !found {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}

// sweep drops buckets that have refilled completely: a missing bucket behaves
// exactly like a full one, so memory follows the clients active in the last
// burst/rate seconds rather than every client ever seen.
func (l *limiter) sweep(now time.Time) {
	for ip, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, ip)
		}
	}
	l.lastSweep = now
}

func (l *limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// middleware answers 429 with Retry-After once a client is over its limit.
func (l *limiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.allow(clientIP(r))
		if !ok {
			secs := int(math.Ceil(wait.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(max(secs, 1)))
			writeProblem(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}
