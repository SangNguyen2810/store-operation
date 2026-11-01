// Package app holds the analytics use cases. It depends on the domain and on
// the Repo port, never on a concrete database or on HTTP.
package app

import (
	"context"

	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/domain"
)

type Service struct {
	repo   Repo
	ranges RangeSource
}

func NewService(repo Repo, ranges RangeSource) *Service {
	return &Service{repo: repo, ranges: ranges}
}

// Ranges returns the date ranges a request may ask for.
func (s *Service) Ranges(ctx context.Context) ([]domain.Range, error) { return s.ranges.Ranges(ctx) }

func (s *Service) Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error) {
	return s.repo.Revenue(ctx, f, g)
}

func (s *Service) TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error) {
	return s.repo.TopFilms(ctx, f, limit)
}

// KPIs compares the window with the immediately preceding window of equal length.
func (s *Service) KPIs(ctx context.Context, f domain.Filter) (domain.KPIs, error) {
	cur, err := s.repo.Totals(ctx, f)
	if err != nil {
		return domain.KPIs{}, err
	}
	prev, err := s.repo.Totals(ctx, f.PreviousWindow())
	if err != nil {
		return domain.KPIs{}, err
	}
	return domain.CompareKPIs(cur, prev), nil
}
