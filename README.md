<h1 align="center">Store Operations Dashboard</h1>

<p align="center">
  A read-only analytics dashboard for a DVD-rental business: KPIs with period-over-period deltas,
  revenue by store over time, and top films. Built on the <a href="https://github.com/devrimgunduz/pagila">Pagila</a> sample database.
</p>

<p align="center">
  <a href="https://github.com/SangNguyen2810/store-operation/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/SangNguyen2810/store-operation/actions/workflows/ci.yml/badge.svg"></a>
</p>

---

## Quick start

**Prerequisites:** Docker, Go 1.25+, Node.js 22+, and a Pagila database on `localhost:5432`. To start Pagila, run `docker compose up -d pagila` from the [Pagila repo](https://github.com/devrimgunduz/pagila).

```sh
make install   # web dependencies, once
make up        # the backend in Docker: Redis, redis service, stores, ranges, analytics and the gateway (:8080)
make dev       # the web dev server on :5173
```

Open <http://localhost:5173>. `make down` stops the backend. For the web app in Docker too, run `make up-all` and open <http://localhost:3000>.

The services in Docker reach Pagila on the host; set `DOCKER_DATABASE_URL` if yours is elsewhere. `GATEWAY_PORT`, `WEB_PORT` and `REDIS_PORT` change the published ports (defaults 8080, 3000 and 6380). If the web app shows "Not Found" errors, something else may hold `:8080`: check with `lsof -nP -iTCP:8080 -sTCP:LISTEN`.

| Command | What it does |
|---|---|
| `make up` / `make down` | Start or stop the backend in Docker |
| `make up-all` | Like `make up`, plus the web app at <http://localhost:3000> |
| `make dev` | The web dev server on `:5173` (needs `make up`) |
| `make dev-analytics` / `dev-stores` / `dev-ranges` / `dev-gateway` | Run one service locally while the rest runs in Docker |
| `make seed-presets` | Store [`infra/presets.json`](infra/presets.json) as the list of date ranges |
| `make test` / `make check` | Go and web tests / plus lint, typecheck and build |
| `make e2e` / `make k6-smoke` | Browser tests / API smoke test through the gateway (needs `make up`) |
| `make ci` / `make ci-full` | Exactly what the CI pipeline runs: all services, the web app and the image builds / plus the browser and k6 tests (needs Pagila and k6) |

## CI/CD

`.github/workflows/ci.yml` runs on every pull request, every push to `main` and every `v*` tag. Its jobs call the same `make` targets you run locally, so `make ci` passing means the pipeline should pass:

| Job | Runs | Locally |
|---|---|---|
| `go` (one per service) | `gofmt`, `go vet`, tests with the coverage floors, tests with the race detector, a build | `make ci-go-analytics` (or `stores`, `ranges`, `gateway`, `redis`) |
| `web` | lint, typecheck, tests with the coverage floors, the production build | `make ci-web` |
| `docker` | builds every image | `make ci-docker` |
| `e2e`, `k6` | the Playwright suite and the k6 smoke test, against a Pagila service | `make ci-full` |
| `ci-ok` | fails if any job above failed. Require this one check in branch protection | n/a |
| `publish` | after `ci-ok`, on `main` and version tags only: pushes the images to `ghcr.io/<owner>/store-operation-<service>`, tagged with the commit SHA, `latest` (main) and the version (tags such as `v1.2.0`) | n/a |


```mermaid
## Architecture
flowchart LR
    user([Staff member]) --> web["web<br/>React · TypeScript · Vite"]
    web -- "/api/*" --> gateway["gateway :8080<br/>routing · request id · access log<br/>rate limit · timeouts"]
    gateway -- "/api/stores" --> stores["stores :8081"]
    gateway -- "/api/presets" --> ranges["ranges :8083"]
    gateway -- "everything else" --> analytics["analytics :8084"]
    analytics -. "GET /api/presets<br/>30 s copy, built-in fallback" .-> ranges
    stores --> pg[("Pagila<br/>PostgreSQL")]
    analytics --> pg
    stores -. "cache" .-> redissvc["redis service :8082"]
    analytics -. "cache" .-> redissvc
    ranges -. "date-range list" .-> redissvc
    redissvc --> redis[("Redis")]
```

| Service | Folder | Owns |
|---|---|---|
| **gateway** | `app/gateway` | The only thing the browser talks to. Routes by path prefix (`ROUTES`), adds `X-Request-Id`, writes the access log, rate-limits each client IP, times out slow services (`502`, `504`). Standard library only |
| **analytics** | `app/analytics` | `/api/kpis`, `/api/revenue`, `/api/top/films`, `/api/meta`, `/healthz`. Queries Pagila; asks `ranges` which windows are allowed |
| **stores** | `app/stores` | `/api/stores`: the stores that hold inventory, with their details. Queries Pagila |
| **ranges** | `app/ranges` | `/api/presets`: the date ranges the dashboard offers, from a list in Redis or the built-in one. No database |
| **redis service** | `app/redis` | A small HTTP key-value API in front of Redis. Reachable only on the Docker network |
| **web** | `app/web` | The React app. In Docker, nginx serves it and proxies `/api` to the gateway |

How the pieces behave together:

- **Independent services.** Each service is its own Go module with its own Dockerfile, tests and CI workflow, and imports no code from another (a test in each enforces it). The three data services share the Pagila database but split the endpoints and code.
- **Hexagonal inside.** Each service keeps a pure `domain`, `app` use cases that talk to ports, and `adapters` (HTTP, Postgres, cache) that plug into them. Dependencies point inward, and an architecture test fails CI if they do not.
- **Read-only.** The app adds no tables, views or migrations. Database connections are opened with `default_transaction_read_only=on`.
- **Optional cache.** With `CACHE_URL` set, the heavy queries (`analytics`) and store details (`stores`) are served from Redis through the redis service, with identical responses. If that service is down they answer from a small in-memory copy until each entry expires, then from Postgres.
- **Date ranges are data.** `ranges` turns a list of definitions into concrete windows measured from the as-of date. `analytics` refuses any window that is not on that list. If `ranges` is down, `analytics` keeps using the last list it got, or the built-in one, so only `/api/presets` itself fails (`502`).
- **Request path.** Browser → gateway (id, rate limit, log) → the owning service → Pagila, optionally through the cache. The same `X-Request-Id` appears in the gateway's and the services' logs.

## Database

### Web app

`app/web` is React 19 + TypeScript (Vite), with TanStack Query for server state, React Router for the URL, Recharts for the chart and Tailwind CSS v4 for styling. Like the backend, dependencies point inward, and a test (`src/archtest.test.ts`, run by `npm test`) fails if they do not.

```mermaid
flowchart LR
    pages["pages<br/>route-level screens"] --> features
    features["features<br/>containers: call hooks, apply rules,<br/>map data to props"] --> components
    features --> hooks["hooks<br/>one react-query hook per file<br/>+ URL state"]
    hooks --> api["api<br/>typed client, one function per endpoint"]
    features --> lib["lib<br/>formatting, filters"]
    hooks --> lib
    components["components<br/>pure UI: props in, JSX out"] --> lib
    components --> chart["components/libs/recharts<br/>the only door to the chart library"]
    lib --> api
```

```
app/web/src/
  routes.tsx, App.tsx, main.tsx   route table ("/" is the dashboard, anything else 404) and providers
  pages/                          thin route-level screens (HomePage, NotFoundPage)
  features/overview/              OverviewPage, the one container, plus the mapping helpers (kpis, films, storeDetails, pivot)
  hooks/                          useMeta, useRanges, useStores, useKpis, useRevenue, useTopFilms, useFilters, queryKeys
  components/                     one folder per component: Name.tsx, index.ts and Name.test.tsx
    libs/recharts/                wrappers for the chart pieces we use, with our design defaults
  api/                            typed client for the gateway's endpoints
  lib/                            format, filters, useBreakpoint
  index.css                       Tailwind import, light and dark colour tokens, a small base layer
```

| Layer | May import |
|---|---|
| `api` | itself |
| `lib` | `api` (types only) |
| `hooks` | `api`, `lib` |
| `components` | other components and `lib`. Never `hooks`, `features`, `pages`, `api`, react-router or react-query |
| `features` | `components`, `hooks`, `lib`, `api` |
| `pages` | `features` and everything below |
| `recharts` | only `components/libs/recharts` may import it |

- **Pure UI.** A component takes plain props and knows nothing about films or filters. `KpiGrid` renders cards; `features/overview/kpis.ts` decides what they say. `OverviewPage` is the only place that calls the query hooks.
- **Data flow.** The browser calls `/api/*` on the gateway only (Vite proxies it in dev, nginx in Docker). Each endpoint has one hook in its own file; cache keys live in `queryKeys.ts`. The date ranges and stores are loaded once and kept, and the figures refetch when the filters change while the previous data stays on screen.
- **State lives in the URL.** `useFilters` reads and writes the range, store and grouping as query parameters, so a reload or a shared link restores the same view. A link to a range that has left the list falls back to the default range. There is no free date entry: the UI offers only what `GET /api/presets` returns.
- **Each section stands alone.** `AsyncSection` handles loading, empty and error (with Retry) for one query, so one failing endpoint never takes down the page.
- **One door to the chart library.** Everything charts need comes from `components/libs/recharts`, which sets the theme defaults. Swapping the library is a one-folder change. The chart is lazy-loaded.
- **Styling.** Tailwind utilities next to the markup, with no custom class names or `@apply`. Light and dark colours are CSS variables exposed as utilities (`bg-surface`, `text-ink`, `text-muted`, `border-line`, and so on); dark mode follows the OS setting, so components never write `dark:` variants. State is styled from ARIA and data attributes, and tests read those attributes, never utility classes.
- **Responsive.** Mobile-first with two breakpoints, `sm` (640 px) and `lg` (1024 px). On a phone the filters fold behind a **Filters** button, the key figures are 2 × 2 and the sections stack; on desktop the filters are one row and revenue sits beside top films. Touch targets are 44 px. CSS does the sizing; only the chart's pixel height reads the viewport (`useBreakpoint`).
- **Accessibility.** Keyboard-reachable controls, trends shown by arrow and text (not colour alone), and the chart has a text summary and a "show data as a table" view.

The app reads the [Pagila](https://github.com/devrimgunduz/pagila) sample database (PostgreSQL 17, `public` schema) and adds nothing to it. Row counts are from the sample data.

### Entity relationship diagram

Solid lines are real foreign keys. Dotted lines are relationships that exist in the data but have **no foreign key constraint** (see [Things to know](#things-to-know)).

```mermaid
erDiagram
    country ||--o{ city : "country_id"
    city ||--o{ address : "city_id"
    address ||--o{ store : "address_id"
    address ||--o{ customer : "address_id"
    address ||--o{ staff : "address_id"
    store ||--o{ customer : "store_id"
    store ||--o{ staff : "store_id"
    store ||--o{ inventory : "store_id"
    staff ||..o| store : "manager_staff_id (no FK)"
    film ||--o{ inventory : "film_id"
    inventory ||--o{ rental : "inventory_id"
    customer ||--o{ rental : "customer_id"
    staff ||--o{ rental : "staff_id"
    rental ||..o{ payment : "rental_id (no FK)"
    customer ||..o{ payment : "customer_id (no FK)"
    staff ||..o{ payment : "staff_id (no FK)"
    language ||--o{ film : "language_id"
    language |o--o{ film : "original_language_id"
    film ||--o{ film_actor : "film_id"
    actor ||--o{ film_actor : "actor_id"
    film ||--o{ film_category : "film_id"
    category ||--o{ film_category : "category_id"

    country {
        int country_id PK
        text country
        timestamptz last_update
    }
    city {
        int city_id PK
        text city
        int country_id FK
        timestamptz last_update
    }
    address {
        int address_id PK
        text address
        text address2
        text district
        int city_id FK
        text postal_code
        text phone
        timestamptz last_update
    }
    store {
        int store_id PK
        int manager_staff_id
        int address_id FK
        timestamptz last_update
    }
    staff {
        int staff_id PK
        text first_name
        text last_name
        int address_id FK
        text email
        int store_id FK
        boolean active
        text username
        text password
        timestamptz last_update
        bytea picture
    }
    customer {
        int customer_id PK
        int store_id FK
        text first_name
        text last_name
        text email
        int address_id FK
        boolean activebool
        date create_date
        timestamptz last_update
        int active
    }
    language {
        int language_id PK
        char name
        timestamptz last_update
    }
    film {
        int film_id PK
        text title
        text description
        int release_year
        int language_id FK
        int original_language_id FK
        smallint rental_duration
        numeric rental_rate
        smallint length
        numeric replacement_cost
        mpaa_rating rating
        timestamptz last_update
        text_array special_features
        tsvector fulltext
    }
    actor {
        int actor_id PK
        text first_name
        text last_name
        timestamptz last_update
    }
    category {
        int category_id PK
        text name
        timestamptz last_update
    }
    film_actor {
        int actor_id PK
        int film_id PK
        timestamptz last_update
    }
    film_category {
        int film_id PK
        int category_id PK
        timestamptz last_update
    }
    inventory {
        int inventory_id PK
        int film_id FK
        int store_id FK
        timestamptz last_update
    }
    rental {
        int rental_id PK
        timestamptz rental_date
        int inventory_id FK
        int customer_id FK
        timestamptz return_date
        int staff_id FK
        timestamptz last_update
    }
    payment {
        timestamptz payment_date PK
        int payment_id PK
        int customer_id
        int staff_id
        int rental_id
        numeric amount
    }
```

### What the services read

Eight tables are used; everything else in Pagila is untouched. Money is attributed to a store through `payment → rental → inventory.store_id`, the same rule as Pagila's `sales_by_store` view.

| Table | Rows | Used for | Service |
|---|---|---|---|
| `payment` | 16,049 | Revenue, payment counts, top films (filtered on `payment_date`) | analytics |
| `rental` | 16,044 | Rental counts and distinct customers (by `rental_date`), and the link from a payment to an inventory item | analytics |
| `inventory` | 4,581 | The link from a rental to a store and a film; also a store's item count | analytics, stores |
| `film` | 1,000 | Film titles for top films | analytics |
| `store` | 500 (only 2 stocked) | The store list | stores |
| `address`, `city`, `country` | 603, 600, 109 | A store's `City, Country` label, address, district, postal code and phone | stores |
| `staff` | 2 | A store's manager (`store.manager_staff_id`) | stores |
| `customer` | 599 | A store's active customers (`activebool`) | stores |

The other tables (`actor`, `category`, `language`, `film_actor`, `film_category`) are not queried; they are in the diagram to keep it complete. The `ranges` service uses no database.

### Things to know

- **`payment` has no foreign keys** to `rental`, `customer` or `staff`, on the parent table or any partition. The relationship is by convention only, which is why the diagram draws it dotted. `store.manager_staff_id` has none either, because `staff.store_id` points back at `store`.
- **`payment` is partitioned by month** on `payment_date`. The queries filter on the raw column so Postgres reads only the months a window touches; wrapping it in a function would turn that off. There is no index on `payment.rental_id`, and the app cannot add one because it is read-only.
- **A rental can have more than one payment** (16,049 payments for 16,044 rentals), so top films counts `DISTINCT rental_id` for its rentals figure.
- **Only stores 1 and 2 hold inventory.** The other 498 can never have sales, so `/api/stores` leaves them out.

## Configuration

All settings are environment variables and all are optional; see [.env.example](.env.example). The Docker stack sets what it needs in [infra/docker-compose.yml](infra/docker-compose.yml).

**analytics** (`:8084`)

| Variable | Default | Purpose |
|---|---|---|
| `ADDR` | `:8084` | Listen address |
| `DATABASE_URL` | `postgres://postgres:123456@localhost:5432/postgres` | Pagila connection string |
| `ASOF_DATE` | `2022-07-27` | The dashboard's "today" (Pagila's payments end then), served at `/api/meta` |
| `RANGES_URL` | `http://localhost:8083` | The ranges service. Its answer is kept for 30 s; the last good list, or the built-in one, is used when it is down |
| `CACHE_URL` | *(unset)* | The redis service, for example `http://localhost:8082`. Setting it turns the cache on |
| `CACHE_TTL` | `10m` | How long a cached query result stays valid |

**stores** (`:8081`): `ADDR` (`:8081`), `DATABASE_URL` and `CACHE_URL`, as above. Store details are cached for an hour.

**ranges** (`:8083`): `ADDR` (`:8083`), `ASOF_DATE` and `CACHE_URL`, as above. With no `CACHE_URL` it always serves the built-in list.

**gateway** (`:8080`)

| Variable | Default | Purpose |
|---|---|---|
| `ADDR` | `:8080` | Listen address |
| `UPSTREAM_URL` | `http://localhost:8084` | The service that gets `/api/*` and `/healthz` unless a route claims them (analytics in Docker) |
| `ROUTES` | none | Comma-separated `/prefix=http://host:port` pairs, such as `/api/presets=http://ranges:8083`. A prefix matches that exact path and everything below it; the longest match wins |
| `RATE_LIMIT_RPS` | `50` | Requests per second for each client IP. `0` turns the limit off |
| `RATE_LIMIT_BURST` | `100` | Burst size |
| `UPSTREAM_DIAL_TIMEOUT` | `2s` | Time to connect to a service |
| `UPSTREAM_HEADER_TIMEOUT` | `10s` | Time for a service to start answering |

**redis service** (`:8082`): `ADDR` (`:8082`) and `REDIS_URL` (`redis://localhost:6379`).

<a id="date-ranges"></a>
**Changing the date ranges.** The list lives in Redis under `config:presets:v1` as JSON, with no expiry. The ranges service reads it at start and every 60 seconds, and uses the built-in list when the key is missing or the stored list is unusable.

```sh
make seed-presets    # store infra/presets.json (the built-in list; edit it first)
docker compose -f infra/docker-compose.yml exec redis redis-cli del config:presets:v1   # back to the built-in list
```

Each entry is `{ "id", "label", "group", "kind", ... }`, with exactly one `"default": true`, 1 to 50 entries and unique ids. `kind` is one of:

| `kind` | Window |
|---|---|
| `days` | The `days` days ending at the as-of date (1 to 366) |
| `month_to_date` / `quarter_to_date` | The 1st of the as-of date's month / quarter up to the as-of date |
| `last_month` / `last_quarter` | The whole month / quarter before the as-of date's |
| `fixed` | `from` to `to`, both `YYYY-MM-DD`, at most 366 days |

## API reference

A small read-only JSON API. Every endpoint is `GET`, needs no authentication, and never changes data. The examples below are real responses from the Pagila sample data (the as-of date is 2022-07-27) and run against the default `http://localhost:8080`, which is the gateway. The gateway forwards each route to the service that owns it (see [Architecture](#architecture)) and changes nothing in the answers.

| Endpoint | What it returns |
|---|---|
| [`GET /healthz`](#get-healthz) | Whether the analytics service can reach the database |
| [`GET /api/meta`](#get-apimeta) | The dashboard's "today" |
| [`GET /api/presets`](#get-apipresets) | The date ranges the other endpoints accept |
| [`GET /api/stores`](#get-apistores) | The stores that hold inventory, with their details |
| [`GET /api/kpis`](#get-apikpis) | Four headline figures, each compared with the previous period |
| [`GET /api/revenue`](#get-apirevenue) | Revenue over time, one point per bucket and store |
| [`GET /api/top/films`](#get-apitopfilms) | The films that earned the most |

### Conventions

**Date window.** `/api/kpis`, `/api/revenue` and `/api/top/films` all look at a window of whole days. The window must be one of the ranges [`GET /api/presets`](#get-apipresets) lists.

| Parameter | Type | Default | Rules |
|---|---|---|---|
| `from` | date, `YYYY-MM-DD` | the default range's first day | First day of the window, inclusive. Send it together with `to` |
| `to` | date, `YYYY-MM-DD` | the default range's last day | Last day of the window, inclusive. Send it together with `from` |
| `store` | integer, 0 or more | all stores | Only count activity for this store |

- Days are UTC and both ends are inclusive, so `from=2022-07-26&to=2022-07-27` is two days.
- With no dates the window is the **default range**, "Last 30 days" unless the list says otherwise (`2022-06-28` to `2022-07-27`).
- **Only listed ranges are accepted.** Send both dates exactly as `/api/presets` gives them. One date alone, a window that is not in the list (even one day off), or a malformed date is a `400`. There is no free date entry.
- The as-of date exists because Pagila's payments stop on 2022-07-27, so "the last 30 days" needs a fixed anchor. It is in [`/api/meta`](#get-apimeta); the ranges are already measured from it.

**Store attribution.** Money and rentals are attributed to a store through `payment → rental → inventory.store_id`, the same rule as Pagila's `sales_by_store` view. Only stores 1 and 2 hold inventory. An unknown store id is not an error: it simply matches nothing, so figures are 0 and lists are empty.

**Numbers.** Amounts are plain numbers rounded to two decimals (the web app formats them as USD). Percentages are plain numbers too, so `-2.43` means -2.43%.

**Request id.** Every response carries an `X-Request-Id` header, set by the gateway. Send your own (up to 64 letters, digits, `.`, `_` or `-`) and it is kept; otherwise one is generated. The same id appears in the gateway's and the services' logs.

**Rate limit.** Each client address may make `RATE_LIMIT_RPS` requests per second (50 by default) with a burst of `RATE_LIMIT_BURST` (100). Beyond that the gateway answers `429` until a token is free; see [Errors](#errors).

**Caching.** With `CACHE_URL` set, the figures behind `/api/kpis`, `/api/revenue` and `/api/top/films` can be up to `CACHE_TTL` (10 minutes by default) old, and `/api/stores` up to an hour. The responses are byte-identical to the uncached ones, and there is no cache header. `/api/meta`, `/api/presets` and `/healthz` are not cached per request (the ranges service keeps the list in memory and refreshes it every minute; see [Date ranges](#date-ranges)). The data is static (as-of 2022), so staleness is not visible here; live data would need short TTLs or explicit invalidation.

**Content types.** Successful responses are `application/json`. Errors from the services are `application/problem+json` (see [Errors](#errors)).

### `GET /healthz`

Checks that the analytics service can reach the database. Use it for liveness and readiness probes. Every service has its own `/healthz`; through the gateway this one is analytics'.

```sh
curl http://localhost:8080/healthz
```

```json
{ "status": "ok" }
```

If the database cannot be reached, the response is `503` with a [problem body](#errors) whose `detail` is `database unreachable`.

### `GET /api/meta`

Returns the as-of date, the dashboard's "today". The web app reads it once so the date lives in one place, set by the `ASOF_DATE` environment variable.

```sh
curl http://localhost:8080/api/meta
```

```json
{ "asOf": "2022-07-27" }
```

### `GET /api/presets`

Lists the date ranges the dashboard offers, in the order to show them. These are the only windows the analytics endpoints accept, and the web app builds its range buttons and menu from this list. Takes no parameters.

```sh
curl http://localhost:8080/api/presets
```

```json
[
  { "id": "last-7",  "label": "Last 7 days",  "group": "Recent", "from": "2022-07-21", "to": "2022-07-27", "default": false },
  { "id": "last-30", "label": "Last 30 days", "group": "Recent", "from": "2022-06-28", "to": "2022-07-27", "default": true },
  { "id": "2022-q1", "label": "Q1 2022",      "group": "Fixed",  "from": "2022-01-01", "to": "2022-03-31", "default": false }
]
```

(18 entries by default; three are shown.)

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Stable name of the range |
| `label` | string | Text to show |
| `group` | string | The menu heading it is listed under |
| `from`, `to` | date, `YYYY-MM-DD` | The window, both ends inclusive, to pass as `from` and `to` |
| `default` | boolean | Exactly one entry is the default, used when no dates are sent |

The built-in list is **Recent** (last 7, 14, 30, 60, 90 and 180 days), **Periods** (month to date, last month, quarter to date, last quarter) and **Fixed** (February to June 2022, Q1 2022, Q2 2022, and "All data", 2022-01-23 to 2022-07-27). The relative ones move with the as-of date. See [Changing the date ranges](#date-ranges) for where the list comes from.

### `GET /api/stores`

Lists the stores that hold inventory, in id order, with their details. A store with no inventory (Pagila has 500 stores, only 2 stocked) is left out, because it would never have sales. Takes no parameters.

```sh
curl http://localhost:8080/api/stores
```

```json
[
  {
    "id": 1, "label": "Boksburg, South Africa",
    "manager": "Warner Hudson", "phone": "490211944645",
    "address": "569 Baicheng Lane", "district": "Gauteng", "postalCode": "60304",
    "city": "Boksburg", "country": "South Africa",
    "inventoryItems": 2270, "activeCustomers": 326
  },
  {
    "id": 2, "label": "Hamilton, New Zealand",
    "manager": "Lavone O'Reilly", "phone": "657282285970",
    "address": "478 Joliet Way", "district": "Hamilton", "postalCode": "77948",
    "city": "Hamilton", "country": "New Zealand",
    "inventoryItems": 2311, "activeCustomers": 273
  }
]
```

| Field | Type | Meaning |
|---|---|---|
| `id` | integer | Store id, the value to pass as `store` elsewhere |
| `label` | string | `City, Country`, ready to show |
| `manager` | string | The manager's first and last name |
| `phone`, `address`, `district`, `postalCode` | string | As stored in Pagila's `address` table (empty when Pagila has none) |
| `city`, `country` | string | The parts of `label` |
| `inventoryItems` | integer | Items in the store's inventory (computed) |
| `activeCustomers` | integer | Customers of the store whose account is active (computed) |

### `GET /api/kpis`

Four headline figures for the window, each compared with the **previous window**: the window of the same length that ends the day before this one starts. For `2022-06-28` to `2022-07-27` (30 days), the previous window is `2022-05-29` to `2022-06-27`.

**Parameters:** `from`, `to`, `store` (see [Conventions](#conventions)).

```sh
curl 'http://localhost:8080/api/kpis?from=2022-06-28&to=2022-07-27'
```

```json
{
  "revenue":         { "value": 10707.22, "previous": 10973.36, "deltaPct": -2.43 },
  "rentals":         { "value": 4158,     "previous": 2778,     "deltaPct": 49.68 },
  "activeCustomers": { "value": 598,      "previous": 597,      "deltaPct": 0.17 },
  "avgPayment":      { "value": 4.15,     "previous": 4.12,     "deltaPct": 0.73 }
}
```

| Figure | How it is counted |
|---|---|
| `revenue` | Sum of payments made in the window |
| `rentals` | Rentals that started in the window |
| `activeCustomers` | Distinct customers who started a rental in the window |
| `avgPayment` | `revenue` divided by the number of payments, or 0 when there were none |

Each figure has the same three fields:

| Field | Type | Meaning |
|---|---|---|
| `value` | number | The figure for the requested window |
| `previous` | number | The same figure for the previous window |
| `deltaPct` | number or `null` | Percentage change from `previous` to `value`, to two decimals. **`null` when `previous` is 0**, because there is nothing to compare against |

Filtered to one store, with `store=1`, the same window gives the store's share:

```json
{
  "revenue":         { "value": 5434.11, "previous": 5532.99, "deltaPct": -1.79 },
  "rentals":         { "value": 2040,    "previous": 1348,    "deltaPct": 51.34 },
  "activeCustomers": { "value": 576,     "previous": 548,     "deltaPct": 5.11 },
  "avgPayment":      { "value": 4.22,    "previous": 4.25,    "deltaPct": -0.71 }
}
```

A window with no activity is not an error. Every value is 0 and every `deltaPct` is `null`.

### `GET /api/revenue`

Revenue over time as a flat list with **one point per time bucket and store**, ordered by bucket and then store. The list is **zero-filled**: a bucket with no sales still appears with `revenue: 0`, so a chart has no gaps to bridge.

**Parameters:** `from`, `to`, `store`, and:

| Parameter | Type | Default | Rules |
|---|---|---|---|
| `granularity` | `day`, `week` or `month` | `day` | Size of each bucket |

```sh
curl 'http://localhost:8080/api/revenue?from=2022-07-26&to=2022-07-27'
```

```json
[
  { "bucket": "2022-07-26", "storeId": 1, "revenue": 167.57 },
  { "bucket": "2022-07-26", "storeId": 2, "revenue": 134.64 },
  { "bucket": "2022-07-27", "storeId": 1, "revenue": 96.83 },
  { "bucket": "2022-07-27", "storeId": 2, "revenue": 85.77 }
]
```

| Field | Type | Meaning |
|---|---|---|
| `bucket` | date, `YYYY-MM-DD` | The **first day** of the bucket |
| `storeId` | integer | The store the revenue belongs to |
| `revenue` | number | Payments made in the bucket |

**Buckets start on calendar boundaries.** A `week` bucket starts on a Monday and a `month` bucket on the 1st, so the first bucket can start *before* `from`. Only payments inside the requested window are counted. For `from=2022-03-15&to=2022-04-10&granularity=month`:

```json
[
  { "bucket": "2022-03-01", "storeId": 1, "revenue": 3008.99 },
  { "bucket": "2022-03-01", "storeId": 2, "revenue": 3238.13 },
  { "bucket": "2022-04-01", "storeId": 1, "revenue": 1795.66 },
  { "bucket": "2022-04-01", "storeId": 2, "revenue": 1858.52 }
]
```

With `store=2` only that store's points are returned. An unknown store gives an empty list, `[]`. At every granularity, the points add up to the same total as `revenue.value` from [`/api/kpis`](#get-apikpis) for the same window.

### `GET /api/top/films`

The films that earned the most in the window, highest revenue first. Ties are broken by `filmId`, so the order is stable.

**Parameters:** `from`, `to`, `store`, and:

| Parameter | Type | Default | Rules |
|---|---|---|---|
| `limit` | integer | 10 | 1 to 50 |

```sh
curl 'http://localhost:8080/api/top/films?limit=3&store=1'
```

```json
[
  { "filmId": 327, "title": "FOOL MOCKINGBIRD",  "rentals": 6, "revenue": 41.94 },
  { "filmId": 879, "title": "TELEGRAPH VOYAGE",  "rentals": 5, "revenue": 38.95 },
  { "filmId": 202, "title": "DADDY PITTSBURGH",  "rentals": 6, "revenue": 37.94 }
]
```

| Field | Type | Meaning |
|---|---|---|
| `filmId` | integer | Pagila's film id |
| `title` | string | Film title (Pagila stores these in capitals) |
| `rentals` | integer | Rentals of this film that had a payment in the window |
| `revenue` | number | Payments for this film in the window |

A window with no payments returns an empty list, `[]`.

### Errors

Bad input and server failures return a body in the [RFC 7807](https://www.rfc-editor.org/rfc/rfc7807) problem style, with `Content-Type: application/problem+json`:

```json
{ "title": "Bad Request", "status": 400, "detail": "from: want YYYY-MM-DD" }
```

| Status | When | `detail` |
|---|---|---|
| `400` | A date is not `YYYY-MM-DD` | `from: want YYYY-MM-DD` or `to: want YYYY-MM-DD` |
| `400` | Only one of `from` and `to` is sent | `from and to must be sent together` |
| `400` | The window is not one of the offered ranges | `from and to: not one of the offered ranges (see GET /api/presets)` |
| `400` | `store` is not a non-negative integer | `store: want a non-negative integer` |
| `400` | `granularity` is not `day`, `week` or `month` | `granularity: want day, week or month` |
| `400` | `limit` is outside 1 to 50 | `limit: want 1..50` |
| `500` | A query failed | `internal error`. The real cause is logged on the server and never returned |
| `503` | `/healthz` cannot reach the database | `database unreachable` |
| `429` | The gateway's per-client rate limit is exceeded. The `Retry-After` header says how many seconds to wait | `rate limit exceeded` |
| `502` | The gateway cannot reach the service that owns the route (for example `/api/presets` while the ranges service is down) | `upstream unavailable` |
| `504` | The service did not start answering within `UPSTREAM_HEADER_TIMEOUT` (10 s) | `upstream timed out` |

The `429`, `502` and `504` rows come from the gateway; the rest come from the services and pass through the gateway unchanged.

Two cases come from Go's router, not from the app, so their bodies are plain text rather than problem JSON: `405 Method Not Allowed` for any method other than `GET` (the `Allow` header lists what is accepted), and `404 Not Found` for an unknown path. The gateway only forwards `/api/*` and `/healthz`, so a path outside those gets the gateway's own plain-text `404` and an unknown path under `/api/` gets analytics'.
