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
- **Responsive.** Built mobile-first for phone, tablet and desktop, in light and dark.
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

### System at a glance

```mermaid
flowchart LR
    user([Staff member]) --> web
    subgraph browser [Browser]
        web["web<br/>React 19 · TypeScript · Vite<br/>TanStack Query"]
    end
    subgraph server [Go API · one deployable]
        direction TB
        api["cmd/api<br/>composition root"]
        analytics["analytics context<br/>domain · app · adapters"]
        platform["platform<br/>db · config · httpx"]
        api --> analytics
        api --> platform
        analytics --> platform
    end
    web -- "JSON over /api/*" --> api
    analytics -- "pgx · read-only pool" --> pg[("Pagila<br/>PostgreSQL 17")]
```

**Four ideas hold the design together:**

1. **Hexagonal inside, contexts outside.** Each business area is a bounded context with a pure domain, use cases that talk to ports, and adapters that plug the ports into the world.
2. **Dependencies point inward, and a test enforces it.** Layers and contexts cannot reach around each other without failing CI.
3. **The database is only ever read.** No tables, views, indexes or migrations are added, and the pool refuses writes.
4. **Every screen section stands alone.** Loading, empty and error are handled per section, and filters live in the URL, so any view can be shared.

### Backend: hexagonal contexts

```mermaid
flowchart TB
    subgraph ctx [internal/analytics · one bounded context]
        direction TB
        subgraph driving [Driving side]
            http["adapters/http<br/>routes · validation · response types"]
        end
        subgraph core [Core]
            app["app<br/>use cases · Repo port"]
            domain["domain<br/>types · pure rules<br/>(stdlib only)"]
        end
        subgraph driven [Driven side]
            pgadapter["adapters/postgres<br/>implements Repo · owns all SQL"]
        end
        http --> app
        app --> domain
        pgadapter -. "implements Repo" .-> app
        pgadapter --> domain
    end
    platform["internal/platform<br/>db · config · httpx"]
    http --> platform
    pgadapter --> platform
    main["cmd/api"] --> http
    main --> pgadapter
    main --> app
```

Solid arrows are imports and always point inward. The dotted arrow is the one place the flow of control and the import direction differ: `app` defines the `Repo` interface, and `adapters/postgres` satisfies it (dependency inversion). `cmd/api` is the only place that knows every concrete type.

```
api/
  cmd/api/                       composition root: builds the pool, wires contexts, serves HTTP
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

**Where does it go?**

| If you are changing… | It lives in |
|---|---|
| A business rule (how a delta or previous window is computed) | `domain/` |
| What the system can do (a new question it can answer) | `app/` |
| A query, or how Pagila tables are read | `adapters/postgres/` |
| A route, a query parameter, or the JSON shape | `adapters/http/` |
| Timeouts, logging, error format, the DB pool | `platform/` |
| Which pieces are wired together | `cmd/api/main.go` |

**Import rules** (checked on every `go test ./...` by `internal/archtest`):

| Layer | May import |
|---|---|
| `domain` | standard library |
| `app` | standard library, its own `domain` |
| `adapters/*` | its own `domain` and `app`, `platform`, third-party. Never another adapter |
| `platform` | no context |
| a context | itself and `platform`, never another context |
| `cmd` | anything (composition root) |

### Request lifecycle

What happens on `GET /api/kpis?from=2022-06-28&to=2022-07-27&store=1`:

```mermaid
sequenceDiagram
    autonumber
    participant W as web (TanStack Query)
    participant H as adapters/http
    participant A as app.Service
    participant D as domain
    participant P as adapters/postgres
    participant DB as Pagila

    W->>H: GET /api/kpis?from&to&store
    H->>H: parse and validate (400 problem on bad input)
    H->>A: KPIs(ctx, Filter)
    A->>P: Totals(current window)
    P->>DB: one query, read-only
    DB-->>P: revenue, payments, rentals, customers
    A->>D: Filter.PreviousWindow()
    A->>P: Totals(previous window)
    P->>DB: same query, earlier dates
    A->>D: CompareKPIs(current, previous)
    D-->>A: KPIs with deltas
    A-->>H: domain.KPIs
    H->>H: map to response type
    H-->>W: 200 JSON
```

Errors are handled once, at the edge: bad input becomes a `400` problem body with a reason, and anything else is logged server-side and returned as a generic `500`, so internals never reach the client.

### Frontend

The front end follows the same rule as the back end: dependencies point inward, and a test enforces it.

```mermaid
flowchart LR
    pages["pages<br/>route-level screens"] --> features
    features["features<br/>containers: call hooks, apply rules,<br/>map data to props"] --> components
    features --> hooks["hooks<br/>one react-query hook per file<br/>+ URL state"]
    hooks --> api["api<br/>typed client"]
    features --> lib["lib<br/>formatting, filters, dates"]
    hooks --> lib
    components["components<br/>pure UI: props in, JSX out"] --> lib
    components --> libs["components/libs/recharts<br/>the only door to the chart library"]
    lib --> api
```

```
web/src/
  routes.tsx, App.tsx, main.tsx   composition: the route table ("/" dashboard, anything else 404) and providers
  pages/                          thin route-level screens (HomePage, NotFoundPage)
  features/overview/              OverviewPage (the one container) + mapping helpers (kpis.ts, films.ts, OverviewFilters)
  hooks/                          useMeta, useStores, useKpis, useRevenue, useTopFilms, useFilters, queryKeys
  components/                     one folder per component: Name.tsx + index.ts + Name.test.tsx
    AsyncSection  ErrorState  PageHeader  Skeleton  TimeSeriesChart
    KpiCard  KpiGrid  RankedTable  SegmentedControl  SelectField  DateField  FilterPanel
    libs/recharts/                wrappers for the 8 chart pieces we use, with our design defaults
  api/                            typed client, one function per endpoint
  lib/                            dates, format, filters, useBreakpoint
```

| Layer | May import |
|---|---|
| `api` | itself |
| `lib` | `api` (types) |
| `hooks` | `api`, `lib` |
| `components` | other components, `lib`. Never `hooks`, `features`, `pages`, `api`, `react-router` or `react-query` |
| `features` | `components`, `hooks`, `lib`, `api` |
| `pages` | `features` and everything below |
| `'recharts'` | only `components/libs/recharts` may import it |

Rules, enforced by `src/archtest.test.ts` on every `npm test`:

- **Pure UI.** A component takes plain props (strings, numbers, options, callbacks) and has no idea what a "film" or a "filter" is. `KpiGrid` renders cards; `features/overview/kpis.ts` decides what the cards say.
- **Logic in features.** A feature calls the hooks, applies the rules (limits, presets, formatting) and maps domain data to props. `OverviewPage` is the only place that calls the query hooks.
- **One door to the chart library.** Everything charts need comes from `components/libs/recharts`. Each of the eight wrappers sets our theme defaults (grid and axis colours, tooltip style, 2 px lines with no dots and no animation) and lets the caller override any of them. Swapping the library is a one-folder change.
- **One file per query.** Each react-query hook has its own file and test, and cache keys live in `queryKeys.ts`.

### Responsive design

Mobile-first CSS with two breakpoints. Anything CSS can size is sized by CSS; only the chart's pixel height and axis width, which the chart library takes as props, read the viewport (`useBreakpoint`).

| | Mobile (< 640 px) | Tablet (640 to 1023 px) | Desktop (1024 px and up) |
|---|---|---|---|
| **Page width** | Full width, 16 px gutters | Full width, 24 px gutters | Up to 1280 px |
| **Filters** | Presets plus a **Filters** button (shows an active count); dates, store and grouping open in place, with **Reset filters** | Presets, then fields two per row | One aligned row, every control the same height |
| **Key figures** | 2 × 2 | 4 across | 4 across |
| **Revenue / Top films** | Stacked, 240 px chart | Stacked, 300 px chart | Side by side (3:2), 380 px chart |
| **Top films** | Rentals under the title | Rentals column | Rentals column |
| **Touch targets** | 44 px | 44 px on touch, 40 px with a mouse | 40 px |

- `main` is a `minmax(0, 1fr)` grid, so wide children (chart, tables) can never stretch the page past the viewport.
- On touch screens, controls keep the 44 px target at every width (`pointer: coarse`).
- The Filters button exposes `aria-expanded` and `aria-controls`; from 640 px up the fields are always visible and the button is hidden.
- Verified at 375, 768 and 1280 px in light and dark: no horizontal page scroll, and every button, input and select meets the minimum height.

### Cross-cutting concerns

| Concern | How it is handled |
|---|---|
| **Read-only safety** | Pool opened with `default_transaction_read_only=on`; an integration test confirms writes are rejected |
| **Time zone** | Connections run in UTC, windows are inclusive whole days |
| **Timeouts** | 10 s statement timeout, 5 s header read, 15 s response write, 10 s graceful shutdown |
| **Errors** | RFC 7807-style problem bodies; internals are logged, never returned |
| **Observability** | Structured `slog` request logs with method, path, query and duration; `/healthz` pings the database |
| **Accessibility** | Keyboard-reachable controls, deltas shown by arrow and text (not colour alone), chart text alternative and table view |
| **Performance** | Chart library lazy-loaded; first load is about 95 kB gzipped; previous data stays up while new filters load |

### Key decisions

| Decision | Alternatives considered | Why |
|---|---|---|
| Hexagonal layers per context | One flat package; classic handler/service/repository layers | The ports are the seams you cut along when splitting services. Extra layers would add files without adding a boundary. |
| Packages in one `go.mod` | A `go.mod` per context with `go.work` | Separate modules only pay off with two or more real contexts. Revisit when `rentals` exists. |
| Response types in the HTTP adapter | JSON tags on domain types | The domain can change without breaking clients, and the transport can change (for example to gRPC) without touching the domain. |
| Layer rules enforced by a test | A convention in the docs | Rules that are not checked drift. The test needs no extra tooling. |
| Filters in the URL | Component or global state | Shareable, reload-safe, and the browser back button works. |
| One `AsyncSection` for query states | Per-component spinners | A failing query degrades one section, not the page. |
| Store attribution via `payment → rental → inventory.store_id` | Via `staff.store_id` | Matches the `sales_by_store` view; staff-based attribution does not work in this dataset. |
| Chart library isolated and lazy-loaded | Import directly where needed | Swapping the library touches one file, and the first load stays small. |

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
| `web` architecture | Import rules between the layers, and that only `components/libs/recharts` imports `recharts`, with cases proving the rules catch violations |
| `web` | Routing (dashboard at `/`, 404 elsewhere), URL and filter parsing, formatting, data shaping, breakpoint mapping, `AsyncSection` states, KPI cards, filter bar (including the collapsible panel, active count and reset), the page against a mocked API |

Correctness is checked against SQL written independently of the code under test.

**CI** ([ci.yml](.github/workflows/ci.yml)) runs on every push to `main` and every pull request:

- `api`: `gofmt`, `go vet`, `go test`, `go build`
- `web`: `npm ci`, lint, typecheck, tests, build

> CI has no database yet, so the Pagila integration tests **skip** there and run only locally. A green CI run therefore does not prove the SQL is correct. Once a database is added to CI, set `REQUIRE_DB=1` so a missing database fails the run.

## Splitting into microservices later

Today everything is one deployable. The target keeps the same contexts and swaps the in-process wiring for network calls:

```mermaid
flowchart LR
    subgraph today [Today: one deployable]
        direction TB
        t_api["cmd/api"] --> t_an["analytics"]
        t_api --> t_re["rentals (phase 2)"]
        t_an --> t_db[("one Pagila database")]
        t_re --> t_db
    end
    subgraph later [Later: one service per context]
        direction TB
        gw["gateway<br/>routes /api/*"] --> s_an["analytics service"]
        gw --> s_re["rentals service"]
        s_an --> d_an[("analytics DB")]
        s_re --> d_re[("rentals DB")]
    end
    today ==> later
```

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
