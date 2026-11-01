// Package postgres implements the stores Repo port against Pagila. It owns the
// SQL; nothing else may query these tables.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/app"
	"github.com/SangNguyen2810/store-operation/app/stores/internal/stores/domain"
)

// Repo reads Pagila directly.
type Repo struct{ pool *pgxpool.Pool }

// Compile-time check that Repo satisfies the port, so a drifting signature
// fails here rather than at the wiring in cmd/stores.
var _ app.Repo = (*Repo)(nil)

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Stores(ctx context.Context) ([]domain.Store, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.store_id, c.city || ', ' || co.country,
		       st.first_name || ' ' || st.last_name,
		       COALESCE(a.phone, ''), a.address, a.district, COALESCE(a.postal_code, ''), c.city, co.country,
		       (SELECT count(*) FROM inventory i WHERE i.store_id = s.store_id),
		       (SELECT count(*) FROM customer cu WHERE cu.store_id = s.store_id AND cu.activebool)
		FROM store s
		JOIN staff st ON st.staff_id = s.manager_staff_id
		JOIN address a ON a.address_id = s.address_id
		JOIN city c ON c.city_id = a.city_id
		JOIN country co ON co.country_id = c.country_id
		WHERE EXISTS (SELECT 1 FROM inventory i WHERE i.store_id = s.store_id)
		ORDER BY s.store_id`)
	if err != nil {
		return nil, fmt.Errorf("query stores: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Store, error) {
		var s domain.Store
		err := row.Scan(&s.ID, &s.Label, &s.Manager, &s.Phone, &s.Address, &s.District, &s.PostalCode,
			&s.City, &s.Country, &s.InventoryItems, &s.ActiveCustomers)
		return s, err
	})
}
