#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT_PATH="${1:-$ROOT_DIR/perf-budget-report.txt}"
MAX_NS_PER_OP="${MAX_NS_PER_OP:-20000000}"
MAX_BYTES_PER_OP="${MAX_BYTES_PER_OP:-20000}"

echo "Running performance budget benchmark..." >&2
BENCH_OUTPUT="$(cd "$ROOT_DIR/go" && go test ./pkg/monitor -run '^$' -bench BenchmarkService_UpdateMetrics -benchmem -count=1)"
echo "$BENCH_OUTPUT" > "$REPORT_PATH"

BENCH_LINE="$(printf '%s\n' "$BENCH_OUTPUT" | awk '/BenchmarkService_UpdateMetrics/{line=$0} END{print line}')"
if [[ -z "$BENCH_LINE" ]]; then
  {
    echo "status=warn"
    echo "reason=benchmark line not found"
  } >> "$REPORT_PATH"
  echo "WARN: benchmark line not found; see $REPORT_PATH" >&2
  exit 0
fi

NS_PER_OP="$(printf '%s\n' "$BENCH_LINE" | awk '{for(i=1;i<=NF;i++){if($i ~ /ns\/op$/){print $(i-1); exit}}}')"
BYTES_PER_OP="$(printf '%s\n' "$BENCH_LINE" | awk '{for(i=1;i<=NF;i++){if($i ~ /B\/op$/){print $(i-1); exit}}}')"

status="pass"
if [[ -n "$NS_PER_OP" ]] && (( NS_PER_OP > MAX_NS_PER_OP )); then
  status="warn"
fi
if [[ -n "$BYTES_PER_OP" ]] && (( BYTES_PER_OP > MAX_BYTES_PER_OP )); then
  status="warn"
fi

{
  echo ""
  echo "budget_thresholds:"
  echo "  max_ns_per_op: $MAX_NS_PER_OP"
  echo "  max_bytes_per_op: $MAX_BYTES_PER_OP"
  echo "observed:"
  echo "  ns_per_op: ${NS_PER_OP:-unknown}"
  echo "  bytes_per_op: ${BYTES_PER_OP:-unknown}"
  echo "status=$status"
} >> "$REPORT_PATH"

if [[ "$status" == "warn" ]]; then
  echo "WARN: performance benchmark over soft threshold; see $REPORT_PATH" >&2
else
  echo "PASS: performance benchmark within soft thresholds; see $REPORT_PATH" >&2
fi

exit 0
