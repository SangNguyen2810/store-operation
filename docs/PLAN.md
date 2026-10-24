# Store Operations Dashboard: Project Plan

A portfolio project for a Principal Frontend Engineer interview (React/Vue role), built on the Pagila DVD-rental database. It is meant to be walked through on screen as a code sample, so the plan favours clear structure and explainable trade-offs over feature count.

**Status:** Phase 1 is built, verified, and published to GitHub with a CI workflow. CI has no database yet, so the integration tests skip there. Phase 2 and the microservices split are not started.

---

## 1. Goal and constraints

- Show frontend architecture, design choices and problem solving in a code review setting.
- Only 1-2 core frontend features at a time.
- Backend in Go, as simple HTTP servers. No saga, no Redis, no events. Start with one deployable and 1-2 packages, split into microservices later.
- Pagila is **read-only**. The dashboard must never write to it, and everything lives in the `dashboard/` folder.
- Frontend stack: React + TypeScript + Vite + TanStack Query.

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

### Phase 2 (not started): Rentals Explorer
- Frontend: server-side paginated, sortable, filterable table of rentals (status out/returned/overdue, store, category, customer search), cursor pagination or virtualised list, detail drawer with state in the URL, a reusable generic `DataTable<T>`.
- Backend: a second context, `internal/rentals/` (`GET /rentals` with cursor, `GET /rentals/{id}`), laid out like `analytics` (domain, app, adapters) and plugged in with its own `Register(mux, ...)`.
- Needs its own acceptance criteria before work starts. "Overdue" needs the as-of date (rentals end 2022-08-23).

### Later (not started): microservices
Turn each context into its own `cmd/<service>` binary with its own pool, add a gateway, then split the database. See the README for the seam.

### Out of scope
Auth, caching (in-memory first, Redis later), Docker images for the app, Kubernetes, returns and late-fee handling, customer-facing site.

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
        http/                        (package httpadapter) routes, parsing, validation, response types
    platform/                        shared plumbing, knows no context
      db/                            read-only pool (default_transaction_read_only=on, 10s statement timeout)
      config/                        environment settings
      httpx/                         JSON and problem responses, request logging, /healthz, /api/meta
    archtest/                        test that enforces the import rules below
web/                                 React 19 + TypeScript (strict) + Vite
  src/api/                           typed client, one function per endpoint
  src/features/overview/             filters (URL state), query hooks, page and its parts
  src/components/                    AsyncSection, Skeleton, TimeSeriesChart (the only file that imports Recharts)
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
- **Packages in one `go.mod`, not one module per context.** Separate modules and a `go.work` only pay off once there are two or more real contexts. Revisit when `rentals` exists.
- **Response types live in the HTTP adapter.** The domain has no `json` tags, so the domain can change without breaking the API, and the transport can change (for example to gRPC) without touching the domain.
- **No extra repository/service/controller layers** on top of hexagonal. They would add files without adding a boundary.
- **Shared endpoints** (`/healthz`, `/api/meta`) are not analytics concerns, so `cmd/api` registers them next to each context's `Register(mux, ...)`.

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

## 7. Done: published to GitHub with CI

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

## 8. Next steps, in order

1. Check the first CI run on GitHub and fix what fails.
2. Phase 2: write requirements and acceptance criteria for the Rentals Explorer, plan, then build.
3. Optional: add a Pagila database to CI (section 5).
4. Microservices phase: split the packages into binaries, add a gateway, then split databases.
