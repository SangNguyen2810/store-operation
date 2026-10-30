// Package ranges is a driven adapter that supplies the list of date ranges the
// dashboard offers. The list lives in the redis service under one key; this
// adapter keeps a copy and refreshes it on a timer, so a request never waits on
// the service. With no cache, or no list stored, or an unusable one, it serves
// the built-in list.
package ranges

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/app"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
	platformcache "github.com/SangNguyen2810/store-operation/app/api/internal/platform/cache"
)

// Key is where the list is stored in the redis service. It has the persistent
// `config:` prefix, so it never expires. The version changes with the format.
const Key = "config:presets:v1"

// RefreshEvery is how often the copy is refreshed.
const RefreshEvery = 60 * time.Second

const (
	fromBuiltIn = "built-in"
	fromRedis   = "redis"
)

// Compile-time check that Source satisfies the port.
var _ app.RangeSource = (*Source)(nil)

type snapshot struct {
	defs   []domain.RangeDef
	origin string
}

// Source holds the current list.
type Source struct {
	cache *platformcache.Cache
	cur   atomic.Pointer[snapshot]
}

// New returns a source over cache. A nil cache gives a source that always serves the built-in list.
func New(cache *platformcache.Cache) *Source {
	s := &Source{cache: cache}
	s.cur.Store(&snapshot{defs: domain.DefaultRangeDefs(), origin: fromBuiltIn})
	return s
}

// RangeDefs returns the current list. It never blocks and is never empty.
func (s *Source) RangeDefs(context.Context) []domain.RangeDef { return s.cur.Load().defs }

// Origin says where the current list came from: "redis" or "built-in".
func (s *Source) Origin() string { return s.cur.Load().origin }

// Refresh reads the list from the cache once. If the redis service is
// unavailable and holds no copy, the list in use stays as it is.
func (s *Source) Refresh(ctx context.Context) {
	if s.cache == nil {
		return
	}
	b, out := s.cache.Get(ctx, Key)
	switch out {
	case platformcache.Hit, platformcache.Fallback:
		defs, err := decode(b)
		if err != nil {
			slog.WarnContext(ctx, "ranges: ignoring the stored list, using the built-in one", "key", Key, "err", err)
			s.set(domain.DefaultRangeDefs(), fromBuiltIn)
			return
		}
		s.set(defs, fromRedis)
	case platformcache.Miss:
		s.set(domain.DefaultRangeDefs(), fromBuiltIn)
	case platformcache.Unavailable:
		// Keep what is in use; the cache has already logged the outage.
	}
}

func (s *Source) set(defs []domain.RangeDef, origin string) {
	if prev := s.cur.Load(); prev.origin != origin {
		slog.Info("ranges: list source changed", "from", prev.origin, "to", origin, "count", len(defs))
	}
	s.cur.Store(&snapshot{defs: defs, origin: origin})
}

// Run refreshes every `every` until ctx ends. Call Refresh once first, so the
// first request already sees the stored list.
func (s *Source) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Refresh(ctx)
		}
	}
}

// wireDef is the stored JSON form of a domain.RangeDef.
type wireDef struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Days    int    `json:"days,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Default bool   `json:"default,omitempty"`
}

func decode(b []byte) ([]domain.RangeDef, error) {
	var in []wireDef
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, err
	}
	defs := make([]domain.RangeDef, 0, len(in))
	for _, w := range in {
		defs = append(defs, domain.RangeDef{
			ID: w.ID, Label: w.Label, Group: w.Group, Kind: domain.RangeKind(w.Kind),
			Days: w.Days, From: w.From, To: w.To, Default: w.Default,
		})
	}
	if err := domain.ValidateRangeDefs(defs); err != nil {
		return nil, err
	}
	return defs, nil
}
