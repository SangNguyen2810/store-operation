package app

import (
	"context"

	"github.com/SangNguyen2810/store-operation/app/analytics/internal/analytics/domain"
)

// RangeSource is the driven port for the date ranges the dashboard offers. It
// always has a list: when the ranges service is unavailable it returns the
// built-in one.
type RangeSource interface {
	Ranges(ctx context.Context) ([]domain.Range, error)
}

// Repo is the driven port: how the use cases read data. The Postgres adapter implements it.
type Repo interface {
	Totals(ctx context.Context, f domain.Filter) (domain.Totals, error)
	Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error)
	TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error)
}
