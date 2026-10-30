package gateway

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// newProxy forwards to the single upstream. A slow upstream becomes a 504 and
// an unreachable one a 502, as problem responses, so the browser never sees a
// bare connection error.
func newProxy(upstream *url.URL, dial, header time.Duration) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(upstream)
			pr.SetXForwarded()
		},
		Transport: &http.Transport{
			DialContext:           (&net.Dialer{Timeout: dial}).DialContext,
			ResponseHeaderTimeout: header,
			MaxIdleConnsPerHost:   32,
			IdleConnTimeout:       90 * time.Second,
		},
		// The error handler logs; keep net/http from also writing to stderr.
		ErrorLog:     log.New(io.Discard, "", 0),
		ErrorHandler: upstreamError,
	}
}

func upstreamError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		return // the client went away; nobody to answer
	}
	slog.Warn("upstream failed", "path", r.URL.Path, "request_id", r.Header.Get(requestIDHeader), "err", err)
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		writeProblem(w, http.StatusGatewayTimeout, "upstream timed out")
		return
	}
	writeProblem(w, http.StatusBadGateway, "upstream unavailable")
}
