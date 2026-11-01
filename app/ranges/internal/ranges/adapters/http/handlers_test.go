package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/domain"
)

type fakeSvc struct{ err error }

func (f fakeSvc) Ranges(context.Context) ([]domain.Range, error) {
	if f.err != nil {
		return nil, f.err
	}
	return domain.ResolveRanges(domain.DefaultRangeDefs(), time.Date(2022, 7, 27, 0, 0, 0, 0, time.UTC))
}

func do(svc Ranges, method string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	Register(mux, svc)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, "/api/presets", nil))
	return rec
}

func TestPresetsResponse(t *testing.T) {
	rec := do(fakeSvc{}, http.MethodGet)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 18 {
		t.Fatalf("%d ranges, want 18", len(got))
	}
	want := `{"default":true,"from":"2022-06-28","group":"Recent","id":"last-30","label":"Last 30 days","to":"2022-07-27"}`
	if b, _ := json.Marshal(got[2]); string(b) != want {
		t.Errorf("third range\n got %s\nwant %s", b, want)
	}
	defaults := 0
	for _, r := range got {
		if r["default"] == true {
			defaults++
		}
	}
	if defaults != 1 {
		t.Errorf("%d defaults, want 1", defaults)
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
