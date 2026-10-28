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
- **Optional Redis cache.** Set `REDIS_URL` and the heavy queries are served from Redis within a TTL, with identical responses. If Redis is down the API keeps working from Postgres.

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
| `make redis-up` / `make redis-down` | Starts or stops Redis in Docker for the optional cache (see below) |

**Optional query cache.** Start Redis, then point the API at it:

```sh
make redis-up                                     # or REDIS_PORT=6380 make redis-up if 6379 is taken
REDIS_URL=redis://localhost:6379 make dev
```

### Configuration

All settings are environment variables and all are optional. See [.env.example](.env.example).

| Variable | Default | Purpose |
|---|---|---|
| `ADDR` | `:8080` | API listen address |
| `DATABASE_URL` | `postgres://postgres:123456@localhost:5432/postgres` | Pagila connection string |
| `ASOF_DATE` | `2022-07-27` | The "today" for default date ranges |
| `REDIS_URL` | *(unset)* | Turns the query cache on, for example `redis://localhost:6379`. Unset means no cache |
| `CACHE_TTL` | `10m` | How long a cached query result stays valid |
| `TEST_DATABASE_URL` | same as above | Database used by integration tests |
| `REQUIRE_DB` | unset | If set, integration tests fail instead of skipping when the database is unreachable |

Pagila's payments end on 2022-07-27, so "the last 30 days" needs a fixed anchor. The API exposes it at `GET /api/meta` and the UI reads it from there, so there is one source of truth.

## API reference

A small read-only JSON API. Every endpoint is `GET`, needs no authentication, and never changes data. The examples below are real responses from the Pagila sample data (the as-of date is 2022-07-27) and run against the default `http://localhost:8080`.

| Endpoint | What it returns |
|---|---|
| [`GET /healthz`](#get-healthz) | Whether the API can reach the database |
| [`GET /api/meta`](#get-apimeta) | The dashboard's "today" |
| [`GET /api/stores`](#get-apistores) | The stores that hold inventory |
| [`GET /api/kpis`](#get-apikpis) | Four headline figures, each compared with the previous period |
| [`GET /api/revenue`](#get-apirevenue) | Revenue over time, one point per bucket and store |
| [`GET /api/top/films`](#get-apitopfilms) | The films that earned the most |

### Conventions

**Date window.** `/api/kpis`, `/api/revenue` and `/api/top/films` all look at a window of whole days.

| Parameter | Type | Default | Rules |
|---|---|---|---|
| `from` | date, `YYYY-MM-DD` | 29 days before `to` | First day of the window, inclusive |
| `to` | date, `YYYY-MM-DD` | the as-of date (`ASOF_DATE`, 2022-07-27) | Last day of the window, inclusive |
| `store` | integer, 0 or more | all stores | Only count activity for this store |

- Days are UTC and both ends are inclusive, so `from=2022-07-26&to=2022-07-27` is two days.
- With no dates the window is the **30 days ending at the as-of date**. Give only `to` and the window is the 30 days ending there.
- `from` must not be after `to`, and the window can be at most **366 days**.
- The as-of date exists because Pagila's payments stop on 2022-07-27, so "the last 30 days" needs a fixed anchor. Read it from [`/api/meta`](#get-apimeta) instead of hard-coding it.

**Store attribution.** Money and rentals are attributed to a store through `payment → rental → inventory.store_id`, the same rule as Pagila's `sales_by_store` view. Only stores 1 and 2 hold inventory. An unknown store id is not an error: it simply matches nothing, so figures are 0 and lists are empty.

**Numbers.** Amounts are plain numbers rounded to two decimals (the web app formats them as USD). Percentages are plain numbers too, so `-2.43` means -2.43%.

**Caching.** With `REDIS_URL` set, the figures behind `/api/kpis`, `/api/revenue` and `/api/top/films` can be up to `CACHE_TTL` (10 minutes by default) old. The responses are byte-identical to the uncached ones, and there is no cache header. `/api/stores`, `/api/meta` and `/healthz` are never cached. The data is static (as-of 2022), so staleness is not visible here; live data would need short TTLs or explicit invalidation.

**Content types.** Successful responses are `application/json`. Errors from the API are `application/problem+json` (see [Errors](#errors)).

### `GET /healthz`

Checks that the API can reach the database. Use it for liveness and readiness probes.

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

### `GET /api/stores`

Lists the stores that hold inventory, in id order. A store with no inventory (Pagila has 500 stores, only 2 stocked) is left out, because it would never have sales. Takes no parameters.

```sh
curl http://localhost:8080/api/stores
```

```json
[
  { "id": 1, "label": "Boksburg, South Africa" },
  { "id": 2, "label": "Hamilton, New Zealand" }
]
```

| Field | Type | Meaning |
|---|---|---|
| `id` | integer | Store id, the value to pass as `store` elsewhere |
| `label` | string | `City, Country`, ready to show |

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
| `400` | `from` is after `to` | `from must not be after to` |
| `400` | The window is longer than 366 days | `range must not exceed 366 days` |
| `400` | `store` is not a non-negative integer | `store: want a non-negative integer` |
| `400` | `granularity` is not `day`, `week` or `month` | `granularity: want day, week or month` |
| `400` | `limit` is outside 1 to 50 | `limit: want 1..50` |
| `500` | A query failed | `internal error`. The real cause is logged on the server and never returned |
| `503` | `/healthz` cannot reach the database | `database unreachable` |

Two cases come from Go's router, not from the API, so their bodies are plain text rather than problem JSON: `405 Method Not Allowed` for any method other than `GET` (the `Allow` header lists what is accepted), and `404 Not Found` for an unknown path.

## Database

The app reads the [Pagila](https://github.com/devrimgunduz/pagila) sample database (PostgreSQL 17, `public` schema) and adds nothing to it. This section documents the schema as it is, and which parts the app uses. Row counts are from the sample data.

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

### What the app reads

The API uses six tables. Everything else in Pagila is untouched.

| Table | Rows | Used for |
|---|---|---|
| `payment` | 16,049 | Revenue, payment counts, top films. The date filter runs on `payment_date`. |
| `rental` | 16,044 | Rental counts and distinct customers (by `rental_date`), and the link from a payment to an inventory item. |
| `inventory` | 4,581 | The link from a rental to a **store** and a **film**. This is how money is attributed to a store. |
| `film` | 1,000 | Film titles for top films. |
| `store` | 500 (only 2 stocked) | The store list. |
| `address`, `city`, `country` | 603, 600, 109 | The `City, Country` label of a store. |

The remaining tables (`customer`, `staff`, `actor`, `category`, `language`, `film_actor`, `film_category`) are not queried. They are shown above so the diagram is complete.

### How the queries join

Store attribution is a fixed path, the same rule as Pagila's `sales_by_store` view:

```
payment ──rental_id──▶ rental ──inventory_id──▶ inventory ──store_id──▶ store
                                                    └──film_id──▶ film
```

| Endpoint | Repo method | Tables read | Date column |
|---|---|---|---|
| `/api/stores` | `Stores` | `store`, `address`, `city`, `country`, `inventory` | none |
| `/api/kpis` | `Totals` | `payment`, `rental`, `inventory` | `payment_date` for revenue, `rental_date` for rentals and customers |
| `/api/revenue` | `Revenue` | `payment`, `rental`, `inventory` | `payment_date` |
| `/api/top/films` | `TopFilms` | `payment`, `rental`, `inventory`, `film` | `payment_date` |

### Partitioning and indexes

- **`payment` is range-partitioned by month** on `payment_date`: `payment_p2022_01` to `payment_p2022_07`, each with its own primary key `(payment_date, payment_id)`. The queries filter on the raw `payment_date` column, so Postgres reads only the months a window touches. Wrapping that column in a function would turn off this pruning.
- **`rental`** has `(rental_date, inventory_id, customer_id)` (unique), which serves the date filter as an index-only scan, and `(inventory_id)`.
- **`inventory`** has `(store_id, film_id)`.
- **`film`** has the primary key, `title`, and a GiST index on `fulltext`.
- **`payment` has no index on `rental_id`**, so a payment-to-rental join reads the whole `rental` table and hashes it. That is the cheapest plan at this size (each endpoint runs in a few milliseconds). The app cannot add an index because it is read-only.

### Things to know

- **`payment` has no foreign keys** to `rental`, `customer` or `staff`, on the parent table or any partition. The relationship is by convention only, which is why the diagram draws it dotted.
- **`store.manager_staff_id` has no foreign key**, because `staff.store_id` points back at `store` and the two would be circular.
- **A rental can have more than one payment** (16,049 payments for 16,044 rentals). Top films therefore counts `DISTINCT rental_id` for its rentals figure.
- **Only stores 1 and 2 hold inventory.** The other 498 stores can never have sales, so `/api/stores` leaves them out.
- **Not part of the model:** the views (`sales_by_store`, `sales_by_film_category`, `film_list`, `customer_list`, `staff_list`, `actor_info`, `nicer_but_slower_film_list`), the materialised view `rental_by_category`, and two unrelated tables loaded alongside Pagila (`packages_apt_postgresql_org`, `packages_yum_postgresql_org`).

## Architecture

### System at a glance

```mermaid
flowchart LR
    user([Staff member]) --> web
    subgraph browser [Browser]
        web["web<br/>React 19 · TypeScript · Vite<br/>Tailwind CSS · TanStack Query"]
    end
    subgraph server [Go API · one deployable]
        direction TB
        api["cmd/api<br/>composition root"]
        analytics["analytics context<br/>domain · app · adapters"]
        platform["platform<br/>db · rdb · cache · config · httpx"]
        api --> analytics
        api --> platform
        analytics --> platform
    end
    web -- "JSON over /api/*" --> api
    analytics -- "pgx · read-only pool" --> pg[("Pagila<br/>PostgreSQL 17")]
    analytics -. "optional cache · REDIS_URL" .-> redis[("Redis")]
```

**Four ideas hold the design together:**

1. **Hexagonal inside, contexts outside.** Each business area is a bounded context with a pure domain, use cases that talk to ports, and adapters that plug the ports into the world.
2. **Dependencies point inward, and a test enforces it.** Layers and contexts cannot reach around each other without failing CI.
3. **The database is only ever read.** No tables, views, indexes or migrations are added, and the pool refuses writes.
4. **Every screen section stands alone.** Loading, empty and error are handled per section, and filters live in the URL, so any view can be shared.

### Backend: hexagonal contexts

Read this left to right. It is the path of one request, and every arrow means "calls".

```mermaid
flowchart LR
    client(["Client<br/>(the web app)"])

    subgraph driving ["Driving side: starts the call"]
        http["adapters/http<br/>routes · validation<br/>response types"]
    end

    subgraph core ["Core: the hexagon"]
        app["app<br/>use cases"]
        port["Repo<br/>a port: an interface<br/>defined by app"]
    end

    subgraph driven ["Driven side: gets called"]
        cache["adapters/cache<br/>optional · implements Repo<br/>checks Redis first"]
        pg["adapters/postgres<br/>implements Repo<br/>owns all SQL"]
    end

    redis[("Redis")]
    db[("Pagila<br/>PostgreSQL")]

    client -->|"HTTP request"| http
    http -->|"calls a use case"| app
    app -->|"asks for data"| port
    port -->|"implemented by"| pg
    port -.->|"or by, when REDIS_URL is set"| cache
    cache -->|"looks up"| redis
    cache -->|"on a miss, asks the next Repo"| pg
    pg -->|"read-only SQL"| db
```

- **Driving side:** something outside pushes work in. The HTTP adapter turns a request into a use-case call, and the app never knows HTTP exists.
- **Core:** `app` holds the use cases and defines the `Repo` interface it needs. `domain` (the types and pure rules, standard library only) is used by every box here but sits outside this flow.
- **Driven side:** the app reaches out for data through its own interface. The Postgres adapter implements it, so the app never knows about SQL. The optional cache adapter implements the same interface and wraps the Postgres one, so the app does not know a cache exists either. `cmd/api` decides at startup whether to wrap it.

The arrows show who **calls** whom, which is not who imports whom. `app` defines the `Repo` interface and calls it, but `app` never imports `adapters/postgres`. The adapter does not need `app` to work: Go interfaces are satisfied implicitly, so it only needs methods with matching signatures, written in `domain` types. It imports `app` for one line only, `var _ app.Repo = (*Repo)(nil)`, a compile-time check that a drifting signature fails in the adapter's own file instead of at the wiring. The dependency points inward, which is the allowed direction, and `app` knows nothing about SQL. That is what keeps SQL out of the core, and it is checked: the architecture test fails if `app` imports anything but `domain`. `cmd/api` is still the only file that wires both sides together. It builds the pieces once at startup and plugs them together, and it is not in the path of a request.

```
api/
  cmd/api/                       composition root: builds the pool, wires contexts, serves HTTP
  internal/
    analytics/                   one context
      domain/                    types and pure rules. Standard library only.
      app/                       use cases + ports.go (the Repo port). Imports domain only.
      adapters/
        postgres/                driven adapter: implements Repo, owns all SQL
        cache/                   driven adapter: implements Repo in front of another Repo; picks what to cache and the keys
        http/                    driving adapter: routes, validation, response types
    platform/                    shared plumbing that knows no context
      db/  rdb/  cache/          read-only pool, Redis client, read-through cache (Redis, TTL, shared lookups)
      config/  httpx/            env config, JSON/problem/logging/health
    archtest/                    enforces the import rules below
```

**Where does it go?**

| If you are changing… | It lives in |
|---|---|
| A business rule (how a delta or previous window is computed) | `domain/` |
| What the system can do (a new question it can answer) | `app/` |
| A query, or how Pagila tables are read | `adapters/postgres/` |
| What is cached, or how a query's key is built | `adapters/cache/` |
| How caching works (Redis, TTL, shared lookups, failing open) | `platform/cache/` |
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

**With the cache on** (`REDIS_URL` set), every `Totals`, `Revenue` and `TopFilms` call above goes through the cache adapter first. A KPI request makes two `Totals` calls, one per window, and each is cached on its own:

```mermaid
sequenceDiagram
    autonumber
    participant A as app.Service
    participant C as adapters/cache
    participant R as Redis
    participant P as adapters/postgres
    participant DB as Pagila

    A->>C: Totals(window)
    C->>R: GET analytics:v1:totals:from:to:store
    alt hit, within the TTL
        R-->>C: stored JSON
        C-->>A: Totals
    else miss, or Redis unreachable
        R-->>C: nothing
        C->>P: Totals(window)
        P->>DB: one query, read-only
        DB-->>P: figures
        P-->>C: Totals
        C->>R: SET with the TTL
        C-->>A: Totals
    end
```

- Requests for the same key at the same moment share one lookup, so a cold key causes one query.
- A Redis error is logged and treated as a miss, so the request is still answered from Postgres. Errors from Postgres are returned and never cached.
- Each lookup logs one `cache` line with the outcome: `hit`, `miss` or `coalesced`.

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
    KpiCard  KpiGrid  RankedTable  Table  SegmentedControl  SelectField  DateField  Field  FilterPanel
    libs/recharts/                wrappers for the 8 chart pieces we use, with our design defaults
  api/                            typed client, one function per endpoint
  lib/                            dates, format, filters, useBreakpoint
  index.css                       Tailwind import, colour tokens (light and dark) and a small base layer
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

### Styling

Styling is **Tailwind CSS v4**, added through its Vite plugin (no `tailwind.config.js`, no PostCSS file). Components carry their styles as utilities next to their markup; `src/index.css` holds only the shared parts.

- **Colour tokens.** The light and dark colours are CSS variables in `index.css`, and an `@theme inline` block exposes them as utilities: `bg-page`, `bg-surface`, `text-ink`, `text-muted`, `border-line`, `bg-accent`, `text-up`, `text-down`, `bg-skeleton`, `rounded-ui`. Dark mode swaps the variable values with the OS setting, so components never write `dark:` variants and the chart's `var(--series-1)` colours keep working. To add a colour, add the variable in both themes and one line in `@theme`.
- **Base layer.** The page body, the focus ring, and one look for native buttons, inputs and selects, so a component only adds what differs.
- **No custom class names, no `@apply`.** If a class string gets long, split out a component (as with `Field` and `Table`) instead of inventing a class.
- **State comes from variants and attributes.** `aria-pressed:`, `aria-expanded:`, `disabled:` and `data-[open=true]:` style the state the markup already exposes.
- **Dynamic classes need full names.** Tailwind finds classes by scanning the source, so a class built by concatenation (`` `text-${tone}` ``) is never generated. Use a lookup table of complete class names, as `KpiCard` does for the trend colours.
- **Tests never assert utility classes.** Where a test needs a state it reads a data attribute (`data-trend`, `data-collapsible`, `data-stale`, `data-numeric`, `data-wide`, `data-open`) or an ARIA attribute, so restyling cannot break a test.
- **Breakpoints** are Tailwind's `sm` (640 px) and `lg` (1024 px).

### Responsive design

Mobile-first with two breakpoints, `sm` (640 px) and `lg` (1024 px). Anything CSS can size is sized by CSS; only the chart's pixel height and axis width, which the chart library takes as props, read the viewport (`useBreakpoint`).

| | Mobile (< 640 px) | Tablet (640 to 1023 px) | Desktop (1024 px and up) |
|---|---|---|---|
| **Page width** | Full width, 16 px gutters | Full width, 24 px gutters | Up to 1280 px |
| **Filters** | Presets plus a **Filters** button (shows an active count); dates, store and grouping open in place, with **Reset filters** (always shown, disabled when nothing is active) | Presets, then fields two per row | One aligned row, every control the same height |
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
| **Timeouts** | 10 s statement timeout, 5 s header read, 15 s response write, 10 s graceful shutdown; Redis dial, read and write 250 ms with no retries |
| **Errors** | RFC 7807-style problem bodies; internals are logged, never returned |
| **Observability** | Structured `slog` request logs with method, path, query and duration; a `cache` log line per lookup (`hit`, `miss`, `coalesced`) when the cache is on; `/healthz` pings the database, not Redis |
| **Caching** | Optional read-through Redis cache on `Totals`, `Revenue` and `TopFilms`; `CACHE_TTL` (10 min), no invalidation because the data is static; identical lookups share one query; fails open when Redis is down |
| **Accessibility** | Keyboard-reachable controls, deltas shown by arrow and text (not colour alone), chart text alternative and table view |
| **Performance** | Chart library lazy-loaded; first load is about 95 kB gzipped; previous data stays up while new filters load |

### Key decisions

| Decision | Alternatives considered | Why |
|---|---|---|
| Hexagonal layers per context | One flat package; classic handler/service/repository layers | The ports are the seams you cut along when splitting services. Extra layers would add files without adding a boundary. |
| Packages in one `go.mod` | A `go.mod` per context with `go.work` | Separate modules only pay off with two or more real contexts. Revisit when a second context exists. |
| Response types in the HTTP adapter | JSON tags on domain types | The domain can change without breaking clients, and the transport can change (for example to gRPC) without touching the domain. |
| Layer rules enforced by a test | A convention in the docs | Rules that are not checked drift. The test needs no extra tooling. |
| Filters in the URL | Component or global state | Shareable, reload-safe, and the browser back button works. |
| One `AsyncSection` for query states | Per-component spinners | A failing query degrades one section, not the page. |
| Store attribution via `payment → rental → inventory.store_id` | Via `staff.store_id` | Matches the `sales_by_store` view; staff-based attribution does not work in this dataset. |
| Chart library isolated and lazy-loaded | Import directly where needed | Swapping the library touches one folder (`components/libs/recharts`), and the first load stays small. |
| Cache as an adapter on the `Repo` port, mechanics in `platform/cache` | A response cache at a gateway; caching inside the Postgres adapter or the use cases | The use cases and handlers do not change, the cache is switched on in one line of `main.go`, and another context can reuse the platform part. A gateway cache would sit above the hexagon and need its own key handling. |
| The cache fails open | Failing the request when Redis is down | The cache is an optimisation, never a reason the API is down. Redis calls have short timeouts and no retries, so a dead Redis costs milliseconds. |
| Tailwind, with our colour variables mapped in | Hand-written CSS; Tailwind's default palette with `dark:` variants | Styles sit beside the markup and states are explicit, while the existing light and dark tokens keep dark mode automatic and the colours unchanged. |

## Adding a context

To add one (here called `<context>`):

1. Create `internal/<context>/{domain,app,adapters/postgres,adapters/http}`, following `analytics`.
2. Define the repository port in `app/ports.go` and implement it in `adapters/postgres`.
3. Expose `Register(mux, ...)` from `adapters/http`, and call it in `cmd/api/main.go`.
4. Run `go test ./...`. The architecture test covers the new context automatically.
5. To cache its queries, add a thin `adapters/cache` that implements the same port and calls `platform/cache.Fetch` with its own keys, then wrap the repo in `main.go` the way analytics does.

## Testing and CI

| Layer | Coverage |
|---|---|
| `api` domain and app | Window and delta maths, KPI comparison with a fake `Repo` |
| `api` HTTP adapter | Defaults, parameter passthrough, 10 bad-input cases, no leaked internals, health, 405 |
| `api` Postgres adapter (integration) | Totals reconciled against the `sales_by_store` view and raw SQL; revenue series sums to the KPI total at every granularity; zero-fill; top films; pool rejects writes |
| `api` platform cache | Against `miniredis`, so no Redis is needed in CI: repeat served from Redis, TTL set and expiry, separate keys, nil and empty slices preserved, concurrent cold requests load once, Redis down still serves data, errors and unreadable entries not kept, a caller hanging up does not lose the shared result |
| `api` cache adapter | Repeats served from Redis, each KPI window cached on its own, distinct keys per window, store, granularity and limit, domain values survive the JSON round trip, `Stores` passes through |
| `api` architecture | Import rules between layers and contexts, plus a check that the rules catch violations |
| `web` architecture | Import rules between the layers, and that only `components/libs/recharts` imports `recharts`, with cases proving the rules catch violations |
| `web` | Routing (dashboard at `/`, 404 elsewhere), URL and filter parsing, formatting, data shaping, breakpoint mapping, `AsyncSection` states, KPI cards, filter bar (including the collapsible panel, active count and reset), the page against a mocked API |

Correctness is checked against SQL written independently of the code under test.

**CI** ([ci.yml](.github/workflows/ci.yml)) runs on every push to `main` and every pull request:

- `api`: `gofmt`, `go vet`, `go test`, `go build`
- `web`: `npm ci`, lint, typecheck, tests, build

> CI has no database yet, so the Pagila integration tests **skip** there and run only locally. A green CI run therefore does not prove the SQL is correct. Once a database is added to CI, set `REQUIRE_DB=1` so a missing database fails the run.

## Splitting into microservices later

Today everything is one deployable with one context. The target keeps the contexts and swaps the in-process wiring for network calls, so any context added later becomes its own service behind the gateway:

```mermaid
flowchart LR
    subgraph today [Today: one deployable]
        direction TB
        t_api["cmd/api"] --> t_an["analytics"]
        t_an --> t_db[("one Pagila database")]
    end
    subgraph later [Later: one service per context]
        direction TB
        gw["gateway<br/>routes /api/*"] --> s_an["analytics service"]
        gw -.-> s_next["another context's service"]
        s_an --> d_an[("analytics DB")]
        s_next -.-> d_next[("its own DB")]
    end
    today ==> later
```

The seams are already in place:

1. Each context reaches its database only through its own `Repo` port and owns its SQL, and the architecture test stops one context importing another. Any future context does the same, with no cross-context queries.
2. The HTTP adapter depends on a small `Analytics` interface, not a concrete type.
3. To split: give each context its own `cmd/<service>` binary and its own `db.Open` pool, put a gateway in front that routes `/api/*` (the web app only talks to `/api`), then move each context's tables into its own database. Cross-service data goes behind a port: the in-process implementation is swapped for an HTTP or gRPC client, or a replicated snapshot, instead of a join.
4. The cache moves with its context. Keys start with `analytics:v1:`, so services can share one Redis without clashing, and each service decides what to cache.

## Phase 2: Redis cache (built) and gateway (planned)

> **Status:** the Redis cache is built (off unless `REDIS_URL` is set), and the flow above already shows it. The gateway is **not built yet**, so the API still listens on `:8080` itself. The full plan, decisions and acceptance criteria are in [docs/PLAN.md](docs/PLAN.md#10-planned-gateway-and-redis-cache).

Target shape once the gateway exists (the cache part is real today):

```mermaid
flowchart LR
    Browser["Browser"] --> GW
    subgraph edge ["Edge (planned): cmd/gateway, :8080"]
        GW["routing, access log,<br/>per-IP rate limit, timeouts"]
    end
    GW -->|"/api/*, /healthz"| API
    subgraph svc ["Analytics: cmd/api (:8081 behind the gateway)"]
        API["adapters/http"] --> APP["app"]
        APP -->|"Repo port"| CACHE["adapters/cache<br/>(cache-aside)"]
        CACHE -->|"hit"| REDIS[("Redis")]
        CACHE -->|"miss"| PG["adapters/postgres"]
    end
    PG --> DB[("Pagila")]
```

- **Cache.** A new driven adapter implements the same `Repo` port and wraps the Postgres one, so the use cases and handlers do not change. The adapter only chooses what to cache and builds the keys; the mechanics (Redis, TTL, shared lookups, failing open) live in `platform/cache`, so another context can reuse them. It checks Redis first and returns the stored value while it is within the TTL; on a miss it queries Postgres, stores the result and returns it. `Totals`, `Revenue` and `TopFilms` are cached. Concurrent identical misses cause a single query. If Redis is down, requests fall through to Postgres. With `REDIS_URL` unset there is no cache.
- **Gateway.** A small reverse proxy on the standard library. It routes `/api/*` and `/healthz` to analytics, writes one access-log line per request with a request id (also passed upstream), rate-limits per IP with a 429 and `Retry-After`, and enforces timeouts with 502 and 504 responses. It has no auth, load balancing or TLS.
- **Redis** runs in Docker through a `docker-compose.yml` in this repo. The apps themselves do not run in Docker.
- **Honest note on benefit.** The heaviest queries take about 19 ms on this dataset, so the cache is not needed for speed. It is here to demonstrate the pattern. A first quick check (one run on a laptop, top films through curl) took about 9 ms cold and 1 ms warm, against about 7 ms with no cache; proper cold and warm numbers will be added here when the measuring phase is done. The data is static (as-of 2022), so there is a TTL and no invalidation; live data would need short TTLs or explicit invalidation.

## Project status and roadmap

- [x] **Phase 1: Overview.** KPIs, revenue by store, top films, URL filters.
- [x] **Backend refactor.** Hexagonal contexts with enforced layer rules.
- [ ] **Phase 2: Gateway and Redis cache.** The cache is built; the gateway and the measured results are still to do. See above and [docs/PLAN.md](docs/PLAN.md).
- [ ] **Microservices split.** Separate binaries and then separate databases; the gateway comes first, in phase 2.

See [docs/PLAN.md](docs/PLAN.md) for decisions, verification notes and known gaps.
