package cache

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/platform/cache"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

// fakeInner stands in for the Postgres adapter and counts how often it is asked.
type fakeInner struct {
	calls  atomic.Int32
	stores []domain.Store
	err    error
}

func (f *fakeInner) Stores(context.Context) ([]domain.Store, error) {
	f.calls.Add(1)
	return f.stores, f.err
}

// fakeService is the redis service's HTTP contract over a map.
type fakeService struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (f *fakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	k := strings.TrimPrefix(r.URL.EscapedPath(), "/v1/cache/")
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		v, ok := f.m[k]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(v)
	case http.MethodPut:
		f.m[k], _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}
}

func setup(t *testing.T, inner *fakeInner) (*Repo, *fakeService) {
	t.Helper()
	fake := &fakeService{m: map[string][]byte{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	c, err := cache.New(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return New(c, inner), fake
}

func TestStoresAreCachedForAnHour(t *testing.T) {
	inner := &fakeInner{stores: []domain.Store{{ID: 1, Label: "A"}}}
	repo, fake := setup(t, inner)

	first, err := repo.Stores(context.Background())
	if err != nil || len(first) != 1 {
		t.Fatalf("got %+v, %v", first, err)
	}
	second, _ := repo.Stores(context.Background())
	if inner.calls.Load() != 1 || !reflect.DeepEqual(first, second) {
		t.Fatalf("queried %d times; fresh %+v, cached %+v", inner.calls.Load(), first, second)
	}
	if _, ok := fake.m["analytics:v1:stores"]; !ok {
		t.Fatalf("expected the stores key, have %v", fake.m)
	}
	if TTL != time.Hour {
		t.Errorf("TTL = %v, want 1h", TTL)
	}
}

func TestStoreDetailsSurviveTheRoundTrip(t *testing.T) {
	want := domain.Store{
		ID: 2, Label: "Hamilton, New Zealand", Manager: "Lavone O'Reilly", Phone: "657282285970",
		Address: "478 Joliet Way", District: "Hamilton", PostalCode: "77948", City: "Hamilton", Country: "New Zealand",
		InventoryItems: 2311, ActiveCustomers: 273,
	}
	repo, _ := setup(t, &fakeInner{stores: []domain.Store{want}})
	repo.Stores(context.Background())
	got, _ := repo.Stores(context.Background())
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestErrorsAreNotCached(t *testing.T) {
	boom := errors.New("boom")
	inner := &fakeInner{err: boom}
	repo, _ := setup(t, inner)
	if _, err := repo.Stores(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	inner.err, inner.stores = nil, []domain.Store{{ID: 1}}
	if s, err := repo.Stores(context.Background()); err != nil || len(s) != 1 {
		t.Fatalf("after recovery: %v, %v", s, err)
	}
}
