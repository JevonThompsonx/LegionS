#!/bin/bash
# Coverage gate for gamepack.
#
# Pure-logic packages must meet MIN_COVERAGE. Hardware/SSH-coupled packages
# (proton, state, unpack) are EXEMPT: they shell out to Steam/Proton/ROG
# services and need live-device integration testing, not unit coverage.
#
# 80% is the long-term target (per tdd-workflow skill); MIN_COVERAGE is the
# enforced floor that must never regress. Raise it as coverage grows.
set -u
cd "$(dirname "$0")/.."

MIN_COVERAGE="${MIN_COVERAGE:-40}"
EXEMPT="proton|state|unpack"

fail=0
echo "Coverage gate (min ${MIN_COVERAGE}% on gated packages; exempt: ${EXEMPT}):"
while IFS= read -r line; do
  pkg=$(echo "$line" | grep -oE "internal/[a-z-]+" | head -1)
  cov=$(echo "$line" | grep -oE "coverage: [0-9.]+%" | grep -oE "[0-9.]+")
  [ -z "$pkg" ] && continue
  if echo "$pkg" | grep -qE "$EXEMPT"; then
    echo "  exempt  $pkg: ${cov}%"
    continue
  fi
  if awk -v c="$cov" -v m="$MIN_COVERAGE" 'BEGIN { exit !(c >= m) }'; then
    echo "  ok      $pkg: ${cov}%"
  else
    echo "  FAIL    $pkg: ${cov}% (need >= ${MIN_COVERAGE}%)"
    fail=1
  fi
done < <(go test ./internal/... -cover 2>&1 | grep "coverage: [0-9]")

if [ "$fail" = "1" ]; then
  echo "COVERAGE GATE FAILED — add tests before committing."
  exit 1
fi
echo "COVERAGE GATE PASSED."
