// Package app holds the stores use case. It depends on the domain and on the
// Repo port, never on a concrete database or on HTTP.
package app

import (
	"context"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

type Service struct{ repo Repo }

func NewService(repo Repo) *Service { return &Service{repo: repo} }

// Stores lists the shops with their details.
func (s *Service) Stores(ctx context.Context) ([]domain.Store, error) { return s.repo.Stores(ctx) }
