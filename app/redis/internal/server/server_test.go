package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/SangNguyen2810/store-operation/app/redis/internal/store"
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

func newServer(t *testing.T) (http.Handler, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb, err := store.Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Close() })
	return New(store.New(rdb)), mr
}

func do(h http.Handler, method, target, body string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func problem(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantDetail string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status %d, want %d (body %s)", rec.Code, wantStatus, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content type %q", ct)
	}
	var p struct {
		Title  string
		Status int
		Detail string
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Status != wantStatus || p.Title != http.StatusText(wantStatus) || !strings.Contains(p.Detail, wantDetail) {
		t.Errorf("problem = %+v, want %d containing %q", p, wantStatus, wantDetail)
	}
}

func TestPutThenGetReturnsTheSameBytes(t *testing.T) {
	h, _ := newServer(t)
	value := `{"revenue":12.5}` + "\n\x00binary"
	if rec := do(h, "PUT", "/v1/cache/analytics:v1:totals:a-b?ttl=10m", value); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("put: %d %q", rec.Code, rec.Body)
	}
	rec := do(h, "GET", "/v1/cache/analytics:v1:totals:a-b", "")
	if rec.Code != 200 || rec.Body.String() != value {
		t.Fatalf("get: %d %q", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("content type %q", ct)
	}
	if left, _ := strconv.Atoi(rec.Header().Get("X-Cache-Ttl")); left < 599 || left > 600 {
		t.Errorf("X-Cache-Ttl = %q, want about 600 seconds", rec.Header().Get("X-Cache-Ttl"))
	}
}

func TestPutOverwrites(t *testing.T) {
	h, _ := newServer(t)
	do(h, "PUT", "/v1/cache/k?ttl=1m", "old")
	do(h, "PUT", "/v1/cache/k?ttl=1m", "new")
	if got := do(h, "GET", "/v1/cache/k", "").Body.String(); got != "new" {
		t.Errorf("got %q", got)
	}
}

func TestMissingKeyIs404(t *testing.T) {
	h, _ := newServer(t)
	problem(t, do(h, "GET", "/v1/cache/nope", ""), 404, "no such key")
}

func TestExpiredKeyIs404(t *testing.T) {
	h, mr := newServer(t)
	do(h, "PUT", "/v1/cache/k?ttl=1s", "v")
	mr.FastForward(2 * time.Second)
	problem(t, do(h, "GET", "/v1/cache/k", ""), 404, "no such key")
}

func TestKeyRules(t *testing.T) {
	h, _ := newServer(t)
	for _, key := range []string{strings.Repeat("a", 201), "has%20space", "semi;colon", "q%3Fx", "caf%C3%A9"} {
		problem(t, do(h, "PUT", "/v1/cache/"+key+"?ttl=1m", "v"), 400, "key:")
		problem(t, do(h, "GET", "/v1/cache/"+key, ""), 400, "key:")
	}
	for _, key := range []string{"a", strings.Repeat("a", 200), "analytics:v1:kpis:2022-06-28:2022-07-27:all", "A.b_c-d"} {
		if rec := do(h, "PUT", "/v1/cache/"+key+"?ttl=1m", "v"); rec.Code != 204 {
			t.Errorf("key %q: %d %s", key, rec.Code, rec.Body)
		}
	}
}

func TestTTLRules(t *testing.T) {
	h, _ := newServer(t)
	for _, q := range []string{"", "?ttl=", "?ttl=abc", "?ttl=0s", "?ttl=-5m", "?ttl=25h", "?ttl=10"} {
		problem(t, do(h, "PUT", "/v1/cache/k"+q, "v"), 400, "ttl:")
	}
	for _, q := range []string{"?ttl=1s", "?ttl=24h", "?ttl=90m"} {
		if rec := do(h, "PUT", "/v1/cache/k"+q, "v"); rec.Code != 204 {
			t.Errorf("%s: %d %s", q, rec.Code, rec.Body)
		}
	}
}

func TestSettingsMayHaveNoExpiry(t *testing.T) {
	h, mr := newServer(t)
	if rec := do(h, "PUT", "/v1/cache/config:presets:v1", `[]`); rec.Code != 204 {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	mr.FastForward(1000 * time.Hour)
	rec := do(h, "GET", "/v1/cache/config:presets:v1", "")
	if rec.Code != 200 || rec.Header().Get("X-Cache-Ttl") != "" {
		t.Errorf("status %d, X-Cache-Ttl %q: a setting must stay and carry no ttl", rec.Code, rec.Header().Get("X-Cache-Ttl"))
	}
}

func TestBodyRules(t *testing.T) {
	h, _ := newServer(t)
	problem(t, do(h, "PUT", "/v1/cache/k?ttl=1m", ""), 400, "body:")
	problem(t, do(h, "PUT", "/v1/cache/k?ttl=1m", strings.Repeat("x", 1<<20+1)), 413, "1 MiB")
	if rec := do(h, "PUT", "/v1/cache/k?ttl=1m", strings.Repeat("x", 1<<20)); rec.Code != 204 {
		t.Errorf("a value of exactly 1 MiB: %d", rec.Code)
	}
}

func TestOtherMethodsAreRejected(t *testing.T) {
	h, _ := newServer(t)
	for _, m := range []string{"POST", "DELETE", "PATCH"} {
		if rec := do(h, m, "/v1/cache/k", "v"); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: %d", m, rec.Code)
		}
	}
}

func TestRedisDownIs503AndTheCauseIsNotLeaked(t *testing.T) {
	buf := captureLogs(t)
	h, mr := newServer(t)
	mr.Close()
	for _, rec := range []*httptest.ResponseRecorder{
		do(h, "GET", "/v1/cache/k", ""),
		do(h, "PUT", "/v1/cache/k?ttl=1m", "v"),
	} {
		problem(t, rec, 503, "cache backend unavailable")
		if strings.Contains(rec.Body.String(), "connect") || strings.Contains(rec.Body.String(), "127.0.0.1") {
			t.Errorf("the Redis error leaked: %s", rec.Body)
		}
	}
	if !strings.Contains(buf.String(), "redis failed") {
		t.Errorf("the failure should be logged: %s", buf)
	}
}

func TestHealth(t *testing.T) {
	h, mr := newServer(t)
	if rec := do(h, "GET", "/healthz", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("healthy: %d %s", rec.Code, rec.Body)
	}
	mr.Close()
	problem(t, do(h, "GET", "/healthz", ""), 503, "redis unreachable")
}

func TestStats(t *testing.T) {
	h, _ := newServer(t)
	do(h, "PUT", "/v1/cache/k?ttl=1m", "v")
	do(h, "GET", "/v1/cache/k", "")
	do(h, "GET", "/v1/cache/k", "")
	do(h, "GET", "/v1/cache/other", "")
	rec := do(h, "GET", "/v1/stats", "")
	var s store.Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if rec.Header().Get("Content-Type") != "application/json" || s != (store.Stats{Hits: 2, Misses: 1, Sets: 1}) {
		t.Errorf("stats %+v type %q", s, rec.Header().Get("Content-Type"))
	}
}

func TestRequestLog(t *testing.T) {
	buf := captureLogs(t)
	h, _ := newServer(t)
	do(h, "GET", "/v1/cache/nope", "", "X-Request-Id", "abc123")
	out := buf.String()
	for _, want := range []string{"method=GET", "path=/v1/cache/nope", "status=404", "dur=", "request_id=abc123"} {
		if !strings.Contains(out, want) {
			t.Errorf("log %q missing %q", out, want)
		}
	}
	buf.Reset()
	do(h, "GET", "/healthz", "")
	if strings.Contains(buf.String(), "request_id") {
		t.Errorf("no id was sent, so none should be logged: %s", buf)
	}
}
