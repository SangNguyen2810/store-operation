package app

import (
	"context"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
)

// RangeSource is the driven port for the list of date ranges the dashboard
// offers. It always has a list: when the configured one is unavailable it
// returns the built-in one.
type RangeSource interface {
	RangeDefs(ctx context.Context) []domain.RangeDef
}

// Repo is the driven port: how the use cases read data. The Postgres adapter implements it.
type Repo interface {
	Stores(ctx context.Context) ([]domain.Store, error)
	Totals(ctx context.Context, f domain.Filter) (domain.Totals, error)
	Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error)
	TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error)
}
