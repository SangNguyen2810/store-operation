.PHONY: install dev dev-api dev-web redis-up redis-down test test-api test-web test-api-integration cover-api cover-web e2e e2e-install k6-smoke k6-load k6-compare lint typecheck build check

install:
	cd web && npm install

# Runs the API (:8080) and the web dev server (:5173) together. Ctrl-C stops both.
dev:
	$(MAKE) -j2 dev-api dev-web

dev-api:
	cd api && go run ./cmd/api

dev-web:
	cd web && npm run dev

# Optional query cache: `make redis-up`, then `REDIS_URL=redis://localhost:6379 make dev`.
redis-up:
	docker compose up -d --wait redis

redis-down:
	docker compose down

test: test-api test-web

# Integration tests read the live Pagila container and skip when it is unreachable.
test-api:
	cd api && go test ./... -count=1

# The same run as CI: a missing Pagila or Redis fails instead of skipping.
# Redis for the tests: `REDIS_PORT=6380 make redis-up` (uses database 15 on :6380).
test-api-integration:
	cd api && REQUIRE_DB=1 go test ./... -race -count=1

# Tests plus the coverage floors (90% domain/app, 80% elsewhere). Needs Pagila and Redis up.
cover-api:
	cd api && REQUIRE_DB=1 ./scripts/covergate.sh

test-web:
	cd web && npm test

# Tests plus the coverage floors (85% lines and statements, 80% functions and branches).
cover-web:
	cd web && npm run test:coverage

# End-to-end tests in a real browser. They start their own API (:8180) and web build (:4173)
# against the Pagila on localhost:5432 (override with E2E_DATABASE_URL), so `make dev` can keep running.
e2e-install:
	cd web && npx playwright install chromium

e2e:
	cd web && npm run e2e

# k6 API tests (`brew install k6`). They talk to a running API: BASE_URL defaults to http://localhost:8080.
k6-smoke:
	k6 run loadtest/smoke.js

k6-load:
	k6 run loadtest/load.js

# Starts its own APIs, without and with Redis, and compares them (needs `REDIS_PORT=6380 make redis-up`).
k6-compare:
	./loadtest/compare.sh

lint:
	cd api && gofmt -l . | (! grep .) && go vet ./...
	cd web && npm run lint

typecheck:
	cd web && npm run typecheck

build:
	cd api && go build -o bin/api ./cmd/api
	cd web && npm run build

# Everything CI would run.
check: lint typecheck test build
