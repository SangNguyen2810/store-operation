# Store Operations Dashboard: Project Plan

A portfolio project for a Principal Frontend Engineer interview (React/Vue role), built on the Pagila DVD-rental database. It is meant to be walked through on screen as a code sample, so the plan favours clear structure and explainable trade-offs over feature count.

**Status:** Phase 1 is built and verified locally. Publishing it to GitHub with CI is in progress and not finished. Phase 2 and the microservices split are not started.

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

### Backend (phase 1): `analytics` package
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
- Backend: a second package, `rentals` (`GET /rentals` with cursor, `GET /rentals/{id}`), with its own SQL and interface.
- Needs its own acceptance criteria before work starts. "Overdue" needs the as-of date (rentals end 2022-08-23).

### Later (not started): microservices
Turn each package into its own `cmd/<service>` binary with its own pool, add a gateway, then split the database. See the README for the seam.

### Out of scope
Auth, caching (in-memory first, Redis later), Docker images for the app, Kubernetes, returns and late-fee handling, customer-facing site.

---

## 3. Architecture

```
dashboard/
  api/                       Go
    cmd/api/                 main: config, pool, server, graceful shutdown
    internal/analytics/      Service + Repo interface + Postgres repo. Owns all SQL.
    internal/httpapi/        parsing, validation, JSON. No SQL.
    internal/db/             read-only pool (default_transaction_read_only=on, 10s statement timeout)
    internal/config/         environment settings
  web/                       React 19 + TypeScript (strict) + Vite
    src/api/                 typed client, one function per endpoint
    src/features/overview/   filters (URL state), query hooks, page and its parts
    src/components/          AsyncSection, Skeleton, TimeSeriesChart (the only file that imports Recharts)
  Makefile, .env.example, README.md, docs/PLAN.md
```

Key seams: `analytics` reaches the database only through `Repo`; `httpapi` depends on a small `Analytics` interface; the chart library is isolated; `AsyncSection` is the single place for query states.

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
- Nothing has been committed yet.

---

## 5. In progress: publish to GitHub with CI

**Target:** https://github.com/SangNguyen2810/store-operation (it appeared empty when checked). Only `dashboard/` goes in, as the root of that repo. Your pagila checkout and its `origin` (upstream `devrimgunduz/pagila`) are not touched, and nothing is pushed there.

**Approach**
1. Build the new repo in a scratch copy of `dashboard/` (without `node_modules`, `dist`, `bin`), so the pagila working tree stays clean.
2. Add `.github/workflows/ci.yml` with two jobs:
   - **api:** set up Go from `api/go.mod`, check `gofmt`, `go vet`, run `go test` with a Postgres service loaded with Pagila.
   - **web:** Node 22, `npm ci`, `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`.
3. Fix README links that point outside the folder, and tell people to run Pagila from its own repo.
4. Commit and push to `main`, then watch the first CI run and fix what fails.

**Findings so far about running Pagila in CI**
- The newest upstream Pagila does not load on `postgres:17` or `postgres:18`: its schema needs `uuidv7()` (Postgres 18) and the pgvector extension.
- Plan: pin CI to the Pagila commit your local copy is based on, `5ba5a57` (2024-12-02), and use `postgres:17` like your compose file.
- That pinned combination was **not yet verified**. My check was interrupted.
- Integration tests currently skip when the database is unreachable, which would let CI pass without testing the data. Plan: add `REQUIRE_DB=1` in CI so an unreachable database fails the run. This change is in the scratch copy only, not yet in `dashboard/`.

**Open points**
- `gh` is not installed. Pushing uses plain `git` over HTTPS with your keychain credentials, which I have not tested.
- A throwaway test container, `pagila-ci-test` (port 55432), may still be running from the interrupted checks. It is separate from your `pagila` container and can be removed with `docker rm -f pagila-ci-test`.
- Whether to commit as one commit or split backend and frontend.

---

## 6. Next steps, in order

1. Finish section 5: verify the pinned Pagila loads, add the CI workflow, push, get CI green.
2. Phase 2: write requirements and acceptance criteria for the Rentals Explorer, plan, then build.
3. Microservices phase: split the packages into binaries, add a gateway, then split databases.
