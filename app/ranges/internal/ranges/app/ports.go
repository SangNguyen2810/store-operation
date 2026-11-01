package app

import (
	"context"

	"github.com/SangNguyen2810/store-operation/app/ranges/internal/ranges/domain"
)

// RangeSource is the driven port for the list of date ranges the dashboard
// offers. It always has a list: when the configured one is unavailable it
// returns the built-in one.
type RangeSource interface {
	RangeDefs(ctx context.Context) []domain.RangeDef
}
