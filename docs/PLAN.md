# Store Operations Dashboard: Project Plan

A portfolio project for a Principal Frontend Engineer interview (React/Vue role), built on the Pagila DVD-rental database. It is meant to be walked through on screen as a code sample, so the plan favours clear structure and explainable trade-offs over feature count.

**Status:** Phase 1 is built, verified, and published to GitHub with a CI workflow. CI has no database yet, so the integration tests skip there. Phase 2 is now an API gateway plus a Redis cache (section 10): the cache is built, the gateway is not started. The full microservices split is not started.

---

## 1. Goal and constraints

- Show frontend architecture, design choices and problem solving in a code review setting.
- Only 1-2 core frontend features at a time.
- Backend in Go, as simple HTTP servers. No saga, no Redis, no events. Start with one deployable and 1-2 packages, split into microservices later.
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

### Phase 2 (in progress: cache built, gateway not started): gateway and Redis cache
- A gateway in front of the API (routing, access log, per-IP rate limit, timeouts) and a Redis cache for the heavy analytics queries, added as seams without changing the use cases. Details, decisions and acceptance criteria are in section 10.

### Later (not started): microservices
Turn each context into its own `cmd/<service>` binary with its own pool, then split the database. The gateway comes first, in phase 2 (section 10). See the README for the seam.

### Out of scope
Auth, Docker images for the app (Redis alone runs in Docker, section 10), Kubernetes, load balancing and TLS at the gateway, returns and late-fee handling, customer-facing site.

---

## 3. Architecture

```
api/                                 Go, one go.mod
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
web/                                 React 19 + TypeScript (strict) + Vite + Tailwind CSS v4
  src/routes.tsx                     route table: / is the dashboard, * is the 404 page
  src/pages/                         thin route-level screens (HomePage, NotFoundPage)
  src/features/overview/             container: OverviewPage calls the hooks and maps data to props (kpis.ts, films.ts, OverviewFilters)
  src/hooks/                         one react-query hook per file, useFilters (URL state), queryKeys
  src/components/                    pure UI, one folder per component (tsx + index.ts + test)
  src/components/libs/recharts/      the only code that imports Recharts: 8 wrappers with our theme defaults
  src/api/  src/lib/                 typed client; dates, format, filters, useBreakpoint
  src/index.css                      Tailwind import, colour tokens (light/dark), small base layer; no component rules
  src/archtest.test.ts               enforces the front-end import rules
Makefile, .env.example, README.md, docs/PLAN.md, .github/workflows/ci.yml
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
- Integration tests fail instead of skipping when `REQUIRE_DB` is set. It is off in CI for now.

**Decision: CI has no database yet.** The integration tests skip themselves when Pagila is unreachable, so CI cannot fail because of the database. The cost is that a green run does not prove the SQL is right; the 7 integration tests only run locally (`make check` with the Pagila container up).

**To add the database later**
- The newest upstream Pagila does not load on `postgres:17` or `postgres:18` (it needs `uuidv7()` and pgvector).
- Pin Pagila to `5ba5a57` (2024-12-02) and use `postgres:17`. This was checked locally: schema and data load, and the Go tests pass with `REQUIRE_DB=1`.
- In the api job, add a Postgres service, clone Pagila at that commit, load `pagila-schema.sql` then `pagila-data.sql`, and set `TEST_DATABASE_URL` and `REQUIRE_DB=1`.

**Open points**
- The first CI run on GitHub has not been checked.
- The first commit carries a `Co-Authored-By: Claude` trailer, which makes Claude show as a contributor. Removing it needs an amend and force-push, not done yet.

---

## 10. Planned: gateway and Redis cache

**Progress:** Phase A (Redis cache) is built and verified, not yet committed. Phase B (gateway) and Phase C (measure and document) are not started.

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
| Gateway | New `cmd/gateway` and `internal/gateway`, built on the standard library `httputil.ReverseProxy` | No framework; the gateway is a context of its own and imports only `platform` |
| Gateway features | Routing (`/api/*` and `/healthz` to analytics); access log with method, path, status, duration, client IP and request id (the id is passed upstream in `X-Request-Id`); per-IP token-bucket rate limit returning a 429 problem response with `Retry-After`; server timeouts and upstream dial and response-header timeouts returning 502 or 504 problem responses | The four features requested |
| Not in the gateway | Auth, load balancing, TLS, caching, retries | Out of scope; one upstream only |
| Rate-limit key | The connection's remote IP; `X-Forwarded-For` is not trusted | Nothing sits in front of the gateway. Idle limiter entries are evicted so memory stays bounded |
| Ports | Gateway on `:8080`, so the web dev proxy is unchanged; analytics moves to `:8081` via `ADDR` | The browser only ever talks to the gateway |
| Docker | `docker-compose.yml` at the repo root runs Redis only (`redis:7-alpine`, bound to localhost:6379). Pagila keeps its own compose in its own repo | Docker images for the apps stay out of scope |

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
- Quick timing, one run on a laptop (curl total time): top films cold 9 ms, warm about 1 ms, versus about 7 ms without the cache. Phase C repeats this properly.
- Deviations: the generic cache logic was first written inside the analytics adapter, then moved to `platform/cache` at your request (all tests and the end-to-end check were repeated after the move). `go get` first raised the module to Go 1.26; the dependencies were pinned instead (`go-redis` v9.14.0, `miniredis` v2.35.0, `x/sync` v0.17.0) so `go.mod` stays at Go 1.25.0 and CI is unaffected. Port 6379 was taken by a local `redis-server` on the machine, so the Docker host port became configurable.
- Not done in this phase: the log line shows hit, miss or coalesced per query, but there are no counters or metrics endpoint.

**Phase B: Gateway**
- `internal/gateway` (proxy, request id, access log, rate limiter, error responses) and `cmd/gateway`; Makefile runs Redis, analytics, gateway and web; config for ports and limits.
- Tests with `httptest` upstreams: routing, request id passed upstream, access log fields, 429 with `Retry-After`, 504 on a slow upstream, 502 on a dead upstream, and limiter eviction.
- Verify: `go test ./...`, then through the gateway every endpoint returns the same JSON as calling analytics directly, and the web app works unchanged.

**Phase C: Measure and document**
- A small load script that compares cold and warm requests through the gateway; record the numbers in the README.
- README: updated architecture diagram, configuration table, key decisions, and the limits (static data, one upstream). Update this plan and `.env.example`.
- CI is unchanged: no Redis service is needed because the unit tests use `miniredis`.

### Acceptance criteria (draft, confirm before Phase A)

1. Through the gateway the web app works unchanged, and every response is byte-identical to today's.
2. With `REDIS_URL` set, a repeated identical request is served from Redis and Postgres is queried once, shown by a test and a log line.
3. A cached value expires after `CACHE_TTL` and is then queried again.
4. Concurrent identical cold requests cause one Postgres query.
5. With Redis stopped, the API still returns correct data and logs the failure.
6. With `REDIS_URL` unset, behaviour is exactly as today.
7. The gateway writes one access-log line per request with method, path, status, duration, client IP and request id, and passes the request id upstream.
8. Exceeding the rate limit returns a 429 problem response with `Retry-After`; a slow upstream returns a 504 and a dead upstream a 502.
9. The architecture test still passes, `gateway` imports only `platform`, and CI stays green with no database or Redis.
10. The README reports the measured cold and warm times honestly, and says the data is static.

### Risks

- Caching a 19 ms read-only workload adds a moving part. This is a demonstration, and the README says so.
- The cached JSON is coupled to the domain type shapes; the key version prefix is the guard.
- `make dev` now runs four things (Redis, analytics, gateway, web), so failures are harder to read; the access log and request id are there to help.
- New dependencies: `go-redis/v9` (runtime) and `miniredis` (tests only).

---

## 11. Next steps, in order

1. Check the first CI run on GitHub and fix what fails.
2. Phase 2: gateway and Redis cache (section 10): confirm the acceptance criteria, then build Phase A, B, C.
3. Optional: add a Pagila database to CI (section 9).
4. Later: splitting the packages into binaries and databases.
