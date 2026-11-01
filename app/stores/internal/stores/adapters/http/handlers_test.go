package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

type fakeSvc struct{ err error }

func (f fakeSvc) Stores(context.Context) ([]domain.Store, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []domain.Store{{ID: 1, Label: "A"}}, nil
}

func do(svc Stores, method string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, svc)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, "/api/stores", nil))
	return rec
}

func TestStoresResponse(t *testing.T) {
	rec := do(fakeSvc{}, http.MethodGet)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	want := `[{"id":1,"label":"A","manager":"","phone":"","address":"","district":"","postalCode":"","city":"","country":"","inventoryItems":0,"activeCustomers":0}]`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s", got)
	}
}

func TestFailuresDoNotLeak(t *testing.T) {
	rec := do(fakeSvc{err: errors.New("secret")}, http.MethodGet)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("status %d body %s", rec.Code, rec.Body)
	}
}

func TestOnlyGetIsAllowed(t *testing.T) {
	if rec := do(fakeSvc{}, http.MethodPost); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: status %d", rec.Code)
	}
}
