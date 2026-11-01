package rangesclient

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/domain"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

var asOf = time.Date(2022, 7, 27, 0, 0, 0, 0, time.UTC)

const week = `[
  {"id":"week","label":"Week","group":"G","from":"2022-07-21","to":"2022-07-27","default":true},
  {"id":"h1","label":"H1","group":"G","from":"2022-01-01","to":"2022-06-30","default":false}
]`

// fakeRanges is the ranges service: it answers body with status, and counts requests.
type fakeRanges struct {
	mu     sync.Mutex
	status int
	body   string
	calls  atomic.Int32
}

func (f *fakeRanges) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path != "/api/presets" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(f.status)
	_, _ = io.WriteString(w, f.body)
}

func (f *fakeRanges) set(status int, body string) {
	f.mu.Lock()
	f.status, f.body = status, body
	f.mu.Unlock()
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func setup(t *testing.T) (*Client, *fakeRanges, *clock) {
	t.Helper()
	fake := &fakeRanges{status: http.StatusOK, body: week}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, asOf)
	if err != nil {
		t.Fatal(err)
	}
	clk := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	c.now = clk.now
	return c, fake, clk
}

func ids(rs []domain.Range) string {
	var out []string
	for _, r := range rs {
		out = append(out, r.ID)
	}
	return strings.Join(out, ",")
}

func get(t *testing.T, c *Client) []domain.Range {
	t.Helper()
	rs, err := c.Ranges(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return rs
}

func TestTheListComesFromTheService(t *testing.T) {
	c, _, _ := setup(t)
	rs := get(t, c)
	if ids(rs) != "week,h1" || !rs[0].Default || rs[0].From.Format(dateLayout) != "2022-07-21" || rs[0].To.Format(dateLayout) != "2022-07-27" {
		t.Errorf("ranges = %+v", rs)
	}
}

func TestAnAnswerIsReusedUntilItGoesStale(t *testing.T) {
	c, fake, clk := setup(t)
	get(t, c)
	clk.advance(FreshFor - time.Second)
	get(t, c)
	if n := fake.calls.Load(); n != 1 {
		t.Fatalf("%d requests inside the fresh window, want 1", n)
	}
	clk.advance(2 * time.Second)
	get(t, c)
	if n := fake.calls.Load(); n != 2 {
		t.Fatalf("%d requests after it went stale, want 2", n)
	}
}

func TestAChangeAtTheServiceShowsUpAfterTheFreshWindow(t *testing.T) {
	c, fake, clk := setup(t)
	get(t, c)
	fake.set(http.StatusOK, `[{"id":"x","label":"X","group":"G","from":"2022-07-01","to":"2022-07-02","default":true}]`)
	clk.advance(FreshFor + time.Second)
	if got := ids(get(t, c)); got != "x" {
		t.Errorf("ids = %s, want x", got)
	}
}

func TestBeforeTheFirstAnswerTheBuiltInListIsServed(t *testing.T) {
	c, fake, _ := setup(t)
	fake.set(http.StatusServiceUnavailable, "")
	rs := get(t, c)
	want, _ := domain.ResolveRanges(domain.DefaultRangeDefs(), asOf)
	if ids(rs) != ids(want) {
		t.Errorf("ids = %s, want the built-in list", ids(rs))
	}
	if def, ok := domain.DefaultRange(rs); !ok || def.ID != "last-30" {
		t.Errorf("default = %+v", def)
	}
}

func TestAnOutageKeepsTheLastGoodList(t *testing.T) {
	c, fake, clk := setup(t)
	get(t, c)
	fake.set(http.StatusInternalServerError, "")
	clk.advance(FreshFor + time.Second)
	if got := ids(get(t, c)); got != "week,h1" {
		t.Errorf("ids = %s, want the last good list", got)
	}
}

func TestADownServiceIsLeftAloneForAWhile(t *testing.T) {
	c, fake, clk := setup(t)
	fake.set(http.StatusServiceUnavailable, "")
	get(t, c)
	before := fake.calls.Load()
	get(t, c)
	get(t, c)
	if n := fake.calls.Load(); n != before {
		t.Fatalf("%d more requests while marked down", n-before)
	}

	fake.set(http.StatusOK, week)
	clk.advance(retryAfter + time.Second)
	if got := ids(get(t, c)); got != "week,h1" {
		t.Errorf("after recovery ids = %s", got)
	}
}

func TestBadAnswersAreIgnored(t *testing.T) {
	for name, body := range map[string]string{
		"not json":      "nope",
		"empty list":    "[]",
		"no default":    `[{"id":"a","label":"A","group":"G","from":"2022-07-01","to":"2022-07-02"}]`,
		"two defaults":  `[{"id":"a","label":"A","group":"G","from":"2022-07-01","to":"2022-07-02","default":true},{"id":"b","label":"B","group":"G","from":"2022-07-01","to":"2022-07-02","default":true}]`,
		"bad from":      `[{"id":"a","label":"A","group":"G","from":"soon","to":"2022-07-02","default":true}]`,
		"bad to":        `[{"id":"a","label":"A","group":"G","from":"2022-07-01","to":"later","default":true}]`,
		"from after to": `[{"id":"a","label":"A","group":"G","from":"2022-07-03","to":"2022-07-02","default":true}]`,
		"too many":      "[" + strings.Repeat(`{"id":"a","label":"A","group":"G","from":"2022-07-01","to":"2022-07-02","default":true},`, domain.MaxRangeDefs) + `{"id":"z","label":"Z","group":"G","from":"2022-07-01","to":"2022-07-02"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			c, fake, _ := setup(t)
			fake.set(http.StatusOK, body)
			if got := ids(get(t, c)); !strings.HasPrefix(got, "last-7,") {
				t.Errorf("ids = %s, want the built-in list", got)
			}
		})
	}
}

func TestAnUnreachableServiceIsHandled(t *testing.T) {
	c, err := New("http://127.0.0.1:1", asOf) // nothing listens
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(get(t, c)); !strings.HasPrefix(got, "last-7,") {
		t.Errorf("ids = %s, want the built-in list", got)
	}
}

func TestNewRejectsBadURLs(t *testing.T) {
	for _, u := range []string{"", "ranges:8083", "redis://ranges:8083", "http://"} {
		if _, err := New(u, asOf); err == nil {
			t.Errorf("New(%q) should fail", u)
		}
	}
	if _, err := New("http://ranges:8083/", asOf); err != nil {
		t.Errorf("valid URL: %v", err)
	}
}
