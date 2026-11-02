COMPOSE := docker compose -f infra/docker-compose.yml
GO_SERVICES := analytics stores ranges gateway redis
# The services whose Postgres adapters have tests against a real database.
DB_SERVICES := analytics stores

.PHONY: seed-presets install dev dev-analytics dev-stores dev-ranges dev-gateway dev-redis dev-web up up-all down redis-up redis-down test test-analytics test-stores test-ranges test-gateway test-redis test-web cover-analytics cover-stores cover-ranges cover-gateway cover-redis cover-web e2e e2e-install k6-smoke k6-load k6-compare lint typecheck build check ci ci-web ci-docker ci-full test-integration

install:
	cd app/web && npm install

# The web dev server (:5173), talking to the gateway on :8080. Start the backend first with `make up`.
dev: dev-web

# One service as a local process, for working on it while the rest runs in Docker (`make up`). Each needs its settings; see .env.example.
dev-analytics:
	cd app/analytics && RANGES_URL=http://localhost:8083 go run ./cmd/analytics

dev-stores:
	cd app/stores && ADDR=:8081 go run ./cmd/stores

dev-ranges:
	cd app/ranges && go run ./cmd/ranges

dev-gateway:
	cd app/gateway && go run ./cmd/gateway

# The redis service (:8082) in front of a Redis from `make redis-up`. Start analytics with CACHE_URL=http://localhost:8082 to use it.
dev-redis:
	cd app/redis && go run ./cmd/redis

dev-web:
	cd app/web && npm run dev

# The backend in Docker (Redis, redis service, stores, ranges, analytics, gateway) with the cache on; the gateway is http://localhost:8080.
# Needs Pagila on localhost:5432. Then `make dev` for the web app.
up:
	$(COMPOSE) up -d --build --wait redis redis-service stores ranges analytics gateway

# Everything in Docker, including the web app: http://localhost:3000.
up-all:
	$(COMPOSE) up -d --build --wait

down:
	$(COMPOSE) down

# Optional query cache: `make redis-up`, then `make dev-redis` and `CACHE_URL=http://localhost:8082 make dev-analytics`.
redis-up:
	$(COMPOSE) up -d --wait redis

redis-down:
	$(COMPOSE) rm -sf redis

# Stores infra/presets.json as the date-range list the ranges service offers (config:presets:v1 in Redis, no expiry).
# The ranges service picks it up within a minute. `redis-cli del config:presets:v1` goes back to the built-in list.
seed-presets:
	$(COMPOSE) exec -T redis redis-cli -x SET config:presets:v1 < infra/presets.json

test: test-analytics test-stores test-ranges test-gateway test-redis test-web

test-analytics:
	cd app/analytics && go test ./... -count=1

test-stores:
	cd app/stores && go test ./... -count=1

test-ranges:
	cd app/ranges && go test ./... -count=1

test-gateway:
	cd app/gateway && go test ./... -count=1

test-redis:
	cd app/redis && go test ./... -count=1

# Tests plus the coverage floors (90% domain/app, 80% elsewhere).
cover-analytics:
	cd app/analytics && ./scripts/covergate.sh

# Tests plus the coverage floors (90% domain/app, 80% elsewhere).
cover-stores:
	cd app/stores && ./scripts/covergate.sh

# Tests plus the coverage floors (90% domain/app, 80% elsewhere).
cover-ranges:
	cd app/ranges && ./scripts/covergate.sh

# Tests plus the coverage floor (80%).
cover-gateway:
	cd app/gateway && ./scripts/covergate.sh

# Tests plus the coverage floor (80%).
cover-redis:
	cd app/redis && ./scripts/covergate.sh

test-web:
	cd app/web && npm test

# Tests plus the coverage floors (85% lines and statements, 80% functions and branches).
cover-web:
	cd app/web && npm run test:coverage

# One-off download of the browser the end-to-end tests use.
e2e-install:
	cd app/web && npx playwright install chromium

# End-to-end tests in a real browser. They start their own services, gateway (:8180) and web build (:4173)
# against the Pagila on localhost:5432 (override with E2E_DATABASE_URL), so `make dev` can keep running.
e2e:
	cd app/web && npm run e2e

# k6 API tests (`brew install k6`). They talk to a running gateway: BASE_URL defaults to http://localhost:8080.
k6-smoke:
	k6 run loadtest/smoke.js

k6-load:
	k6 run loadtest/load.js

# Starts its own analytics services, without and with Redis, and compares them (needs `make redis-up`).
k6-compare:
	./loadtest/compare.sh

lint:
	cd app/analytics && gofmt -l . | (! grep .) && go vet ./...
	cd app/stores && gofmt -l . | (! grep .) && go vet ./...
	cd app/ranges && gofmt -l . | (! grep .) && go vet ./...
	cd app/gateway && gofmt -l . | (! grep .) && go vet ./...
	cd app/redis && gofmt -l . | (! grep .) && go vet ./...
	cd app/web && npm run lint

typecheck:
	cd app/web && npm run typecheck

build:
	cd app/analytics && go build -o bin/analytics ./cmd/analytics
	cd app/stores && go build -o bin/stores ./cmd/stores
	cd app/ranges && go build -o bin/ranges ./cmd/ranges
	cd app/gateway && go build -o bin/gateway ./cmd/gateway
	cd app/redis && go build -o bin/redis-service ./cmd/redis
	cd app/web && npm run build

# Lint, typecheck, the Go and web tests, and a build. CI also runs the coverage gates, e2e and k6.
check: lint typecheck test build

# The same checks GitHub runs (.github/workflows/ci.yml calls these targets), so a green `make ci` means a green pipeline.
# Per Go service: gofmt, vet, tests with the coverage floors, tests with the race detector, and a build. e.g. `make ci-go-analytics`.
ci-go-%:
	cd app/$* && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	cd app/$* && go vet ./...
	cd app/$* && ./scripts/covergate.sh
	cd app/$* && go test ./... -race -count=1
	cd app/$* && go build ./...

# The Postgres adapters against a real Pagila (build tag `integration`). DATABASE_URL defaults to localhost:5432; a missing database fails the run.
test-integration:
	@for svc in $(DB_SERVICES); do \
		(cd app/$$svc && DATABASE_URL=$${DATABASE_URL:-postgres://postgres:123456@localhost:5432/postgres} go test -tags integration -count=1 ./internal/*/adapters/postgres/) || exit 1; \
	done

# Lint, typecheck, tests with the coverage floors, and the production build. Run `npm ci` in app/web first (CI does).
ci-web:
	cd app/web && npm run lint && npm run typecheck && npm run test:coverage && npm run build

# Builds every image in the stack.
ci-docker:
	$(COMPOSE) build

# Everything that needs no database: all Go services, the web app, and the images.
ci: $(addprefix ci-go-,$(GO_SERVICES)) ci-web ci-docker

# `make ci` plus the browser tests and the k6 smoke test through the gateway. Needs Pagila on localhost:5432 and k6.
ci-full: ci
	$(MAKE) test-integration
	$(MAKE) e2e
	$(MAKE) up
	k6 run loadtest/smoke.js; status=$$?; $(MAKE) down; exit $$status
