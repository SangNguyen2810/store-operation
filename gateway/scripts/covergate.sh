#!/usr/bin/env bash
# Runs the Go tests with coverage and fails when a package is below 80%.
# Not gated: cmd/gateway, the composition root.
set -euo pipefail
cd "$(dirname "$0")/.."

out=$(go test -count=1 -cover ./... 2>&1) || { echo "$out"; exit 1; }
echo "$out" | grep -v 'no test files'

echo "$out" | awk '
  /coverage: [0-9.]+%/ {
    pkg = ""
    for (i = 1; i <= NF; i++) if ($i ~ /\/gateway\//) pkg = $i
    sub(/.*\/gateway\//, "", pkg)
    pct = $0; sub(/.*coverage: /, "", pct); sub(/%.*/, "", pct)
    if (pkg ~ /^cmd\//) next
    if (pct + 0 < 80) { printf "FAIL %s: %.1f%% < 80%%\n", pkg, pct; bad = 1 }
    checked++
  }
  END { if (bad) exit 1; printf "coverage gate passed (%d packages)\n", checked }'
