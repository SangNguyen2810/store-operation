#!/usr/bin/env bash
# Runs load.js four times and prints the latencies side by side, with the cache hit ratio of each run:
#   no cache      the API straight to Postgres
#   cache         the API with the redis service and Redis
#   + gateway     the same cached API, through the gateway (rate limit off, so the column shows the cost of the extra hop)
#   service down  the API with a CACHE_URL nobody listens on: only its in-memory copy and Postgres
# It starts and stops its own processes (ports 8190 to 8194) and uses Redis database 13 on REDIS_URL
# (default: the Docker Redis from `REDIS_PORT=6380 make redis-up`). It needs Pagila reachable
# (DATABASE_URL, default: the local one) and k6 on the PATH.
set -euo pipefail
cd "$(dirname "$0")/.."

REDIS_URL=${REDIS_URL:-redis://localhost:6380/13}
out=$(mktemp -d)
pids=()
cleanup() { for p in "${pids[@]:-}"; do kill "$p" 2>/dev/null || true; done; rm -rf "$out"; }
trap cleanup EXIT

(cd app/api && go build -o "$out/api" ./cmd/api)
(cd app/gateway && go build -o "$out/gateway" ./cmd/gateway)
(cd app/redis && go build -o "$out/redis-service" ./cmd/redis)

wait_for() { # url, name, log
  for _ in $(seq 50); do curl -sf "$1" >/dev/null && return; sleep 0.2; done
  echo "$2 did not start:"; cat "$3"; exit 1
}
start_api() { # port, cache url or empty
  ADDR=":$1" CACHE_URL="$2" "$out/api" >"$out/api-$1.log" 2>&1 &
  pids+=("$!")
  wait_for "http://localhost:$1/healthz" "API on :$1" "$out/api-$1.log"
}

# Start from a cold cache so the first requests miss.
redis-cli -u "$REDIS_URL" flushdb >/dev/null 2>&1 || docker compose -f infra/docker-compose.yml exec -T redis redis-cli -n 13 flushdb >/dev/null

ADDR=:8193 REDIS_URL="$REDIS_URL" "$out/redis-service" >"$out/redis-service.log" 2>&1 &
pids+=("$!")
wait_for http://localhost:8193/healthz "the redis service" "$out/redis-service.log"

start_api 8191 ""
start_api 8192 http://localhost:8193
start_api 8194 http://localhost:1 # nothing listens: the service is "down"

ADDR=:8190 UPSTREAM_URL=http://localhost:8192 RATE_LIMIT_RPS=0 "$out/gateway" >"$out/gateway.log" 2>&1 &
pids+=("$!")
wait_for http://localhost:8190/healthz "the gateway" "$out/gateway.log"

# Cache outcomes an API logged since line $2 of its log: "hit miss coalesced fallback".
outcomes() { tail -n +"$(($2 + 1))" "$out/api-$1.log" | grep -o 'outcome=[a-z]*' | sort | uniq -c | awk '{c[$2]=$1} END {printf "%d %d %d %d", c["outcome=hit"], c["outcome=miss"], c["outcome=coalesced"], c["outcome=fallback"]}'; }

run() { # label, port, cache tag, label of the API whose log to read (or empty)
  local from=0
  [ -n "$4" ] && from=$(wc -l <"$out/api-$4.log")
  k6 run --quiet --summary-export="$out/$1.json" -e BASE_URL="http://localhost:$2" -e CACHE="$3" loadtest/load.js >"$out/$1.txt" 2>&1 || true
  if [ -n "$4" ]; then outcomes "$4" "$from" >"$out/$1.outcomes"; else echo "0 0 0 0" >"$out/$1.outcomes"; fi
}
echo "Running load.js without the cache (about 45 s)…";        run off 8191 off ""
echo "Running load.js with the cache (about 45 s)…";           run on 8192 on 8192
echo "Running load.js with the cache, through the gateway (about 45 s)…"; run gateway 8190 on 8192
echo "Running load.js with the redis service down (about 45 s)…"; run down 8194 down 8194

python3 - "$out" <<'PY'
import json, sys
d = sys.argv[1]
names = [("off", "no cache"), ("on", "cache"), ("gateway", "+ gateway"), ("down", "service down")]
def load(n): return json.load(open(f"{d}/{n}.json"))["metrics"]
ms = {n: load(n) for n, _ in names}
rows = [("all requests", None), ("/api/kpis", "/api/kpis"), ("/api/revenue", "/api/revenue"), ("/api/top/films", "/api/top/films")]
print(f"\n{'':18}" + "".join(f"{label:>22}" for _, label in names))
print(f"{'':18}" + f"{'avg':>8}{'p95':>8}{'max':>6}" * len(names))
for label, k in rows:
    cells = []
    for n, _ in names:
        key = f"http_req_duration{{name:{k}}}" if k else "http_req_duration"
        s = ms[n].get(key)
        cells.append("%8.1f%8.1f%6.0f" % (s["avg"], s["p(95)"], s["max"]) if s else " " * 22)
    print(f"{label:18}" + "".join(f"{c:>22}" for c in cells))
print(f"\n{'requests':18}" + "".join(f"{ms[n]['http_reqs']['count']:>22d}" for n, _ in names))
print(f"{'failed':18}" + "".join(f"{ms[n]['http_req_failed']['value'] * 100:>21.2f}%" for n, _ in names))
def counts(n): return [int(x) for x in open(f"{d}/{n}.outcomes").read().split()]
print()
cells = []
for n, _ in names:
    hit, miss, coal, fb = counts(n)
    total = hit + miss + coal + fb
    cells.append("-" if total == 0 else "%d/%d/%d/%d" % (hit, miss, coal, fb))
print(f"{'hit/miss/coal/mem':18}" + "".join(f"{c:>22}" for c in cells))
ratios = []
for n, _ in names:
    hit, miss, coal, fb = counts(n)
    total = hit + miss + coal + fb
    ratios.append("-" if total == 0 else "%.1f%%" % ((hit + fb) / total * 100))
print(f"{'served from cache':18}" + "".join(f"{c:>22}" for c in ratios))
print("\n(milliseconds; hit = redis service, mem = the API's in-memory copy, coal = joined a query already running;")
print(" one local run is an indication, not a benchmark)")
PY
