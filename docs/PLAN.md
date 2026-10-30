# Store Operations Dashboard: Project Plan

A portfolio project for a Principal Frontend Engineer interview (React/Vue role), built on the Pagila DVD-rental database. It is meant to be walked through on screen as a code sample, so the plan favours clear structure and explainable trade-offs over feature count.

**Status:** Phase 1 is built, verified, and published to GitHub with a CI workflow. A testing push came before the gateway (section 11) and is complete: Go unit tests (the backend integration tests were later removed), web unit and integration tests with a coverage gate, Playwright end-to-end tests, and k6. Phase 2, an API gateway plus a Redis cache (section 10), is done: the cache is in the API and the gateway is its own service in `gateway/`, with Docker images and a compose stack for the whole system. The repository was then restructured into a monorepo (section 13): `app/` for the services, `infra/` for Docker, and one CI workflow per piece. Redis then became a service of its own, the cache went over HTTP with an in-memory fallback, and the date ranges and store details became server-driven (section 14, all five phases done). The full microservices split (one binary per context, separate databases) is not started.

---

## 1. Goal and constraints

- Show frontend architecture, design choices and problem solving in a code review setting.
- Only 1-2 core frontend features at a time.
- Backend in Go, as simple HTTP servers. No saga, no events. Started with one deployable and 1-2 packages; a Redis cache and a separate gateway service came in phase 2, and a full microservices split is still later.
- Pagila is **read-only**. The dashboard must never write to it, and everything lives in the `dashboard/` folder.
- Frontend stack: React + TypeScript + Vite + Tailwind CSS v4 + TanStack Query.

### Decisions made along the way

| Decision | Choice | Why |
|---|---|---|
| Project idea | Store Operations Dashboard (analytics) | Shows data-heavy UI, server state, charts, URL state, accessibility |
| Earlier saga plan (`rental-platform/`) | Dropped | Much larger scope than needed. Its deleted files in the working tree are untouched. |
| Data layer in phase 1 | Query Pagila directly, one database | Simplest. The database split happens in the microservices phase. |
| Go stack | `net/http` (1.22 routing), `pgx/v5`, `slog` | Minimal dependencies |
| Store attribution | `payment -> rental -> inventory.store_id` | Matches the `sales_by_store` view. Staff-based attribution does not work in this dataset. |
| As-of date | `ASOF_DATE`, default 2022-07-27 | Payments end that day, so "last 30 days" needs a fixed anchor |
| As-of date source for UI | `GET /api/meta` | One source of truth, no duplicated constant |

### Facts about this Pagila copy that shaped the design
- 500 stores exist, but only stores 1 and 2 hold inventory, so only those appear in the UI.
- Payments run 2022-01-23 to 2022-07-27. Rentals run to 2022-08-23.
- `payment` is partitioned by month.

---

## 2. Features

### Frontend (phase 1): Overview
- **Filter bar:** date presets (7/30/90 days), from/to dates, store, group by day/week/month.
- **Key figures:** revenue, rentals, active customers, average payment, each with the previous period and a % delta.
- **Revenue chart:** per-store lines over time, with a text summary and a "show data as a table" toggle.
- **Top films:** top 10 by revenue with a relative bar.
- **Filters live in the URL**, so reload or a shared link restores the view. Only non-default values are written.
- **Loading, empty and error states** with Retry. An error in one section does not break the others. Previous data stays on screen, dimmed, while new filters load.
- **Accessibility:** keyboard-reachable controls, deltas conveyed by arrow and text (not colour alone) with spoken labels, chart text alternative.
- **Performance:** the chart library is lazy-loaded (first load 300 kB, 95 kB gzipped).
- Light and dark colour tokens.
- **Responsive:** mobile, tablet and desktop layouts (see section 6).
- **Routing:** `/` is the dashboard; any other path shows a 404 page with a link home (`src/pages/`). A production host needs an SPA rewrite to `index.html`.

### Backend (phase 1): `analytics` context
| Endpoint | Returns |
|---|---|
| `GET /healthz` | database ping |
| `GET /api/meta` | `{asOf}` |
| `GET /api/stores` | stores that hold inventory |
| `GET /api/kpis` | four KPIs with previous window and `deltaPct` |
| `GET /api/revenue` | one point per (bucket, store), zero-filled |
| `GET /api/top/films` | top films by revenue, `limit` 1-50 |

Shared params: `from`, `to` (YYYY-MM-DD, inclusive, max 366 days), `store`; `granularity` for revenue. Defaults are the 30 days ending at the as-of date. Bad input returns an RFC 7807-style `400`. Internal errors never leak details.

### Phase 2 (done): gateway and Redis cache
- A gateway in front of the API (routing, access log, per-IP rate limit, timeouts) and a Redis cache for the heavy analytics queries, added as seams without changing the use cases. The gateway is a separate service with its own module and Docker image. Details, decisions and acceptance criteria are in section 10.

### Later (not started): microservices
Turn each context into its own `cmd/<service>` binary with its own pool, then split the database. The gateway is already its own service, built in phase 2 (section 10). See the README for the seam.

### Out of scope
Auth, Kubernetes, load balancing and TLS at the gateway, returns and late-fee handling, customer-facing site.

---

## 3. Architecture

```
api/                                 (now app/api, see section 13) Go module: the analytics API, with a Dockerfile
  cmd/api/                           composition root: config, pool, mux, graceful shutdown
  internal/
    analytics/                       one bounded context, hexagonal inside
      domain/                        types and pure rules (window, delta, KPIs). Standard library only.
      app/                           use cases (Service) + ports.go (Repo, the driven port). Imports domain only.
      adapters/
        postgres/                    implements Repo; owns all the SQL
        cache/                       implements Repo in front of another Repo (optional, REDIS_URL); picks what to cache and the keys
        http/                        (package httpadapter) routes, parsing, validation, response types
    platform/                        shared plumbing, knows no context
      db/                            read-only pool (default_transaction_read_only=on, 10s statement timeout)
      rdb/                           Redis client with short timeouts and no retries
      cache/                         read-through cache: Redis, TTL, shared lookups, fail-open (generic, no context types)
      config/                        environment settings
      httpx/                         JSON and problem responses, request logging, /healthz, /api/meta
    archtest/                        test that enforces the import rules below
gateway/                             Go module 2 of 2: its own go.mod, Dockerfile and tests; standard library only
  cmd/gateway/                       composition root: config, server, graceful shutdown
  internal/gateway/                  proxy, request id, access log, per-IP rate limiter, 429/502/504 problem responses
  internal/config/                   environment settings
  internal/archtest/                 fails if the gateway imports anything but the standard library
web/                                 React 19 + TypeScript (strict) + Vite + Tailwind CSS v4 (Dockerfile + nginx.conf serve it)
  src/routes.tsx                     route table: / is the dashboard, * is the 404 page
  src/pages/                         thin route-level screens (HomePage, NotFoundPage)
  src/features/overview/             container: OverviewPage calls the hooks and maps data to props (kpis.ts, films.ts, OverviewFilters)
  src/hooks/                         one react-query hook per file, useFilters (URL state), queryKeys
  src/components/                    pure UI, one folder per component (tsx + index.ts + test)
  src/components/libs/recharts/      the only code that imports Recharts: 8 wrappers with our theme defaults
  src/api/  src/lib/                 typed client; dates, format, filters, useBreakpoint
  src/index.css                      Tailwind import, colour tokens (light/dark), small base layer; no component rules
  src/archtest.test.ts               enforces the front-end import rules
Makefile, docker-compose.yml, .env.example, README.md, docs/PLAN.md, loadtest/, .github/workflows/ci.yml
```

### Backend layering (hexagonal, one context per module)

Dependencies point inward only. `internal/archtest` fails `go test ./...` (so CI) when one is broken.

| Layer | May import |
|---|---|
| `domain` | standard library |
| `app` | standard library, its own `domain` |
| `adapters/*` | its own `domain` and `app`, `platform`, third-party. Never another adapter. |
| `platform` | no context |
| a context | itself and `platform`, never another context |
| `cmd` | anything (composition root) |

Decisions:
- **Packages in one `go.mod`, not one module per context.** Separate modules and a `go.work` only pay off once there are two or more real contexts. Revisit when a second context exists.
- **Response types live in the HTTP adapter.** The domain has no `json` tags, so the domain can change without breaking the API, and the transport can change (for example to gRPC) without touching the domain.
- **No extra repository/service/controller layers** on top of hexagonal. They would add files without adding a boundary.
- **Shared endpoints** (`/healthz`, `/api/meta`) are not analytics concerns, so `cmd/api` registers them next to each context's `Register(mux, ...)`.
- **The cache is a decorator on the port.** `cmd/api` wraps the Postgres repo in the cache adapter when `REDIS_URL` is set; the use cases and handlers never know. See section 10.

Other key seams: the chart library is isolated; `AsyncSection` is the single place for query states.

---

## 4. What was done (phase 1, parts 1-3)

**Part 1: Backend.** Scaffold, `analytics` Repo and Service (stores, KPIs, revenue, top films), HTTP handlers with validation and problem errors, then `/api/meta`.
- Verified: `go vet` and `go test ./...` pass. Unit tests cover window and delta maths. Handler tests cover defaults, passthrough, 10 bad-input cases, no leaked internals, health and 405. Integration tests against the live container reconcile totals with `sales_by_store` and raw SQL, check that the revenue series sums to the KPI total at every granularity, check zero-fill, check top films, and confirm the pool rejects writes.
- A live `curl` for 2022-06-28 to 2022-07-27 matched independent SQL exactly (revenue 10,707.22, rentals 4,158).

**Part 2: Frontend.** Scaffold and config, typed client, URL filter state and query hooks, then the UI components and styling.
- Verified: typecheck, lint, production build and 34 tests (filters, formatting, data shaping, `AsyncSection` states, KPI cards, filter bar, page against a mocked API). A headless-Chrome screenshot of the running app showed all sections with correct figures, and a deep link restored its filters.
- Fixed along the way: a screen-reader label read "down +2.4%"; lazy loading of the chart was initially ineffective because of a static import of the skeleton from the same file.

**Part 3: Wrap-up.** Makefile (`dev`, `test`, `lint`, `typecheck`, `build`, `check`), `.env.example`, README with the architecture and the split path.
- Verified: `make check` exits clean.

### Acceptance criteria (phase 1)

| Criterion | Status |
|---|---|
| `/api/kpis`, `/revenue`, `/top/films` return correct numbers | Met (integration tests and live check) |
| UI shows KPIs with deltas, revenue chart by store, top films | Met (screenshot) |
| Filters in the URL restore the view | Met (unit and page tests, rendered deep link) |
| Skeleton, empty and error states with retry | Met by tests; not looked at visually |
| Go and frontend tests, typecheck and lint pass | Met (`make check`) |
| README with architecture and microservices path | Met |

### Known gaps
- Dark mode is defined but never rendered or checked.
- jsdom cannot render the chart SVG, so tests cover the data, summary and table, not the lines.

---

## 5. Done: backend refactor to hexagonal contexts

Requirements were confirmed first: packages in one `go.mod`, re-layer `analytics` only, hexagonal without extra layers.

| Phase | What | Verified |
|---|---|---|
| 1 | Split `analytics` into `domain`, `app` (with the `Repo` port) and `adapters/postgres`. Pure rules became exported domain methods. | `go test ./...`; the 7 integration tests ran against live Pagila |
| 2 | `httpapi` became `adapters/http` with its own response types; `db`, `config` and shared HTTP helpers moved to `platform/`; `main` builds the mux. | 16 requests (every endpoint, filters, empty ranges, 400s) returned byte-identical responses before and after |
| 3 | Added `internal/archtest` to enforce the import rules; README and this plan updated. | `make check` clean; a planted violation is caught |

Acceptance criteria: no behavior change (met), layers hold and are enforced in CI (met), domain has no transport tags (met), contexts are isolated (met), tests moved and pass (met), docs match (met).

---

## 6. Done: responsive layout and filter redesign

Requirements confirmed first: collapsible Filters panel on mobile, two-column layout on desktop, verification by screenshots at 375, 768 and 1280 px.

| Phase | What | Verified |
|---|---|---|
| 1 | Mobile-first `index.css`; `main` as a `minmax(0, 1fr)` grid (fixes the page growing wider than the viewport on phones); 2×2 KPIs on mobile; Revenue beside Top films on desktop. | Screenshots at three widths, no horizontal overflow |
| 2 | Filter redesign: segmented 7d/30d/90d presets, aligned fields, collapsible panel on mobile with active count and **Reset filters**, 44 px touch targets. | 4 new tests (toggle, count, reset, presets not counted); keyboard and `aria-expanded` |
| 3 | Chart height and axis width per breakpoint (`useBreakpoint`), axis ticks without cents, Top films shows rentals under the title on mobile, data table dates no longer wrap. | Screenshots in light and dark; `make check` |

Acceptance criteria: no horizontal scroll at 375, 768, 1280 px (met); mobile filters collapse with a count (met); KPIs 2×2 on mobile, 4 across from 640 px (met); desktop is about 1280 px with Revenue beside Top films (met); touch targets at least 44 px, measured in the browser (met); behavior unchanged, filters still in the URL (met, tests); keyboard and screen-reader behavior preserved (met by tests, not tried with a real screen reader).

Found while reviewing the screenshots: the selected preset was white on white, and the "To" date input stuck out of the mobile card. Both fixed.

Known gaps: tested in headless Chrome with touch emulation, not on real devices or other browsers; Safari date inputs and the native select look differ and were not checked.

---

## 7. Done: front-end structure (hooks, pure components, chart wrappers)

Requirements confirmed first: generic UI primitives that take plain props, `useFilters` beside the query hooks, one small test per hook, a thin wrapper per chart piece with our defaults, and a rule that only the wrapper folder imports `recharts`.

| Phase | What | Verified |
|---|---|---|
| 1 | `src/hooks/`: one file per query plus `useFilters` and `queryKeys`, each with a test. `filters.ts` moved to `lib/` so hooks do not import a feature. | 58 tests; screenshots pixel-identical |
| 2 | Every shared component in its own folder (component, `index.ts`, test). Added `ErrorState` and `PageHeader`. `AsyncSection` no longer imports react-query. | 66 tests; screenshots pixel-identical |
| 3 | Generic components (`KpiCard`, `KpiGrid`, `RankedTable`, `SegmentedControl`, `SelectField`, `DateField`, `FilterPanel`). Feature containers map domain data to props. Old `KpiCards`, `TopFilms`, `FilterBar` removed. | 98 tests; layout identical, 2,076 of about 20 million pixels differ (glyph anti-aliasing) |
| chart | `components/libs/recharts/`: 8 wrappers with defaults, 27 tests; `TimeSeriesChart` uses only them. | Breaking each default on purpose was caught by a test (10 of 11; the 11th was a redundant default, removed). Screenshots unchanged by the wrappers. |
| 4 | `src/archtest.test.ts` enforces the layer rules and the single recharts door; README and PLAN updated. | Planted violations are caught; `make check` |

Notes:
- The pixel differences in phase 3 come from how text is split into nodes (one string versus several), which shifts glyph edges by a pixel or so. Layout and sizes are identical.
- The rule test reads the directory with Node at run time. An earlier version used a Vite glob, which Vite caches, so newly added files were not checked until the test file changed. Node's types (`@types/node`) were already a dev dependency, so nothing was added.
- `minTickGap` on the X axis cannot be observed in jsdom (no text measurement), so only the screenshots cover it.

---

## 8. Done: Tailwind CSS

Requirements confirmed first: Tailwind v4 through the Vite plugin, the existing colour variables mapped into `@theme` (so dark mode stays automatic), and tests that read data attributes instead of utility classes.

| Phase | What | Verified |
|---|---|---|
| 1 | Installed `tailwindcss` and `@tailwindcss/vite`; `@theme inline` maps the colour variables to utilities. Old CSS kept alongside. Tailwind's reset removes bold headings, so headings got an explicit bold. | Production screenshots pixel-identical to a fresh baseline |
| 2 | Leaf components moved to utilities (`Skeleton`, `KpiCard`, `KpiGrid`, `PageHeader`, `ErrorState`, `AsyncSection`, `RankedTable`, chart data table, `NotFoundPage`), with a shared `Table`/`Th`/`Td`. State exposed as data attributes. | 163 tests; 10/10 pixel-identical, three runs |
| 3 | Filters (`FilterPanel`, `SegmentedControl`, `SelectField`, `DateField`, new `Field`), page layout and the native-control base layer. `index.css` reduced to tokens and a small base layer. | 166 tests; 10/10 pixel-identical, three runs; 14 extra images (active/custom filters, enabled Reset, empty state, 404, light and dark) identical to a build of the pre-Tailwind commit |
| 4 | README and PLAN describe the setup; final `make check`. | `make check` |

Acceptance criteria: Tailwind v4 via the plugin (met); `index.css` holds only the import, theme mapping, tokens and base layer (met); no custom class names or `@apply` in components (met); dark mode follows the OS with no `dark:` prefixes (met); tests never assert utility classes, apart from one that checks a passed-in `className` comes through (met); no visible change (met, pixel-identical on the production build); CSS size reported (below).

Notes:
- **The CSS did not get smaller.** The production CSS went from 6.16 kB (2.00 kB gzipped) to 15.67 kB (4.35 kB gzipped), because Tailwind's reset and theme variables outweigh the hand-written rules they replaced. The gain is maintainability, not bundle size.
- **Dev server screenshots flake, production ones do not.** In the Vite dev server a chart sometimes captured blank because of hot-reload CSS injection; the production build never did, so it was the reference throughout.
- **Not covered by screenshots:** keyboard focus rings and hover. The focus rule is unchanged in the base layer, and there are no hover styles.

---

## 9. Done: published to GitHub with CI

**Repo:** https://github.com/SangNguyen2810/store-operation. Phase 1 is on `main`; the Pagila checkout and its upstream are untouched.

**What was done**
- Initial commit of the dashboard, then a rename of the Go module (`github.com/SangNguyen2810/store-operation/api`) and the web package (`store-operation-web`), then the CI workflow.
- `.github/workflows/ci.yml` runs on pushes to `main` and on pull requests, with two jobs:
  - **api:** Go from `api/go.mod`, `gofmt`, `go vet`, `go test`, `go build`.
  - **web:** Node 22, `npm ci`, lint, typecheck, tests, build.
- Integration tests fail instead of skipping when `REQUIRE_DB` is set. (Now set in CI, see section 11.)

**Original decision (superseded by section 11, Phase B): CI had no database.** The integration tests skip themselves when Pagila is unreachable, so CI cannot fail because of the database. The cost is that a green run does not prove the SQL is right; the 7 integration tests only run locally (`make check` with the Pagila container up).

**To add the database later**
- The newest upstream Pagila does not load on `postgres:17` or `postgres:18` (it needs `uuidv7()` and pgvector).
- Pin Pagila to `5ba5a57` (2024-12-02) and use `postgres:17`. This was checked locally: schema and data load, and the Go tests pass with `REQUIRE_DB=1`.
- In the api job, add a Postgres service, clone Pagila at that commit, load `pagila-schema.sql` then `pagila-data.sql`, and set `TEST_DATABASE_URL` and `REQUIRE_DB=1`.

**Open points**
- The first CI run on GitHub has not been checked.
- The first commit carries a `Co-Authored-By: Claude` trailer, which makes Claude show as a contributor. Removing it needs an amend and force-push, not done yet.

---

## 10. Planned: gateway and Redis cache

**Progress:** Phases A, B and C are done. The gateway was built as a separate service (own Go module, Docker images, compose stack), which changed the plan below; see "Phase B and C result".

**Goal.** Show system-architecture skills on the existing app: an edge gateway and a cache, each added at a seam so the use cases and handlers do not change.

**Requirements (answered).** The cache lives in Redis and is checked first: on a hit within the TTL the cached value is returned, otherwise the query runs and the result is stored. The gateway does routing, access log, rate limit and timeouts only. Redis runs in Docker.

**Measured baseline.** On this Pagila copy the heaviest analytics queries over the full date range take about 19 ms each (psql `\timing`, one run each: payments joined to rentals, and top films). The data is small, so the cache is not needed for speed here. Its value is as an architecture demonstration, and Phase C reports whatever is measured, including a small gain, rather than assuming one.

### Decisions

| Area | Decision | Why |
|---|---|---|
| Cache location | Cache-aside in Redis. The mechanics live in `platform/cache` (generic: Redis, TTL, shared lookups, fail-open). A thin driven adapter `analytics/adapters/cache` implements `app.Repo`, wraps `postgres.Repo`, and only decides what is cached and how keys are built. Wired in `main.go`: `app.NewService(cache.New(platformcache.New(rc, ttl), postgres.New(pool)))` | The use cases and handlers do not change, and another context can reuse the platform part. Chosen over keeping everything in the adapter, which would have meant copying the mechanics into the next context. A gateway response cache was rejected: it sits above the hexagon and needs its own key normalisation |
| What is cached | `Totals`, `Revenue`, `TopFilms`. Not `Stores` (tiny) | `KPIs` calls `Totals` twice (current and previous window), so each window is cached on its own and reused |
| Flow | Check Redis; on a hit return it; on a miss query Postgres, store with the TTL, return | The behaviour asked for |
| TTL and invalidation | `CACHE_TTL`, default 10 minutes; no invalidation | The dataset is static (as-of 2022). Live data would need short TTLs or explicit invalidation, and the README will say so |
| Key | `analytics:v1:<method>:<from>:<to>:<store or all>:<granularity or limit>` | Deterministic and unique per query. The version prefix is bumped when the stored format changes |
| Stored value | JSON of the domain result | Domain types carry no JSON tags, so field names are the Go names. The `v1` prefix protects against format changes |
| Stampede | `singleflight` per key | One cold key causes one Postgres query, however many requests arrive together |
| Failure | Fail open: a Redis error is logged and the request is served from Postgres | The cache must never be a reason the API is down |
| Switch | Cache is off when `REDIS_URL` is unset (the default) | Tests, CI and `make dev` without Redis behave as today |
| Gateway | **Changed during the build:** a separate Go module in `gateway/` (`cmd/gateway`, `internal/gateway`), built on the standard library `httputil.ReverseProxy`, with its own Dockerfile | No framework. A separate module means it shares no code with the API and imports only the standard library (a small problem writer is duplicated on purpose). It was first planned as a package inside the API module |
| Gateway features | Routing (`/api/*` and `/healthz` to analytics); access log with method, path, status, duration, client IP and request id (the id is passed upstream in `X-Request-Id`); per-IP token-bucket rate limit returning a 429 problem response with `Retry-After`; server timeouts and upstream dial and response-header timeouts returning 502 or 504 problem responses | The four features requested |
| Not in the gateway | Auth, load balancing, TLS, caching, retries | Out of scope; one upstream only |
| Rate-limit key | The connection's remote IP; `X-Forwarded-For` is not trusted | Nothing sits in front of the gateway. Idle limiter entries are evicted so memory stays bounded |
| Ports | Gateway on `:8080`, so the web dev proxy is unchanged; analytics moves to `:8081` via `ADDR` | The browser only ever talks to the gateway |
| Docker | **Changed during the build:** `docker-compose.yml` runs Redis, the API, the gateway and the web app (nginx, which proxies `/api` to the gateway). Only the gateway (8080) and web (3000) ports are published, plus Redis. Pagila keeps its own compose in its own repo and the API reaches it through `host.docker.internal` | The gateway was asked to be its own service started by Docker, so images for the apps moved into scope |

### Phases

Each phase ships and is verified on its own; stop for a go-ahead after each.

**Phase A: Redis cache (done)**
- `docker-compose.yml` for Redis, `REDIS_URL` and `CACHE_TTL` in config and `.env.example`, the `go-redis` dependency.
- `analytics/adapters/cache` with the key builder, JSON codec, `singleflight` and fail-open behaviour; wire it in `main.go`.
- Tests with `miniredis` (pure Go, so they run in CI with no Redis): hit, miss, TTL expiry, two windows of one KPI call cached separately, key uniqueness per filter, concurrent cold requests cause one inner call, Redis down still returns data. An integration test against real Redis skips when it is unreachable.
- Verify: `go test ./...`, `go vet`, `gofmt`, the architecture test (adapters may use third-party packages), then by hand: a repeat request is a logged hit, stopping Redis still returns 200, and responses are byte-identical with and without the cache.

**Phase A result**
- Built: `platform/cache` (generic `Fetch`: Redis, JSON codec, TTL, `singleflight` via `DoChan`, fail-open), `analytics/adapters/cache` (thin: which methods are cached and the keys), `platform/rdb` (short timeouts, no retries; an unreachable Redis is only logged at start), `REDIS_URL` and `CACHE_TTL` config, wiring in `main.go`, `docker-compose.yml` (host port set with `REDIS_PORT`, default 6379), `make redis-up` and `make redis-down`, and architecture-test cases for the new adapter.
- Tests: 9 in `platform/cache` and 5 in the adapter, against `miniredis`, all passing with `-race`. Three deliberate breakages (no TTL, store left out of the key, no `singleflight`) each made a test fail, in the package that owns that behaviour, then were reverted. A fourth (wrong method label in a key) survived because the films key has an extra segment, so it cannot collide: an equivalent change, not a gap. The full Go suite, `gofmt`, `go vet` and `oxlint` pass; the Postgres integration tests ran against the live Pagila container.
- End to end, with Postgres and Redis in Docker: 11 requests (success, empty windows, unknown store, a 400) returned status and body byte-identical to an API without the cache, on both the cold and the warm call. Entries carry the 10 minute TTL. With Redis stopped, requests still return 200 with the same bodies and a warning is logged.
- Quick timing, one run on a laptop (curl total time): top films cold 9 ms, warm about 1 ms, versus about 7 ms without the cache. Under 20 concurrent users the gain is larger; see the Phase F result in section 11.
- Deviations: the generic cache logic was first written inside the analytics adapter, then moved to `platform/cache` at your request (all tests and the end-to-end check were repeated after the move). `go get` first raised the module to Go 1.26; the dependencies were pinned instead (`go-redis` v9.14.0, `miniredis` v2.35.0, `x/sync` v0.17.0) so `go.mod` stays at Go 1.25.0 and CI is unaffected. Port 6379 was taken by a local `redis-server` on the machine, so the Docker host port became configurable.
- Not done in this phase: the log line shows hit, miss or coalesced per query, but there are no counters or metrics endpoint.

**Phase B: Gateway (done, built as a separate service; see "Phase B and C result")**
- `gateway/` module (proxy, request id, access log, rate limiter, error responses) and `cmd/gateway`; Dockerfiles and compose for all services; Makefile runs the API, gateway and web; config for ports and limits.
- Tests with `httptest` upstreams: routing, request id passed upstream, access log fields, 429 with `Retry-After`, 504 on a slow upstream, 502 on a dead upstream, and limiter eviction.
- Verify: `go test ./...`, then through the gateway every endpoint returns the same JSON as calling analytics directly, and the web app works unchanged.

**Phase C: Measure and document (done)**
- Point the existing k6 scripts (section 11, Phase F) at the gateway with `BASE_URL`, compare with the direct numbers, and record them in the README.
- README: updated architecture diagram, configuration table, key decisions, and the limits (static data, one upstream). Update this plan and `.env.example`.
- CI already has Redis and Pagila services (section 11, Phase B); the gateway needs nothing more for its own tests.

**Phase B and C result**
- Changed from the plan: the gateway is its own Go module in `gateway/` (standard library only, no code shared with `api/`; a small problem writer is duplicated on purpose), and Docker images for the apps are now in scope. `docker-compose.yml` runs Redis, the API (:8081), the gateway (:8080) and the web app (nginx, :3000). The API default port moved to :8081.
- Built: proxy with routing, `X-Request-Id`, access log, per-IP token bucket with eviction, 429/502/504 problem responses; the API logs the request id; Makefile, e2e, k6 and CI (`gateway` and `docker` jobs) updated.
- Verified: 13 paths byte-identical through the gateway and directly (status, body, content type), also through the nginx container with the cache on; 502 with the API stopped; 429 with `Retry-After`; e2e 11/11 and k6 smoke 30/30 through the gateway; six deliberate breakages in the gateway, five caught, one equivalent.
- Measured (`make k6-compare`, one laptop run, 20 users): no cache avg 8.5 ms / p95 22.0; cache 1.8 / 3.6; cache through the gateway 2.0 / 4.0. The gateway costs about 0.2 ms per request.
- Found: at the default limit (50 requests/s per IP) the k6 load test is rate limited (1,342 of about 6,900 checks failed with 429s, latency unaffected). The limit is working as designed; load tests through the gateway run it with `RATE_LIMIT_RPS=0`. In the Docker stack all browsers share the web container's IP, so they share one bucket.
- Not done: the CI `gateway`, `docker` and updated `k6` jobs have not run on GitHub; the nginx container was checked with curl, not a browser.

### Acceptance criteria (all met; see the results above)

1. Through the gateway the web app works unchanged, and every response is byte-identical to today's.
2. With `REDIS_URL` set, a repeated identical request is served from Redis and Postgres is queried once, shown by a test and a log line.
3. A cached value expires after `CACHE_TTL` and is then queried again.
4. Concurrent identical cold requests cause one Postgres query.
5. With Redis stopped, the API still returns correct data and logs the failure.
6. With `REDIS_URL` unset, behaviour is exactly as today.
7. The gateway writes one access-log line per request with method, path, status, duration, client IP and request id, and passes the request id upstream.
8. Exceeding the rate limit returns a 429 problem response with `Retry-After`; a slow upstream returns a 504 and a dead upstream a 502.
9. The architecture test still passes, the gateway imports only the standard library (its own architecture test), and CI stays green (not yet run on GitHub).
10. The README reports the measured cold and warm times honestly, and says the data is static.

### Risks

- Caching a 19 ms read-only workload adds a moving part. This is a demonstration, and the README says so.
- The cached JSON is coupled to the domain type shapes; the key version prefix is the guard.
- `make dev` now runs three processes (API, gateway, web), and `make up` four containers plus Redis, so failures are harder to read; the access log and request id are there to help.
- New dependencies: `go-redis/v9` (runtime) and `miniredis` (tests only).

---

## 11. In progress: tests before the gateway

**Request:** unit and integration tests for the Go API and the web app, Playwright end-to-end tests, and k6 API tests, all before the gateway. Phases A to F, one at a time.

**Acceptance criteria:** Go tests pass against a real Postgres and Redis and fail in CI if either is missing; coverage floors are enforced (Go 90% domain/app, 80% elsewhere; web 85% lines); web integration tests cover filter, URL, error and retry, reset and 404; Playwright passes six scenarios on the real stack; `make k6-smoke` and `make k6-load` run with p95 under 200 ms and errors under 1%; docs updated; mutation checks reported honestly.

| Phase | Scope | Status |
|---|---|---|
| A | Go unit tests for the gaps | Done |
| B | Go integration tests and CI services | Done (see below) |
| C | Web unit tests and coverage | Done (see below) |
| D | Web integration tests (MSW) | Done (see below) |
| E | Playwright end-to-end | Done (see below) |
| F | k6 | Done (see below) |

**Phase A result.** New tests for `domain` (20% to 100%), `app` (54.5% to 100%), the HTTP adapter (response shapes, `stores`, 500s on every endpoint), `httpx`, `config` and `rdb`. Eleven deliberate bugs were tried: ten were caught, one real survivor (a delta that ignored negative previous values) got a test and is now caught, and one control that replaced a line with itself survived as expected.

**Phase B result.**
- `internal/platform/testutil` opens Pagila and Redis for tests and skips when unreachable, or fails when `REQUIRE_DB` is set.
- `cmd/api` now builds its handler in `newHandler`, and `cmd/api/server_integration_test.go` drives the real router, use cases and SQL over HTTP, with and without the cache, against real Pagila and Redis (10 tests). `platform/db` and `platform/cache` have their own integration tests (3 and 3).
- `api/scripts/covergate.sh` enforces the coverage floors; `make test-api-integration` and `make cover-api` run it locally.
- CI: Postgres 17 loaded with Pagila at `5ba5a57`, a Redis service, `REQUIRE_DB=1`, the gate and a `-race` run.
- Verified: the whole suite passes with `-race` against a fresh `postgres:17` loaded with the pinned Pagila files (exactly the CI recipe) and the Docker Redis; unreachable services skip locally and fail with `REQUIRE_DB=1`; the gate fails when a package drops below its floor or the database is missing. Eight mutations were tried (meta route removed, wrong as-of date, cache key ignoring `to` or `limit`, value never stored, no TTL, pool not read-only, changed statement timeout): all caught, the key ones by the adapter unit tests rather than the integration tests.
- Not verified: the workflow on GitHub itself (it cannot be run locally; the YAML parses and every step was run by hand).
- Coverage now: domain, app, cache adapter and rdb at 100%; HTTP adapter 95.7%; httpx 95.0%; platform cache 94.7%; db 92.9%; Postgres adapter 86.2%.

**Phase C result.**
- Added `@vitest/coverage-v8`, `npm run test:coverage` and thresholds (85% lines and statements, 80% functions and branches) over all of `src` (tests, `main.tsx`, barrels and the types file excluded). CI runs it instead of plain `npm test`; `make cover-web` runs it locally.
- Finding: the API client and date helpers were already 100% covered through the hooks, so "untested" in my earlier assessment was too strong. They now also have direct tests, because a hook test would not notice a wrong query string or a swapped error field: `api/client`, `api/analytics`, `lib/dates`, `hooks/queryKeys`, plus `useBreakpoint`, `App`, `HomePage`, `RevenueChart`, an `AsyncSection` error with no message, the `To` date field, the page with the store list failing, and `formatCurrencyWhole`.
- Result: 40 to 47 test files, 166 to 230 tests; coverage 98.6% statements, 100% branches, 97.3% functions, 99.6% lines. The only uncovered code is the chart tooltip formatter callbacks (jsdom cannot render a recharts tooltip), which the Playwright phase will exercise.
- Verified: `tsc -b`, `oxlint` and all tests pass; the gate fails when a threshold is raised above the real figure; thirteen deliberate bugs (query-string handling, error fields, URLs, cache keys, date maths, breakpoint logic, chart label, store list) were all caught, and a no-op control survived as expected.

**Phase D result.**
- Added `msw` (dev). `src/test/msw/handlers.ts` is a small fake of the Go API that answers by query string (figures and films depend on the store, the series follows the granularity) using the real paths and JSON field names. `server.ts` starts it per test file with `onUnhandledRequest: 'error'`, records every request, and resolves relative URLs against the page origin (Node's `fetch` cannot take them). `renderApp` mounts the real route table with a fresh query cache and a URL probe.
- `src/integration/dashboard.integration.test.tsx`: 25 tests in groups: first load, changing a filter (store, All stores, preset, granularity, custom date, reversed range), the URL as state (shared link, five kinds of invalid parameters), browser Back, reset, errors and retry (section, KPI, page-level, failure after a filter change), empty and loading states (previous data kept and busy, last choice wins in a race), routing.
- Result: 48 test files, 255 tests; coverage 99.0% statements, 100% branches, 97.9% functions, 99.6% lines. Three consecutive runs gave identical results (about 2 s for the group).
- Verified: `tsc -b`, `oxlint` and all tests pass. Sixteen deliberate bugs in the app (URL writing and parsing, query keys, kept data, busy state, both Retry buttons, Reset, the 404 route, request parameters, error detail, history push versus replace) were tried: fifteen were caught at once. The sixteenth, replacing instead of pushing history entries, survived because nothing tested the Back button; a Back test now catches it.
- Deviations: `userEvent.type` does not work on a date input in jsdom, so date tests use `fireEvent.change` like the existing filter tests. The first run captured `fetch` before MSW patched it and so hit the real network; fixed by taking it after `listen()`. One lint rule flagged `useFakeApi` as a hook, so it is `setupFakeApi`.
- Not covered here, on purpose: real layout and the chart's drawing and tooltip (jsdom cannot), which is Phase E.

**Phase E result.**
- Added `@playwright/test` (dev; Chromium only). `web/playwright.config.ts` starts the real stack itself on ports of its own so `make dev` keeps running: `go run ./cmd/api` on `:8180` against the local Pagila, then `npm run build` and `vite preview` on `:4173`. `vite.config.ts` gained `preview.proxy` and an `API_PROXY` variable so the production build reaches that API; Vitest now only collects `src/**/*.test.*` so it never picks up the specs.
- `web/e2e/dashboard.e2e.ts`: 11 tests. The figures are compared with what the API returns for the same window, with a guard that they are non-zero so the comparison cannot pass on an empty database. They cover the overview, store filter plus reload, presets plus real browser Back, grouping by month, Reset, 404 and the way back, an injected 503 with Retry, the chart tooltip, and three phone-width tests (folded filters and no horizontal scroll, the active-filter count, the films table losing its rentals column).
- CI: a third job `e2e` (Postgres service, shared `load-pagila` action, Chromium, `npm run e2e`, report uploaded on failure). `make e2e-install` and `make e2e` run it locally.
- Result: 11 passed in about 6 s, three runs in a row with identical results.
- Verified by deliberately breaking the product: eleven bugs in the web app and the Go API (store not written to the URL, cents dropped from money, filters not folded on phones, tooltip removed, rentals column kept on phones, no 404 route, Retry doing nothing, Back not undoing filters, Reset doing nothing, the API ignoring the store filter) were tried. Ten were caught. One survived by design: the API's previous window one day short, because these specs take their expected figures from the same API, so a consistent error on both sides is invisible to them; the Go domain test catches it (`TestPreviousWindow` fails).
- Deviations: a locator for the chart matched the legend icons too (they are images), so the chart is found by its "Revenue per…" description; and the first error test passed nothing to wait for because the app retries a failed query once, so the injected failure had to happen twice. Port 8080 was already taken by the developer's own API, which is why the stack uses `:8180`.
- Not verified: the `e2e` job on GitHub itself (it cannot be run locally). Its steps are the ones run here, plus `playwright install --with-deps`.

**Phase F result.**
- Installed k6 v2.3.0 (`brew install k6`). `loadtest/`: `lib.js` (base URL, shared requests and shape checks), `smoke.js`, `load.js`, `compare.sh` and a short README. `make k6-smoke`, `make k6-load` and `make k6-compare` run them; `BASE_URL` points them at any API, so they can later go through the gateway; `CACHE=on|off` only tags the results.
- Smoke: 30 checks over every endpoint, all pass, p95 about 12 ms. Load against the running API: 3,542 requests in 45 s, 0% failed, 100% of 6,831 checks passed, p95 19.5 ms overall (thresholds: 200 ms overall, 250 ms per endpoint, under 1% errors).
- Cold versus warm (`make k6-compare`, three runs): average 7.8 to 7.9 ms without the cache and 1.7 ms with it; p95 about 20 ms against 3.5 to 3.7 ms. Recorded in the README, with the caveat that the request mix reuses a small set of filters, so it favours the cache.
- CI: a fourth job `k6` runs the smoke script against a real API and Pagila. The load test is not run in CI because runner timings are too noisy for a latency threshold.
- Verified by running broken APIs on their own port: a renamed JSON field, a bad granularity accepted, a 300 ms slowdown in KPIs, and one store's films failing were all caught, and the unmodified API passed as the control. One survived at first: films ranked backwards. The lowest-revenue films all tie at $0.99, so a "not increasing" check accepted a list of equal values. A second check on a wide window, where the best film clearly earns more than the fifth, now catches it.
- Findings and fixes: the first smoke run failed because I expected a problem body for an unknown path, but the router's own 404 and 405 are plain text (documented in the README), so that check is now status only. That led me to correct an overstatement of mine from Phase B: the README said the HTTP integration test checks "400 and 404 problem bodies", but it checks the 404 status only. `compare.sh` first ran `go build` from the wrong directory. A first mutation helper had its arguments shifted and reported meaningless results; it was discarded and redone, and a pattern-based `pkill` that could have hit the developer's own API was refused, so only processes I started are signalled, by PID.
- Not verified: the `k6` job on GitHub itself.

**Backend integration tests removed.** At the developer's request all Go integration tests were removed: the Postgres adapter, the HTTP stack, `platform/db`, the real-Redis cache tests, and the `testutil` helper, together with `make test-api-integration`, the `REQUIRE_DB` and `TEST_*` settings, and the Postgres and Redis services of the `api` CI job. The web integration tests (Phase D) were first removed by mistake and then restored; they stay. What the removal gives up: nothing now checks the SQL against Pagila's own `sales_by_store` view, that the pool rejects writes, or that the cache returns byte-identical responses through the real HTTP stack; the Go unit tests (including the cache against an in-memory Redis), the web integration tests, the end-to-end tests and the k6 smoke test still cover the visible behaviour. The Go coverage gate no longer counts `adapters/postgres` and `platform/db`, which now have no tests. Phase B above describes the removed work.

---

## 12. Next steps, in order

1. Push and check the first CI run on GitHub (five workflows: api, gateway, redis, web, stack, with e2e, k6 and docker jobs in stack) and fix what fails.
2. (done) The testing work in section 11.
3. (done) Phase 2: gateway and Redis cache (section 10).
4. Later: splitting the packages into binaries and databases.

---

## 13. Done: restructure into a monorepo

*This section records the restructure as it was done. `shared/` described below was removed in section 14, when Redis became a service and the cache client moved into the API.*

**Request.** Put `api`, `gateway` and `web` in an `app` folder, take the Redis code out of the API into a shared folder, and shape the repo like a big-tech microservice monorepo.

**Layout (the paths in sections 3 to 12 predate it).**

```
app/api  app/gateway  app/web   the deployable services
shared/                         Go module: cache (read-through Redis cache) and rdb (Redis client)
infra/                          docker-compose.yml (Redis, API, gateway, web)
loadtest/  docs/  Makefile  .env.example  .github/
```

**Decisions**

| Area | Decision | Why |
|---|---|---|
| What is shared | Only `cache` and `rdb`, moved from `api/internal/platform`. `httpx`, `config` and `db` stay in the API | They are the only generic parts with a likely second user. Moving `httpx` now would leave two styles beside the gateway's own problem writer |
| What stays in the API | `analytics/adapters/cache`: which queries are cached and how keys are built | A business decision, not plumbing |
| Redis the server | `infra/docker-compose.yml`, a deployed dependency | A cache service of our own would add a network hop to a 1 ms lookup. The client library is shared; the server is infrastructure |
| Gateway | Still imports nothing from `shared/` or the API; its test now rejects both | Services must stay independently replaceable |
| Module paths | Renamed to match the folders: `.../app/api`, `.../app/gateway`, `.../shared`. The API reaches `shared` through a `replace` directive, so it builds the same locally, in Docker and in CI without a `go.work` | One path scheme, no workspace file to keep in sync |
| Dependencies | Versions stay pinned (go-redis 9.14.0, miniredis 2.35.0, x/sync 0.17.0) and `go.mod` stays at Go 1.25.0 | `go mod tidy` otherwise raises Go to 1.26 |
| API image | Built from the repository root (`app/api/Dockerfile`, with a `Dockerfile.dockerignore`) so `shared/` is in the build context | The only service with a dependency outside its folder |
| CI | `ci.yml` split into `api.yml`, `gateway.yml`, `shared.yml`, `web.yml` and `stack.yml`, each with path filters. `api.yml` also watches `shared/**` | A change to one service builds only that service, as in a large monorepo. Jobs that need several services (docker, e2e, k6) run whenever any of them changes |

**Result**
- Moves were done with `git mv`, so file history follows the files.
- Verified: `make check` (lint, typecheck, all tests, build); coverage gates for api (6 packages), gateway (2), shared (2) and web (98.96% statements, 100% branches); e2e 11/11; the Docker stack built from `infra/` with all four containers healthy; 16 cold and warm responses byte-identical to the standalone API; `make k6-compare` (cache with gateway 2.2 ms average against 1.9 ms direct).
- Architecture tests: the API's no longer list `platform/cache` and `platform/rdb`; the cache adapter may import `shared/cache`; `shared` has a test that fails if it imports a service; the gateway's test rejects both the API and `shared`.
- Not done: the five new workflows have not run on GitHub, so the path filters and the `shared` replace directive in CI are untested there. Files under `docs/` other than this section still use the old paths.

---

## 14. Done: the redis service and data-driven ranges

**Request.** Make Redis its own service (`app/redis`, HTTP between it and the API), keep the in-memory fallback when it fails, store the date ranges the UI offers in Redis (served by an API, validated on every request, with a built-in list when Redis is down), and add store metadata. README updated in every phase.

**Decisions**

| Area | Decision | Why |
|---|---|---|
| Redis service API | HTTP: `GET` and `PUT /v1/cache/{key}?ttl=`, `GET /v1/stats`, `GET /healthz`. Keys `[A-Za-z0-9:._-]{1,200}`, values up to 1 MiB, ttl up to 24 h and required except for keys starting `config:` | Plain HTTP like the rest of the system; `config:` keys carry settings that must not expire |
| gRPC | Not used. gateway to API and API to redis service stay HTTP | The gateway forwards requests without knowing the API; gRPC would make it translate every endpoint. Nothing else calls between services |
| API failure flow | Redis service answers: use it. It fails: use the API's in-memory copy if still alive (500 entries, expiry from `X-Cache-Ttl`). Otherwise Postgres | Fail open, and survive a short outage without hitting the database |
| Date ranges | Served by `GET /api/presets` (read-only for now), from Redis when it has a list, else a list built into the API. The API keeps a copy for 60 s. Requests whose `from` and `to` are not a listed range get `400`; no dates means the default range | Option A: custom dates are no longer accepted, so the set of windows is small and every one can be cached |
| Range list | Recent (last 7, 14, 30 default, 60, 90, 180 days), Periods (month to date, last month, quarter to date, last quarter), Fixed (Feb to Jun 2022, Q1, Q2, all data) | Data spans 2022-01-23 to 2022-07-27 |
| Store metadata | Manager, phone, address, district, postal code, city, country (Pagila); inventory items and active customers (computed). No email, no invented fields. Cached in Redis for 1 h | Real data only; email is personal data |
| UI | Quick buttons for 7, 30 and 90 days, a grouped "More ranges" dropdown, and a store detail line under the filters | Small, visible change |

**Phases (stop after each; README, docs and `.env.example` updated in each)**
1. `app/redis` service. **Done.**
2. API cache client over HTTP, shared lookups, in-memory fallback; remove `shared/`. **Done.**
3. Ranges list, validation, `GET /api/presets`, store metadata. **Done.**
4. Web app: ranges from the API, "More ranges", store detail line, tests. **Done.**
5. Wiring: compose, Makefile, CI, k6 comparison with hit ratios, final docs. **Done.**

**Phase 1 result**
- Built: `app/redis` (module of its own, standard library plus go-redis), with `server`, `store` and `config` packages, a Dockerfile (22 MB image), a CI workflow `redis.yml`, `make dev-redis`, `make test-redis` and `make cover-redis`, and a `redis-service` entry in `infra/docker-compose.yml` with no published port. Nothing calls it yet; the API still uses `shared/` directly.
- Verified: gofmt, vet and race tests pass three times in a row; coverage 100% config, 95.9% server, 94.3% store (floor 80%); eight deliberate breakages of the limits, header, leak protection, retries and expiry handling were each caught (two survivors at first, a testable `remaining` function and a retry assertion fixed them). Against a real Redis container with `curl`: put, get with `X-Cache-Ttl: 600`, miss, limits, `config:` key stored with no expiry, stats; with Redis stopped, `503` and a failing `/healthz`, and a clean recovery with the stored value intact. In Docker the service is healthy and unreachable from the host.
- Note: go-redis turns `MaxRetries: -1` into 0 inside the client, so the test asserts 0.
- Not done: the new CI workflow has not run on GitHub.

**Phase 2 result**
- Built: `app/api/internal/platform/cache` replaces `shared/` (deleted, with its workflow and Makefile targets). It has an HTTP client for the redis service, the read-through `Fetch` with shared lookups, and the in-memory copy (500 entries; the entry closest to expiry goes first; expiry from `X-Cache-Ttl`, 10 min when the key has none). `Get` reports `Hit`, `Miss`, `Fallback` or `Unavailable`. The API's `REDIS_URL` became `CACHE_URL`, and its Docker build context is now `app/api` alone.
- Decision added: after a failure the service is not called again for 5 s. Without that, every request during an outage would wait for the 500 ms client timeout. A key the healthy service reports missing is dropped from the memory copy, so the copy never outlives the service's answer.
- Verified: race tests; coverage 95.2% for the cache package; the old behaviours (shared lookups, error not cached, caller hanging up, unreadable entry) were ported as tests and new ones cover outages, recovery, the retry delay, bad `X-Cache-Ttl` values and the memory limit.

**Phase 3 result**
- Built: `domain/ranges.go` (definitions, validation, resolution against the as-of date, `FindRange`), the `RangeSource` port, `adapters/ranges` (reads `config:presets:v1`, keeps a copy, refreshes every 60 s, built-in list on a miss or an unusable list, keeps the list in use when the service is unavailable), `GET /api/presets`, strict validation in the HTTP adapter (both dates or neither, and the pair must be an offered range), and store details in `Stores` (manager, phone, address, district, postal code, city, country, inventory items, active customers), cached for 1 h. `make seed-presets` and `infra/presets.json` put a list in Redis.
- Decisions added: the stored list is JSON with a `kind` (`days`, `month_to_date`, `last_month`, `quarter_to_date`, `last_quarter`, `fixed`) rather than finished dates, so relative ranges keep moving with the as-of date. When two entries have the same dates the first wins in the UI.
- Verified: domain 100%, ranges adapter 100%; against Pagila, a real Redis and the redis service: 18 ranges served, store details match the database, unlisted and half windows get `400`, a custom list seeded in Redis took effect within the minute (and the old default was then refused), an outage was answered from memory with the new window still working from Postgres, and the service returned by itself.

**Phase 4 result**
- Built: `useRanges`, `fetchRanges`, filters keyed on the API's list (default from the list, URL carries the dates, a link to a range that left the list shows the default), 7d, 30d and 90d buttons from the list, a grouped "More ranges" menu (`SelectField` now supports groups), a store line under the filters (a summary of all stores, or the selected store's details), and a page-level error with Retry when the list cannot be loaded. `DateField` and `lib/dates` were removed.
- Verified: 255 web unit and integration tests, coverage 98.8% lines; 14 end-to-end tests in a real browser against the real API (three new ones for the menu, refusal of an unlisted window, and the store line).

**Phase 5 result**
- Built: compose with the API on `CACHE_URL=http://redis-service:8082`, the CI path filters without `shared/`, k6 scripts that use only offered ranges and check `/api/presets`, store details and the refusals, and a four-column `make k6-compare` (no cache, cache, cache plus gateway, redis service down) that reports how many lookups the cache served.
- Measured (one laptop, 20 users, 45 s each): no cache 6.3 ms average and 15.6 ms p95; with the cache 2.3 and 5.9 with 97.6% of lookups served from it; through the gateway 2.4 and 6.0 with 99.7%; redis service down 0.9 and 1.4, served from the in-memory copy. The HTTP hop to the redis service costs about 0.4 ms against the previous direct Redis call, and the memory copy is faster than any network hop. Zero failed requests in all four.
- Verified in Docker: the whole stack up and healthy, `/api/presets` through the web container, the cache filling Redis, the redis service stopped and the API still answering (from memory, and from Postgres for a new window), then started again; the k6 smoke test passed through the gateway (37 of 37 checks).
- Mutation check on the new Go code: 21 deliberate breakages (cache fallback, retry delay, memory limit, header parsing, quarter and month maths, validation, list handling); three survived at first and each exposed a missing assertion (a miss dropping the memory copy, the reason for half a window, the cache copy used at refresh), which were added; all 21 are now caught.
- Not done: no workflow, including the new ones, has run on GitHub; the nginx container is only checked by hand; in the Docker stack all browsers share one rate-limit bucket (the web container's address).
