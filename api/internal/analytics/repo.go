package analytics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepo reads Pagila directly. Revenue is attributed to a store through
// payment -> rental -> inventory.store_id, the same rule as the sales_by_store view.
type PgRepo struct{ pool *pgxpool.Pool }

func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{pool: pool} }

func (r *PgRepo) Stores(ctx context.Context) ([]Store, error) {
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
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Store, error) {
		var s Store
		err := row.Scan(&s.ID, &s.Label)
		return s, err
	})
}

func (r *PgRepo) Totals(ctx context.Context, f Filter) (Totals, error) {
	var t Totals
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
		f.From, endExclusive(f), f.StoreID,
	).Scan(&t.Revenue, &t.Payments, &t.Rentals, &t.ActiveCustomers)
	if err != nil {
		return Totals{}, fmt.Errorf("query totals: %w", err)
	}
	return t, nil
}

// Revenue returns one row per (bucket, store), zero-filled so charts have no gaps.
func (r *PgRepo) Revenue(ctx context.Context, f Filter, g Granularity) ([]RevenuePoint, error) {
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
		f.From, endExclusive(f), string(g), f.StoreID)
	if err != nil {
		return nil, fmt.Errorf("query revenue: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (RevenuePoint, error) {
		var p RevenuePoint
		err := row.Scan(&p.Bucket, &p.StoreID, &p.Revenue)
		return p, err
	})
}

func (r *PgRepo) TopFilms(ctx context.Context, f Filter, limit int) ([]FilmStat, error) {
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
		f.From, endExclusive(f), f.StoreID, limit)
	if err != nil {
		return nil, fmt.Errorf("query top films: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (FilmStat, error) {
		var s FilmStat
		err := row.Scan(&s.FilmID, &s.Title, &s.Rentals, &s.Revenue)
		return s, err
	})
}
