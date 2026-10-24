package httpx_test

import (
	"context"
	"errors"
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
