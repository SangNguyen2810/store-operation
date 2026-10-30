COMPOSE := docker compose -f infra/docker-compose.yml

.PHONY: install dev dev-api dev-gateway dev-web up down redis-up redis-down test test-api test-gateway test-shared test-web cover-api cover-gateway cover-shared cover-web e2e e2e-install k6-smoke k6-load k6-compare lint typecheck build check

install:
	cd app/web && npm install

# Runs the API (:8081), the gateway (:8080) and the web dev server (:5173) together. Ctrl-C stops all.
dev:
	$(MAKE) -j3 dev-api dev-gateway dev-web

dev-api:
	cd app/api && go run ./cmd/api

dev-gateway:
	cd app/gateway && go run ./cmd/gateway

dev-web:
	cd app/web && npm run dev

# The whole stack in Docker (Redis, API, gateway, web): http://localhost:3000. Needs Pagila on localhost:5432.
up:
	$(COMPOSE) up -d --build --wait

down:
	$(COMPOSE) down

# Optional query cache: `make redis-up`, then `REDIS_URL=redis://localhost:6379 make dev`.
redis-up:
	$(COMPOSE) up -d --wait redis

redis-down:
	$(COMPOSE) rm -sf redis

test: test-api test-gateway test-shared test-web

test-api:
	cd app/api && go test ./... -count=1

test-gateway:
	cd app/gateway && go test ./... -count=1

test-shared:
	cd shared && go test ./... -count=1

# Tests plus the coverage floors (90% domain/app, 80% elsewhere).
cover-api:
	cd app/api && ./scripts/covergate.sh

# Tests plus the coverage floor (80%).
cover-gateway:
	cd app/gateway && ./scripts/covergate.sh

cover-shared:
	cd shared && ./scripts/covergate.sh

test-web:
	cd app/web && npm test

# Tests plus the coverage floors (85% lines and statements, 80% functions and branches).
cover-web:
	cd app/web && npm run test:coverage

# One-off download of the browser the end-to-end tests use.
e2e-install:
	cd app/web && npx playwright install chromium

# End-to-end tests in a real browser. They start their own API (:8180) and web build (:4173)
# against the Pagila on localhost:5432 (override with E2E_DATABASE_URL), so `make dev` can keep running.
e2e:
	cd app/web && npm run e2e

# k6 API tests (`brew install k6`). They talk to a running gateway: BASE_URL defaults to http://localhost:8080.
k6-smoke:
	k6 run loadtest/smoke.js

k6-load:
	k6 run loadtest/load.js

# Starts its own APIs, without and with Redis, and compares them (needs `REDIS_PORT=6380 make redis-up`).
k6-compare:
	./loadtest/compare.sh

lint:
	cd app/api && gofmt -l . | (! grep .) && go vet ./...
	cd app/gateway && gofmt -l . | (! grep .) && go vet ./...
	cd shared && gofmt -l . | (! grep .) && go vet ./...
	cd app/web && npm run lint

typecheck:
	cd app/web && npm run typecheck

build:
	cd app/api && go build -o bin/api ./cmd/api
	cd app/gateway && go build -o bin/gateway ./cmd/gateway
	cd app/web && npm run build

# Lint, typecheck, the Go and web tests, and a build. CI also runs the coverage gates, e2e and k6.
check: lint typecheck test build
