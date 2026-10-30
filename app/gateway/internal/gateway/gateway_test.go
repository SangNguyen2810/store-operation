package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// newGateway fronts upstream with the gateway and returns its address.
func newGateway(t *testing.T, upstream http.Handler, o Options) *httptest.Server {
	t.Helper()
	up := httptest.NewServer(upstream)
	t.Cleanup(up.Close)
	o.Upstream = mustURL(t, up.URL)
	if o.DialTimeout == 0 {
		o.DialTimeout = time.Second
	}
	if o.HeaderTimeout == 0 {
		o.HeaderTimeout = 2 * time.Second
	}
	gw := httptest.NewServer(New(o))
	t.Cleanup(gw.Close)
	return gw
}

func get(t *testing.T, url string, header ...string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func body(t *testing.T, r *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func problem(t *testing.T, r *http.Response) (status int, detail string) {
	t.Helper()
	if ct := r.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content type %q", ct)
	}
	var p struct {
		Title  string
		Status int
		Detail string
	}
	if err := json.Unmarshal([]byte(body(t, r)), &p); err != nil {
		t.Fatal(err)
	}
	if p.Title != http.StatusText(p.Status) {
		t.Errorf("title %q does not match status %d", p.Title, p.Status)
	}
	return p.Status, p.Detail
}

func TestRoutesAndPassesResponsesThroughUntouched(t *testing.T) {
	gw := newGateway(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/kpis":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Upstream", "yes")
			w.WriteHeader(http.StatusAccepted)
			io.WriteString(w, `{"q":"`+r.URL.RawQuery+`"}`+"\n")
		case "/healthz":
			io.WriteString(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}), Options{})

	resp := get(t, gw.URL+"/api/kpis?store=1&from=2022-07-01")
	if resp.StatusCode != http.StatusAccepted || resp.Header.Get("X-Upstream") != "yes" || resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("status %d headers %v", resp.StatusCode, resp.Header)
	}
	if got := body(t, resp); got != `{"q":"store=1&from=2022-07-01"}`+"\n" {
		t.Errorf("body %q", got)
	}
	if got := body(t, get(t, gw.URL+"/healthz")); got != "ok" {
		t.Errorf("healthz %q", got)
	}
}

func TestUpstreamErrorStatusIsForwarded(t *testing.T) {
	gw := newGateway(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"title":"Bad Request","status":400}`)
	}), Options{})
	resp := get(t, gw.URL+"/api/kpis")
	if resp.StatusCode != 400 || !strings.Contains(body(t, resp), `"status":400`) {
		t.Errorf("a 400 from the API must reach the client as is, got %d", resp.StatusCode)
	}
}

func TestUnroutedPathIsNotForwarded(t *testing.T) {
	hit := false
	gw := newGateway(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }), Options{})
	if resp := get(t, gw.URL+"/admin"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("status %d", resp.StatusCode)
	}
	if hit {
		t.Error("an unrouted path reached the upstream")
	}
}

func TestRequestIDIsGeneratedAndSentUpstream(t *testing.T) {
	var seen string
	gw := newGateway(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-Request-Id")
	}), Options{})
	resp := get(t, gw.URL+"/api/meta")
	id := resp.Header.Get("X-Request-Id")
	if len(id) != 16 || id != seen {
		t.Errorf("response id %q, upstream saw %q", id, seen)
	}
	if other := get(t, gw.URL+"/api/meta").Header.Get("X-Request-Id"); other == id {
		t.Error("two requests got the same id")
	}
}

func TestRequestIDFromClient(t *testing.T) {
	var seen string
	gw := newGateway(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.Header.Get("X-Request-Id") }), Options{})

	if resp := get(t, gw.URL+"/api/meta", "X-Request-Id", "trace-42.a_b"); resp.Header.Get("X-Request-Id") != "trace-42.a_b" || seen != "trace-42.a_b" {
		t.Errorf("a safe client id should be kept: response %q upstream %q", resp.Header.Get("X-Request-Id"), seen)
	}
	for _, bad := range []string{"has space", "semi;colon", strings.Repeat("a", 65), "<script>"} {
		if resp := get(t, gw.URL+"/api/meta", "X-Request-Id", bad); resp.Header.Get("X-Request-Id") == bad || len(resp.Header.Get("X-Request-Id")) != 16 {
			t.Errorf("unsafe id %q was kept: %q", bad, resp.Header.Get("X-Request-Id"))
		}
	}
}

func TestAccessLog(t *testing.T) {
	buf := captureLogs(t)
	gw := newGateway(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}), Options{})
	resp := get(t, gw.URL+"/api/kpis?store=1")
	id := resp.Header.Get("X-Request-Id")
	out := buf.String()
	for _, want := range []string{"method=GET", "path=/api/kpis", `query="store=1"`, "status=418", "dur=", "ip=127.0.0.1", "request_id=" + id} {
		if !strings.Contains(out, want) {
			t.Errorf("log %q missing %q", out, want)
		}
	}
	if n := strings.Count(out, `msg=request`); n != 1 {
		t.Errorf("%d access lines for one request", n)
	}
}

func TestAccessLogDefaultsToOK(t *testing.T) {
	buf := captureLogs(t)
	gw := newGateway(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "x") }), Options{})
	get(t, gw.URL+"/healthz")
	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("log %q", buf)
	}
}

func TestRateLimit(t *testing.T) {
	now := time.Date(2022, 7, 27, 12, 0, 0, 0, time.UTC)
	buf := captureLogs(t)
	gw := newGateway(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}),
		Options{RatePerSec: 1, Burst: 2, Now: func() time.Time { return now }})

	for i := 0; i < 2; i++ {
		if resp := get(t, gw.URL+"/api/meta"); resp.StatusCode != 200 {
			t.Fatalf("request %d within the burst: %d", i+1, resp.StatusCode)
		}
	}
	resp := get(t, gw.URL+"/api/meta")
	if resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") != "1" {
		t.Fatalf("status %d Retry-After %q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
	if status, _ := problem(t, resp); status != 429 {
		t.Errorf("problem status %d", status)
	}
	if resp.Header.Get("X-Request-Id") == "" {
		t.Error("a rejected request still gets an id")
	}
	if !strings.Contains(buf.String(), "status=429") {
		t.Errorf("a rejected request must be logged: %s", buf)
	}

	now = now.Add(time.Second)
	if resp := get(t, gw.URL+"/api/meta"); resp.StatusCode != 200 {
		t.Errorf("after a second the bucket should have a token again: %d", resp.StatusCode)
	}
}

func TestRateLimitOffWhenRateIsZero(t *testing.T) {
	gw := newGateway(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Options{Burst: 1})
	for i := 0; i < 20; i++ {
		if resp := get(t, gw.URL+"/api/meta"); resp.StatusCode != 200 {
			t.Fatalf("request %d: %d", i+1, resp.StatusCode)
		}
	}
}

func TestSlowUpstreamIs504(t *testing.T) {
	buf := captureLogs(t)
	release := make(chan struct{})
	gw := newGateway(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }),
		Options{HeaderTimeout: 50 * time.Millisecond})
	t.Cleanup(func() { close(release) })

	resp := get(t, gw.URL+"/api/kpis")
	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if status, detail := problem(t, resp); status != 504 || detail != "upstream timed out" {
		t.Errorf("problem %d %q", status, detail)
	}
	if !strings.Contains(buf.String(), "upstream failed") {
		t.Errorf("failure not logged: %s", buf)
	}
}

func TestDeadUpstreamIs502(t *testing.T) {
	up := httptest.NewServer(http.NotFoundHandler())
	dead := up.URL
	up.Close() // nothing listens there any more

	gw := httptest.NewServer(New(Options{Upstream: mustURL(t, dead), DialTimeout: time.Second, HeaderTimeout: time.Second}))
	t.Cleanup(gw.Close)

	resp := get(t, gw.URL+"/api/kpis")
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if status, detail := problem(t, resp); status != 502 || detail != "upstream unavailable" {
		t.Errorf("problem %d %q", status, detail)
	}
}

func TestClientGoneIsNotAnError(t *testing.T) {
	captureLogs(t)
	rec := httptest.NewRecorder()
	upstreamError(rec, httptest.NewRequest(http.MethodGet, "/api/kpis", nil), http.ErrAbortHandler)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("unknown error should be 502, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	upstreamError(rec, httptest.NewRequest(http.MethodGet, "/api/kpis", nil), context.Canceled)
	if rec.Body.Len() != 0 {
		t.Errorf("a cancelled request got a body: %s", rec.Body)
	}
}
