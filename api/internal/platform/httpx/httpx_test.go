package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/api/internal/platform/httpx"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(h http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec
}

func TestHealth(t *testing.T) {
	if rec := get(httpx.Health(fakeDB{})); rec.Code != http.StatusOK {
		t.Errorf("healthy: %d", rec.Code)
	}
	if rec := get(httpx.Health(fakeDB{err: errors.New("down")})); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("unhealthy: %d", rec.Code)
	}
}

func TestMeta(t *testing.T) {
	rec := get(httpx.Meta(time.Date(2022, 7, 27, 0, 0, 0, 0, time.UTC)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"asOf":"2022-07-27"`) {
		t.Errorf("status %d body %s", rec.Code, rec.Body)
	}
}

func TestWriteProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteProblem(rec, http.StatusBadRequest, "nope")
	if rec.Code != 400 || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var p httpx.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Title != "Bad Request" || p.Status != 400 || p.Detail != "nope" {
		t.Errorf("problem = %+v", p)
	}
}

func TestFailHidesTheError(t *testing.T) {
	buf := captureLogs(t)
	rec := httptest.NewRecorder()
	httpx.Fail(rec, httptest.NewRequest(http.MethodGet, "/x", nil), errors.New("secret sql"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("leaked: %s", rec.Body)
	}
	if !strings.Contains(buf.String(), "secret sql") {
		t.Errorf("the error should still be logged: %s", buf)
	}
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestLogRequests(t *testing.T) {
	buf := captureLogs(t)
	h := httpx.LogRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/kpis?store=1", nil))
	if rec.Code != http.StatusTeapot {
		t.Errorf("handler response lost: %d", rec.Code)
	}
	out := buf.String()
	for _, want := range []string{"method=GET", "path=/api/kpis", `query="store=1"`, "dur="} {
		if !strings.Contains(out, want) {
			t.Errorf("log %q missing %q", out, want)
		}
	}
}

func TestLogRequestsCarriesTheRequestID(t *testing.T) {
	buf := captureLogs(t)
	h := httpx.LogRequests(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/api/kpis", nil)
	req.Header.Set("X-Request-Id", "abc123")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if !strings.Contains(buf.String(), "request_id=abc123") {
		t.Errorf("log %q missing the request id", buf)
	}

	buf.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/kpis", nil))
	if strings.Contains(buf.String(), "request_id") {
		t.Errorf("no id was sent, so none should be logged: %q", buf)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteJSON(rec, http.StatusCreated, map[string]int{"a": 1})
	if rec.Code != 201 || rec.Header().Get("Content-Type") != "application/json" || strings.TrimSpace(rec.Body.String()) != `{"a":1}` {
		t.Errorf("status %d type %q body %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body)
	}
}
