// Package gateway is the edge in front of the API: routing, request ids, an
// access log, a per-client rate limit and upstream timeouts. It knows nothing
// about what the API does.
package gateway

import (
	"net/http"
	"net/url"
	"time"
)

type Options struct {
	Upstream      *url.URL
	RatePerSec    float64 // 0 turns the rate limit off
	Burst         int
	DialTimeout   time.Duration
	HeaderTimeout time.Duration
	// Now is the clock the limiter uses; nil means time.Now.
	Now func() time.Time
}

// New returns the gateway handler. Order, outermost first: request id, access
// log, rate limit, routes. The limit sits inside the log so rejected requests
// are logged too.
func New(o Options) http.Handler {
	proxy := newProxy(o.Upstream, o.DialTimeout, o.HeaderTimeout)
	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.Handle("/healthz", proxy)

	var h http.Handler = mux
	if o.RatePerSec > 0 {
		now := o.Now
		if now == nil {
			now = time.Now
		}
		h = newLimiter(o.RatePerSec, o.Burst, now).middleware(h)
	}
	return withRequestID(withAccessLog(h))
}
