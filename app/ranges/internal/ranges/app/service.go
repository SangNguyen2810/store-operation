// Package app holds the date-range use case. It depends on the domain and on
// the RangeSource port, never on a concrete store or on HTTP.
package app

import (
	"context"
	"time"

	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/domain"
)

type Service struct {
	ranges RangeSource
	asOf   time.Time
}

// NewService wires the use case. asOf is the dashboard's "today", which the
// relative date ranges are measured from.
func NewService(ranges RangeSource, asOf time.Time) *Service {
	return &Service{ranges: ranges, asOf: asOf}
}

// Ranges returns the date ranges the dashboard offers, resolved against as-of.
func (s *Service) Ranges(ctx context.Context) ([]domain.Range, error) {
	return domain.ResolveRanges(s.ranges.RangeDefs(ctx), s.asOf)
}
