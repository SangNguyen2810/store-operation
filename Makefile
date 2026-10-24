.PHONY: install dev dev-api dev-web test test-api test-web lint typecheck build check

install:
	cd web && npm install

# Runs the API (:8080) and the web dev server (:5173) together. Ctrl-C stops both.
dev:
	$(MAKE) -j2 dev-api dev-web

dev-api:
	cd api && go run ./cmd/api

dev-web:
	cd web && npm run dev

test: test-api test-web

# Integration tests read the live Pagila container and skip when it is unreachable.
test-api:
	cd api && go test ./... -count=1

test-web:
	cd web && npm test

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
