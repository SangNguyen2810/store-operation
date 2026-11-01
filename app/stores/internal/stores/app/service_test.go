package app

import (
	"context"
	"errors"
	"testing"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

type fakeRepo struct{ err error }

func (f fakeRepo) Stores(context.Context) ([]domain.Store, error) {
	return []domain.Store{{ID: 1, Label: "A"}}, f.err
}

func TestStoresPassThrough(t *testing.T) {
	s, err := NewService(fakeRepo{}).Stores(context.Background())
	if err != nil || len(s) != 1 || s[0].ID != 1 {
		t.Errorf("stores = %v, %v", s, err)
	}
}

func TestStoresErrorsPassThrough(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewService(fakeRepo{err: boom}).Stores(context.Background()); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}
