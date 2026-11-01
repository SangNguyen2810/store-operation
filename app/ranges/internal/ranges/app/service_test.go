package app

import (
	"context"
	"testing"
	"time"

	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/domain"
)

var asOf = time.Date(2022, 7, 27, 0, 0, 0, 0, time.UTC)

type builtIn struct{}

func (builtIn) RangeDefs(context.Context) []domain.RangeDef { return domain.DefaultRangeDefs() }

// badSource offers a list that cannot be resolved.
type badSource struct{}

func (badSource) RangeDefs(context.Context) []domain.RangeDef { return nil }

func TestRangesAreResolvedAgainstAsOf(t *testing.T) {
	got, err := NewService(builtIn{}, asOf).Ranges(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range got {
		if r.ID == "last-30" {
			if r.From.Format("2006-01-02") != "2022-06-28" || !r.To.Equal(asOf) || !r.Default {
				t.Errorf("last-30 = %+v", r)
			}
			return
		}
	}
	t.Error("last-30 is missing")
}

func TestAnUnusableListIsAnError(t *testing.T) {
	if _, err := NewService(badSource{}, asOf).Ranges(context.Background()); err == nil {
		t.Fatal("an empty list must not resolve")
	}
}
