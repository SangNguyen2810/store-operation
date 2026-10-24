# Store Operations Dashboard

A staff dashboard over the [Pagila](../README.md) DVD-rental database: KPIs with period-over-period deltas, revenue by store over time, and top films. Go API, React frontend.

The dashboard only **reads** Pagila. It adds no tables, views, indexes or migrations, and its connections are opened with `default_transaction_read_only=on`.

## Run it

Needs Go, Node 22+, and the Pagila container from the repo root running (`docker compose up -d pagila`).

```sh
make install   # npm install
make dev       # API on :8080, web on :5173
```

Open http://localhost:5173. `make check` runs lint, typecheck, all tests and the production build.

Configuration is by environment variable; see [.env.example](.env.example). Pagila's payments end on 2022-07-27, so "today" is pinned with `ASOF_DATE` and the UI reads it from `GET /api/meta`.

## API

| Endpoint | Returns |
|---|---|
| `GET /api/meta` | `{asOf}` |
| `GET /api/stores` | stores that hold inventory |
| `GET /api/kpis` | revenue, rentals, active customers, average payment, each with the previous window and `deltaPct` |
| `GET /api/revenue` | one point per (bucket, store), zero-filled so charts have no gaps |
| `GET /api/top/films` | top films by revenue (`limit` 1-50) |

Shared query parameters: `from`, `to` (YYYY-MM-DD, inclusive, max 366 days), `store`. `/api/revenue` also takes `granularity=day|week|month`. Defaults are the 30 days ending at the as-of date. Bad input returns a `400` problem body.

## Architecture

```
web/                         React 19, TypeScript, Vite, TanStack Query
  src/api/                   typed client, one function per endpoint
  src/features/overview/     filters (URL state), query hooks, page and its parts
  src/components/            AsyncSection (loading/empty/error), TimeSeriesChart

api/                         Go, net/http, pgx
  internal/analytics/        Service + Repo interface + Postgres repo. Owns all SQL.
  internal/httpapi/          parsing, validation, JSON. No SQL.
  internal/db/               read-only pool
```

Decisions worth knowing:

- **Filters live in the URL.** Reload or share the link and you get the same view. Only non-default values are written.
- **One state component.** `AsyncSection` handles loading, empty and error (with retry) for every section, so a failing query degrades one section, not the page. While new filters load, the previous data stays up, dimmed.
- **The chart library is isolated** in `TimeSeriesChart` and lazy-loaded, so swapping it touches one file and the first load stays small.
- **Store attribution** follows the `sales_by_store` view: `payment -> rental -> inventory.store_id`. Staff-based attribution does not work in this dataset.
- **Correctness is tested against independent SQL.** Integration tests reconcile totals with `sales_by_store` and raw queries, and check that the revenue series sums to the KPI total at every granularity.

## Splitting into microservices later

The seam is already in place:

1. `analytics` talks to the database only through the `Repo` interface and owns its SQL. A future `rentals` package should do the same, with no cross-package queries.
2. `httpapi` depends on a small `Analytics` interface, not on a concrete type.
3. To split: give each package its own `cmd/<service>` binary and its own `db.Open` pool, put a gateway in front that routes `/api/*` (the web app already only talks to `/api`), and then move each package's tables into its own database. Cross-service data (for example a rental's customer name) becomes an API call or a replicated snapshot instead of a join.

## Tests

- `api`: unit tests (window and delta maths), handler tests (validation, errors, no leaked internals), and integration tests against the live Pagila container. These skip if the database is unreachable.
- `web`: unit tests for URL/filter parsing, formatting and data shaping, plus component tests for each state, the filter bar, and the page against a mocked API.
