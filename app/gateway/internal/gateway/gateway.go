// Package gateway is the edge in front of the API: routing, request ids, an
// access log, a per-client rate limit and upstream timeouts. It knows nothing
// about what the API does.
package gateway

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Route sends the requests under Prefix (an exact path and everything below
// it, such as /api/stores) to Upstream instead of the default one.
type Route struct {
	Prefix   string
	Upstream *url.URL
}

type Options struct {
	// Upstream takes /api/ and /healthz, and everything a Route does not claim.
	Upstream *url.URL
	// Routes are checked first; the longest matching prefix wins.
	Routes        []Route
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
	// A map, so a route for /healthz replaces the default instead of clashing with it.
	handlers := map[string]http.Handler{"/api/": proxy, "/healthz": proxy}
	for _, r := range o.Routes {
		p := newProxy(r.Upstream, o.DialTimeout, o.HeaderTimeout)
		handlers[r.Prefix] = p
		if !strings.HasSuffix(r.Prefix, "/") {
			handlers[r.Prefix+"/"] = p
		}
	}
	mux := http.NewServeMux()
	for pattern, h := range handlers {
		mux.Handle(pattern, h)
	}

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
