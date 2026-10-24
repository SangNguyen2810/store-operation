<h1 align="center">Store Operations Dashboard</h1>

<p align="center">
  A read-only analytics dashboard for a DVD-rental business: KPIs with period-over-period deltas,
  revenue by store over time, and top films.
</p>

<p align="center">
  <a href="https://github.com/SangNguyen2810/store-operation/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/SangNguyen2810/store-operation/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-strict-3178C6?logo=typescript&logoColor=white">
  <img alt="PostgreSQL" src="https://img.shields.io/badge/PostgreSQL-17-4169E1?logo=postgresql&logoColor=white">
</p>

---

## Overview

Store Operations Dashboard gives staff a single page to answer three questions: *how is the business doing, how does each store compare, and what is earning the most?* It is built on the [Pagila](https://github.com/devrimgunduz/pagila) sample database, a Go API and a React frontend.

**Highlights**

- **Key figures with context.** Revenue, rentals, active customers and average payment, each compared with the previous window of the same length.
- **Revenue by store.** One line per store, grouped by day, week or month, zero-filled so charts have no gaps. Includes a text summary and a "show data as a table" view.
- **Top films.** Ranked by revenue, with a relative bar.
- **Shareable views.** Every filter lives in the URL. Reload or send the link and you get the same view.
- **Resilient by design.** Each section loads, empties and fails on its own, with retry. One failing query never takes down the page.
- **Read-only.** The app adds no tables, views, indexes or migrations. Its database connections are opened with `default_transaction_read_only=on`.

## Quick start

**Prerequisites:** Go 1.25+, Node.js 22+, and a Pagila database reachable on `localhost:5432`. To start Pagila, run `docker compose up -d pagila` from the [Pagila repo](https://github.com/devrimgunduz/pagila).

```sh
make install   # install web dependencies
make dev       # API on :8080, web on :5173
```

Open <http://localhost:5173>.

| Command | What it does |
|---|---|
| `make dev` | Runs the API and the web dev server together |
| `make test` | Runs Go and web tests |
| `make lint` | `gofmt`, `go vet`, and `oxlint` |
| `make typecheck` | TypeScript project build check |
| `make build` | Builds the API binary and the production web bundle |
| `make check` | Everything CI runs: lint, typecheck, test, build |

### Configuration

All settings are environment variables and all are optional. See [.env.example](.env.example).

| Variable | Default | Purpose |
|---|---|---|
| `ADDR` | `:8080` | API listen address |
| `DATABASE_URL` | `postgres://postgres:123456@localhost:5432/postgres` | Pagila connection string |
| `ASOF_DATE` | `2022-07-27` | The "today" for default date ranges |
| `TEST_DATABASE_URL` | same as above | Database used by integration tests |
| `REQUIRE_DB` | unset | If set, integration tests fail instead of skipping when the database is unreachable |

Pagila's payments end on 2022-07-27, so "the last 30 days" needs a fixed anchor. The API exposes it at `GET /api/meta` and the UI reads it from there, so there is one source of truth.

## API

| Endpoint | Returns |
|---|---|
| `GET /healthz` | Database ping |
| `GET /api/meta` | `{ asOf }` |
| `GET /api/stores` | Stores that hold inventory |
| `GET /api/kpis` | Revenue, rentals, active customers and average payment, each with the previous window and `deltaPct` |
| `GET /api/revenue` | One point per (bucket, store), zero-filled |
| `GET /api/top/films` | Top films by revenue (`limit` 1-50) |

**Query parameters**

- `from`, `to`: `YYYY-MM-DD`, inclusive, at most 366 days apart.
- `store`: filter to one store.
- `granularity`: `day`, `week` or `month` (`/api/revenue` only).
- Defaults to the 30 days ending at the as-of date.

Invalid input returns `400` with an RFC 7807-style problem body. Internal errors never leak details to the client.

```sh
curl 'http://localhost:8080/api/kpis?from=2022-06-28&to=2022-07-27'
```

## Architecture

```
┌────────────────────┐   /api/*   ┌──────────────────────────────┐        ┌────────────┐
│  web (React 19)    │ ─────────▶ │  api (Go, net/http)          │ ─────▶ │  Pagila    │
│  Vite, TanStack    │            │  hexagonal, one context per  │  pgx   │ PostgreSQL │
│  Query             │ ◀───────── │  module, read-only pool      │ ◀───── │ (read-only)│
└────────────────────┘    JSON    └──────────────────────────────┘        └────────────┘
```

### Backend: hexagonal contexts

Each business area is a **bounded context** with its own domain, use cases and adapters. Dependencies point inward only.

```
api/
  cmd/api/                       composition root: wires contexts, serves HTTP
  internal/
    analytics/                   one context
      domain/                    types and pure rules. Standard library only.
      app/                       use cases + ports.go (the Repo port). Imports domain only.
      adapters/
        postgres/                driven adapter: implements Repo, owns all SQL
        http/                    driving adapter: routes, validation, response types
    platform/                    shared plumbing that knows no context
      db/  config/  httpx/       read-only pool, env config, JSON/problem/logging/health
    archtest/                    enforces the import rules below
```

```
 HTTP request ─▶ adapters/http ─▶ app (use case) ─▶ domain
                                      │
                              Repo port (interface)
                                      ▲
                               adapters/postgres ─▶ Pagila
```

| Layer | May import |
|---|---|
| `domain` | standard library |
| `app` | standard library, its own `domain` |
| `adapters/*` | its own `domain` and `app`, `platform`, third-party. Never another adapter |
| `platform` | no context |
| a context | itself and `platform`, never another context |

`internal/archtest` checks these rules on every `go test ./...`, so CI fails if a layer reaches the wrong way.

### Frontend

```
web/
  src/api/                   typed client, one function per endpoint
  src/features/overview/     URL filter state, query hooks, page and its parts
  src/components/            AsyncSection (loading/empty/error), TimeSeriesChart
```

### Design decisions

- **Filters live in the URL.** Only non-default values are written, so links stay short.
- **One state component.** `AsyncSection` handles loading, empty and error (with retry) for every section. While new filters load, the previous data stays on screen, dimmed, instead of flashing a spinner.
- **The chart library is isolated** in `TimeSeriesChart` and lazy-loaded. Swapping it touches one file, and the first load stays small.
- **Store attribution follows the `sales_by_store` view:** `payment → rental → inventory.store_id`. Staff-based attribution does not work in this dataset.
- **Accessible by default.** Controls are keyboard-reachable, deltas are conveyed by arrow and text rather than colour alone, and the chart has a text alternative and a table view.
- **The API contract is separate from the domain.** Response types live in the HTTP adapter, so the domain has no `json` tags and can change without breaking clients.
- **SQL stays in one place.** The Postgres adapter is the only code that queries these tables.
- **Packages in one `go.mod`.** Splitting into separate Go modules only pays off once there are two or more real contexts.

## Adding a context

To add one (for example `rentals`):

1. Create `internal/rentals/{domain,app,adapters/postgres,adapters/http}`, following `analytics`.
2. Define the repository port in `app/ports.go` and implement it in `adapters/postgres`.
3. Expose `Register(mux, ...)` from `adapters/http`, and call it in `cmd/api/main.go`.
4. Run `go test ./...`. The architecture test covers the new context automatically.

## Testing and CI

| Layer | Coverage |
|---|---|
| `api` domain and app | Window and delta maths, KPI comparison with a fake `Repo` |
| `api` HTTP adapter | Defaults, parameter passthrough, 10 bad-input cases, no leaked internals, health, 405 |
| `api` Postgres adapter (integration) | Totals reconciled against the `sales_by_store` view and raw SQL; revenue series sums to the KPI total at every granularity; zero-fill; top films; pool rejects writes |
| `api` architecture | Import rules between layers and contexts, plus a check that the rules catch violations |
| `web` | URL and filter parsing, formatting, data shaping, `AsyncSection` states, KPI cards, filter bar, the page against a mocked API |

Correctness is checked against SQL written independently of the code under test.

**CI** ([ci.yml](.github/workflows/ci.yml)) runs on every push to `main` and every pull request:

- `api`: `gofmt`, `go vet`, `go test`, `go build`
- `web`: `npm ci`, lint, typecheck, tests, build

> CI has no database yet, so the Pagila integration tests **skip** there and run only locally. A green CI run therefore does not prove the SQL is correct. Once a database is added to CI, set `REQUIRE_DB=1` so a missing database fails the run.

## Splitting into microservices later

The seams are already in place:

1. Each context reaches its database only through its own `Repo` port and owns its SQL, and the architecture test stops one context importing another. A future `rentals` context does the same, with no cross-context queries.
2. The HTTP adapter depends on a small `Analytics` interface, not a concrete type.
3. To split: give each context its own `cmd/<service>` binary and its own `db.Open` pool, put a gateway in front that routes `/api/*` (the web app only talks to `/api`), then move each context's tables into its own database. Cross-service data, such as a rental's customer name, goes behind a port: the in-process implementation is swapped for an HTTP or gRPC client, or a replicated snapshot, instead of a join.

## Project status and roadmap

- [x] **Phase 1: Overview.** KPIs, revenue by store, top films, URL filters.
- [x] **Backend refactor.** Hexagonal contexts with enforced layer rules.
- [ ] **Phase 2: Rentals Explorer.** Not started.
- [ ] **Microservices split.** Separate binaries, a gateway, then separate databases.

See [docs/PLAN.md](docs/PLAN.md) for decisions, verification notes and known gaps.
