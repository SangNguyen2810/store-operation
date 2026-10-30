// Package postgres implements the analytics Repo port against Pagila. It owns
// all of the SQL; nothing else may query these tables.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/app"
	"github.com/SangNguyen2810/store-operation/app/api/internal/analytics/domain"
)

// Repo reads Pagila directly. Revenue is attributed to a store through
// payment -> rental -> inventory.store_id, the same rule as the sales_by_store view.
type Repo struct{ pool *pgxpool.Pool }

// Compile-time check that Repo satisfies the port, so a drifting signature
// fails here rather than at the wiring in cmd/api.
var _ app.Repo = (*Repo)(nil)

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Stores(ctx context.Context) ([]domain.Store, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.store_id, c.city || ', ' || co.country
		FROM store s
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
		err := row.Scan(&s.ID, &s.Label)
		return s, err
	})
}

func (r *Repo) Totals(ctx context.Context, f domain.Filter) (domain.Totals, error) {
	var t domain.Totals
	err := r.pool.QueryRow(ctx, `
		WITH pay AS (
			SELECT COALESCE(sum(p.amount), 0)::float8 AS revenue, count(*) AS n
			FROM payment p
			JOIN rental r ON r.rental_id = p.rental_id
			JOIN inventory i ON i.inventory_id = r.inventory_id
			WHERE p.payment_date >= $1 AND p.payment_date < $2
			  AND ($3::int IS NULL OR i.store_id = $3)
		), ren AS (
			SELECT count(*) AS rentals, count(DISTINCT r.customer_id) AS customers
			FROM rental r
			JOIN inventory i ON i.inventory_id = r.inventory_id
			WHERE r.rental_date >= $1 AND r.rental_date < $2
			  AND ($3::int IS NULL OR i.store_id = $3)
		)
		SELECT pay.revenue, pay.n, ren.rentals, ren.customers FROM pay, ren`,
		f.From, f.EndExclusive(), f.StoreID,
	).Scan(&t.Revenue, &t.Payments, &t.Rentals, &t.ActiveCustomers)
	if err != nil {
		return domain.Totals{}, fmt.Errorf("query totals: %w", err)
	}
	return t, nil
}

// Revenue returns one row per (bucket, store), zero-filled so charts have no gaps.
func (r *Repo) Revenue(ctx context.Context, f domain.Filter, g domain.Granularity) ([]domain.RevenuePoint, error) {
	if !g.Valid() {
		return nil, fmt.Errorf("invalid granularity %q", g)
	}
	rows, err := r.pool.Query(ctx, `
		WITH buckets AS (
			SELECT generate_series(date_trunc($3::text, $1::timestamptz), $2::timestamptz - interval '1 microsecond',
			                       ('1 ' || $3::text)::interval) AS bucket
		), stores AS (
			SELECT DISTINCT i.store_id FROM inventory i WHERE ($4::int IS NULL OR i.store_id = $4)
		), sales AS (
			SELECT date_trunc($3::text, p.payment_date) AS bucket, i.store_id, sum(p.amount) AS revenue
			FROM payment p
			JOIN rental r ON r.rental_id = p.rental_id
			JOIN inventory i ON i.inventory_id = r.inventory_id
			WHERE p.payment_date >= $1 AND p.payment_date < $2
			  AND ($4::int IS NULL OR i.store_id = $4)
			GROUP BY 1, 2
		)
		SELECT to_char(b.bucket, 'YYYY-MM-DD'), s.store_id, COALESCE(x.revenue, 0)::float8
		FROM buckets b
		CROSS JOIN stores s
		LEFT JOIN sales x ON x.bucket = b.bucket AND x.store_id = s.store_id
		ORDER BY b.bucket, s.store_id`,
		f.From, f.EndExclusive(), string(g), f.StoreID)
	if err != nil {
		return nil, fmt.Errorf("query revenue: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.RevenuePoint, error) {
		var p domain.RevenuePoint
		err := row.Scan(&p.Bucket, &p.StoreID, &p.Revenue)
		return p, err
	})
}

func (r *Repo) TopFilms(ctx context.Context, f domain.Filter, limit int) ([]domain.FilmStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT fi.film_id, fi.title, count(DISTINCT r.rental_id), sum(p.amount)::float8
		FROM payment p
		JOIN rental r ON r.rental_id = p.rental_id
		JOIN inventory i ON i.inventory_id = r.inventory_id
		JOIN film fi ON fi.film_id = i.film_id
		WHERE p.payment_date >= $1 AND p.payment_date < $2
		  AND ($3::int IS NULL OR i.store_id = $3)
		GROUP BY fi.film_id, fi.title
		ORDER BY sum(p.amount) DESC, fi.film_id
		LIMIT $4`,
		f.From, f.EndExclusive(), f.StoreID, limit)
	if err != nil {
		return nil, fmt.Errorf("query top films: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.FilmStat, error) {
		var s domain.FilmStat
		err := row.Scan(&s.FilmID, &s.Title, &s.Rentals, &s.Revenue)
		return s, err
	})
}
