#!/usr/bin/env bash
# Runs the Go tests with coverage and fails when a package is below its floor:
# 90% for domain and app (the business rules), 80% for every other package.
# Not gated: cmd/api (the composition root) and the two packages that only talk
# to Postgres, adapters/postgres and platform/db, which have no unit tests.
set -euo pipefail
cd "$(dirname "$0")/.."

out=$(go test -count=1 -cover ./... 2>&1) || { echo "$out"; exit 1; }
echo "$out" | grep -v 'no test files'

echo "$out" | awk '
  /coverage: [0-9.]+%/ {
    pkg = ""
    for (i = 1; i <= NF; i++) if ($i ~ /\/api\//) pkg = $i
    sub(/.*\/api\//, "", pkg)
    pct = $0; sub(/.*coverage: /, "", pct); sub(/%.*/, "", pct)
    if (pkg ~ /^cmd\// || pkg ~ /adapters\/postgres$/ || pkg ~ /platform\/db$/) next
    floor = (pkg ~ /\/(domain|app)$/) ? 90 : 80
    if (pct + 0 < floor) { printf "FAIL %s: %.1f%% < %d%%\n", pkg, pct, floor; bad = 1 }
    checked++
  }
  END { if (bad) exit 1; printf "coverage gate passed (%d packages)\n", checked }'
