package ranges

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
	platformcache "github.com/SangNguyen2810/store-operation/app/api/internal/platform/cache"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// service is the redis service's contract over one stored value.
type service struct {
	mu   sync.Mutex
	val  []byte
	down atomic.Bool
	gets atomic.Int32
}

func (s *service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.down.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path != "/v1/cache/"+Key {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.gets.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.val == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, _ = w.Write(s.val)
}

func (s *service) store(v string) {
	s.mu.Lock()
	s.val = []byte(v)
	s.mu.Unlock()
}

func newWithCache(t *testing.T) (*platformcache.Cache, *service) {
	t.Helper()
	svc := &service{}
	srv := httptest.NewServer(svc)
	t.Cleanup(srv.Close)
	c, err := platformcache.New(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return c, svc
}

func setup(t *testing.T) (*Source, *service) {
	t.Helper()
	c, svc := newWithCache(t)
	return New(c), svc
}

const custom = `[
  {"id":"week","label":"This week","group":"Mine","kind":"days","days":7,"default":true},
  {"id":"h1","label":"First half","group":"Mine","kind":"fixed","from":"2022-01-01","to":"2022-06-30"},
  {"id":"mtd","label":"Month so far","group":"Mine","kind":"month_to_date"}
]`

func ids(defs []domain.RangeDef) string {
	var out []string
	for _, d := range defs {
		out = append(out, d.ID)
	}
	return strings.Join(out, ",")
}

func TestStartsWithTheBuiltInList(t *testing.T) {
	s, _ := setup(t)
	if s.Origin() != "built-in" || len(s.RangeDefs(context.Background())) != 18 {
		t.Errorf("origin %q, %d ranges", s.Origin(), len(s.RangeDefs(context.Background())))
	}
}

func TestAStoredListReplacesTheBuiltInOne(t *testing.T) {
	s, svc := setup(t)
	svc.store(custom)
	s.Refresh(context.Background())

	if got := ids(s.RangeDefs(context.Background())); got != "week,h1,mtd" || s.Origin() != "redis" {
		t.Errorf("ids %q origin %q", got, s.Origin())
	}
	d := s.RangeDefs(context.Background())[1]
	if d.Kind != domain.FixedDateRange || d.From != "2022-01-01" || d.To != "2022-06-30" || d.Group != "Mine" || d.Label != "First half" {
		t.Errorf("fields were not carried over: %+v", d)
	}
	if first := s.RangeDefs(context.Background())[0]; !first.Default || first.Days != 7 {
		t.Errorf("first = %+v", first)
	}
}

func TestRemovingTheStoredListGoesBackToBuiltIn(t *testing.T) {
	s, svc := setup(t)
	svc.store(custom)
	s.Refresh(context.Background())
	svc.mu.Lock()
	svc.val = nil
	svc.mu.Unlock()
	s.Refresh(context.Background())
	if s.Origin() != "built-in" || len(s.RangeDefs(context.Background())) != 18 {
		t.Errorf("origin %q, %d ranges", s.Origin(), len(s.RangeDefs(context.Background())))
	}
}

func TestAnUnusableStoredListIsIgnored(t *testing.T) {
	for name, val := range map[string]string{
		"not json":        `{"nope"`,
		"not a list":      `{"id":"x"}`,
		"empty list":      `[]`,
		"no default":      `[{"id":"a","label":"A","group":"G","kind":"days","days":7}]`,
		"unknown kind":    `[{"id":"a","label":"A","group":"G","kind":"fortnight","default":true}]`,
		"bad fixed dates": `[{"id":"a","label":"A","group":"G","kind":"fixed","from":"x","to":"y","default":true}]`,
	} {
		t.Run(name, func(t *testing.T) {
			s, svc := setup(t)
			svc.store(val)
			s.Refresh(context.Background())
			if s.Origin() != "built-in" || len(s.RangeDefs(context.Background())) != 18 {
				t.Errorf("origin %q, %d ranges", s.Origin(), len(s.RangeDefs(context.Background())))
			}
		})
	}
}

func TestAGoodListThatTurnsBadFallsBackToBuiltIn(t *testing.T) {
	s, svc := setup(t)
	svc.store(custom)
	s.Refresh(context.Background())
	svc.store(`[]`)
	s.Refresh(context.Background())
	if s.Origin() != "built-in" {
		t.Errorf("origin %q", s.Origin())
	}
}

func TestAnOutageKeepsTheListInUse(t *testing.T) {
	s, svc := setup(t)
	svc.store(custom)
	s.Refresh(context.Background())

	svc.down.Store(true)
	// The first refresh during the outage is answered from the cache's own memory copy; later
	// ones, once that has expired, find nothing. Either way the list in use must not change.
	s.Refresh(context.Background())
	if got := ids(s.RangeDefs(context.Background())); got != "week,h1,mtd" {
		t.Errorf("ids %q after an outage", got)
	}
}

func TestAnOutageUsesTheCachesInMemoryCopyOfTheList(t *testing.T) {
	c, svc := newWithCache(t)
	s := New(c)
	svc.store(custom)
	c.Get(context.Background(), Key) // the cache now holds a copy; the source has not looked yet

	svc.down.Store(true)
	s.Refresh(context.Background())
	if s.Origin() != "redis" || ids(s.RangeDefs(context.Background())) != "week,h1,mtd" {
		t.Errorf("origin %q ids %q", s.Origin(), ids(s.RangeDefs(context.Background())))
	}
}

func TestAnOutageAtStartKeepsTheBuiltInList(t *testing.T) {
	s, svc := setup(t)
	svc.down.Store(true)
	s.Refresh(context.Background())
	if s.Origin() != "built-in" || len(s.RangeDefs(context.Background())) != 18 {
		t.Errorf("origin %q", s.Origin())
	}
}

func TestWithoutACacheTheBuiltInListIsServed(t *testing.T) {
	s := New(nil)
	s.Refresh(context.Background())
	if s.Origin() != "built-in" || len(s.RangeDefs(context.Background())) != 18 {
		t.Errorf("origin %q", s.Origin())
	}
}

func TestRunRefreshesOnATimerAndStops(t *testing.T) {
	s, svc := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx, 20*time.Millisecond); close(done) }()

	svc.store(custom)
	deadline := time.Now().Add(2 * time.Second)
	for s.Origin() != "redis" {
		if time.Now().After(deadline) {
			t.Fatal("the timer never picked up the stored list")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop when its context ended")
	}
}

func TestTheBuiltInListSurvivesTheWireFormat(t *testing.T) {
	// The stored form must be able to express every built-in range.
	var parts []string
	for _, d := range domain.DefaultRangeDefs() {
		p := `{"id":"` + d.ID + `","label":"` + d.Label + `","group":"` + d.Group + `","kind":"` + string(d.Kind) + `"`
		if d.Days > 0 {
			p += `,"days":` + strconv.Itoa(d.Days)
		}
		if d.From != "" {
			p += `,"from":"` + d.From + `","to":"` + d.To + `"`
		}
		if d.Default {
			p += `,"default":true`
		}
		parts = append(parts, p+"}")
	}
	got, err := decode([]byte("[" + strings.Join(parts, ",") + "]"))
	if err != nil {
		t.Fatal(err)
	}
	want := domain.DefaultRangeDefs()
	if len(got) != len(want) {
		t.Fatalf("%d defs, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("def %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
