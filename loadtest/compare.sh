#!/usr/bin/env bash
# Runs load.js three times and prints the latencies side by side: against the API without the Redis
# cache, against the API with it, and against the cached API through the gateway (rate limit off, so
# the column shows what the extra hop costs). Starts and stops its own processes (ports 8191, 8192
# and 8190) and uses Redis database 13 on REDIS_URL (default: the Docker Redis from `REDIS_PORT=6380 make redis-up`).
# It needs Pagila reachable (DATABASE_URL, default: the local one) and k6 on the PATH.
set -euo pipefail
cd "$(dirname "$0")/.."

REDIS_URL=${REDIS_URL:-redis://localhost:6380/13}
out=$(mktemp -d)
pids=()
cleanup() { for p in "${pids[@]:-}"; do kill "$p" 2>/dev/null || true; done; rm -rf "$out"; }
trap cleanup EXIT

(cd app/api && go build -o "$out/api" ./cmd/api)
(cd app/gateway && go build -o "$out/gateway" ./cmd/gateway)

start() { # port, redis url or empty
  ADDR=":$1" REDIS_URL="$2" "$out/api" >"$out/api-$1.log" 2>&1 &
  pids+=("$!")
  for _ in $(seq 50); do curl -sf "http://localhost:$1/healthz" >/dev/null && return; sleep 0.2; done
  echo "API on :$1 did not start:"; cat "$out/api-$1.log"; exit 1
}

# Start from a cold cache so the first requests miss.
redis-cli -u "$REDIS_URL" flushdb >/dev/null 2>&1 || docker compose -f infra/docker-compose.yml exec -T redis redis-cli -n 13 flushdb >/dev/null

start 8191 ""
start 8192 "$REDIS_URL"

ADDR=:8190 UPSTREAM_URL=http://localhost:8192 RATE_LIMIT_RPS=0 "$out/gateway" >"$out/gateway.log" 2>&1 &
pids+=("$!")
for _ in $(seq 50); do curl -sf http://localhost:8190/healthz >/dev/null && break; sleep 0.2; done
curl -sf http://localhost:8190/healthz >/dev/null || { echo "gateway did not start:"; cat "$out/gateway.log"; exit 1; }

run() { # label, port, cache tag
  k6 run --quiet --summary-export="$out/$1.json" -e BASE_URL="http://localhost:$2" -e CACHE="$3" loadtest/load.js >"$out/$1.txt" 2>&1 || true
}
echo "Running load.js without the cache (about 45 s)…"; run off 8191 off
echo "Running load.js with the cache (about 45 s)…";    run on 8192 on
echo "Running load.js with the cache, through the gateway (about 45 s)…"; run gateway 8190 on

python3 - "$out" <<'PY'
import json, sys
d = sys.argv[1]
def load(n): return json.load(open(f"{d}/{n}.json"))["metrics"]
off, on, gw = load("off"), load("on"), load("gateway")
def val(m, k, f): return m.get(f"http_req_duration{{name:{k}}}", m.get("http_req_duration"))[f] if k else m["http_req_duration"][f]
rows = [("all requests", None), ("/api/kpis", "/api/kpis"), ("/api/revenue", "/api/revenue"), ("/api/top/films", "/api/top/films")]
print(f"\n{'':18}{'no cache':>22}{'with cache':>22}{'cache + gateway':>22}")
print(f"{'':18}" + f"{'avg':>8}{'p95':>8}{'max':>6}" * 3)
for label, k in rows:
    cells = []
    for m in (off, on, gw):
        key = f"http_req_duration{{name:{k}}}" if k else "http_req_duration"
        s = m.get(key)
        cells.append("%8.1f%8.1f%6.0f" % (s["avg"], s["p(95)"], s["max"]) if s else " " * 22)
    print(f"{label:18}{cells[0]:>22}{cells[1]:>22}{cells[2]:>22}")
print("\nrequests: %d / %d / %d; failed: %.2f%% / %.2f%% / %.2f%%" % (
    off["http_reqs"]["count"], on["http_reqs"]["count"], gw["http_reqs"]["count"],
    off["http_req_failed"]["value"] * 100, on["http_req_failed"]["value"] * 100, gw["http_req_failed"]["value"] * 100))
print("(milliseconds; one local run is an indication, not a benchmark)")
PY
