#!/usr/bin/env bash
# Runs load.js against the API twice, once without and once with the Redis cache, and prints the
# latencies side by side. Starts and stops its own API processes (ports 8191 and 8192) and uses
# Redis database 13 on REDIS_URL (default: the Docker Redis from `REDIS_PORT=6380 make redis-up`).
# It needs Pagila reachable (DATABASE_URL, default: the local one) and k6 on the PATH.
set -euo pipefail
cd "$(dirname "$0")/.."

REDIS_URL=${REDIS_URL:-redis://localhost:6380/13}
out=$(mktemp -d)
pids=()
cleanup() { for p in "${pids[@]:-}"; do kill "$p" 2>/dev/null || true; done; rm -rf "$out"; }
trap cleanup EXIT

(cd api && go build -o "$out/api" ./cmd/api)

start() { # port, redis url or empty
  ADDR=":$1" REDIS_URL="$2" "$out/api" >"$out/api-$1.log" 2>&1 &
  pids+=("$!")
  for _ in $(seq 50); do curl -sf "http://localhost:$1/healthz" >/dev/null && return; sleep 0.2; done
  echo "API on :$1 did not start:"; cat "$out/api-$1.log"; exit 1
}

# Start from a cold cache so the first requests miss.
redis-cli -u "$REDIS_URL" flushdb >/dev/null 2>&1 || docker compose exec -T redis redis-cli -n 13 flushdb >/dev/null

start 8191 ""
start 8192 "$REDIS_URL"

run() { # label, port, cache tag
  k6 run --quiet --summary-export="$out/$1.json" -e BASE_URL="http://localhost:$2" -e CACHE="$3" loadtest/load.js >"$out/$1.txt" 2>&1 || true
}
echo "Running load.js without the cache (about 45 s)…"; run off 8191 off
echo "Running load.js with the cache (about 45 s)…";    run on 8192 on

python3 - "$out" <<'PY'
import json, sys
d = sys.argv[1]
def load(n): return json.load(open(f"{d}/{n}.json"))["metrics"]
off, on = load("off"), load("on")
def val(m, k, f): return m.get(f"http_req_duration{{name:{k}}}", m.get("http_req_duration"))[f] if k else m["http_req_duration"][f]
rows = [("all requests", None), ("/api/kpis", "/api/kpis"), ("/api/revenue", "/api/revenue"), ("/api/top/films", "/api/top/films")]
print(f"\n{'':18}{'no cache':>22}{'with cache':>22}")
print(f"{'':18}{'avg':>8}{'p95':>8}{'max':>6}{'avg':>8}{'p95':>8}{'max':>6}")
for label, k in rows:
    cells = []
    for m in (off, on):
        key = f"http_req_duration{{name:{k}}}" if k else "http_req_duration"
        s = m.get(key)
        cells.append("%8.1f%8.1f%6.0f" % (s["avg"], s["p(95)"], s["max"]) if s else " " * 22)
    print(f"{label:18}{cells[0]:>22}{cells[1]:>22}")
print("\nrequests: no cache %d, with cache %d; failed: %.2f%% / %.2f%%" % (
    off["http_reqs"]["count"], on["http_reqs"]["count"],
    off["http_req_failed"]["value"] * 100, on["http_req_failed"]["value"] * 100))
print("(milliseconds; one local run is an indication, not a benchmark)")
PY
