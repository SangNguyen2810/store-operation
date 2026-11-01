package app

import (
	"context"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

// Repo is the driven port: how the use case reads stores. The Postgres adapter
// implements it, and the cache adapter wraps it.
type Repo interface {
	Stores(ctx context.Context) ([]domain.Store, error)
}
