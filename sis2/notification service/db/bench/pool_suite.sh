#!/usr/bin/env bash
# Connection pooling: 400 app connections straight to PostgreSQL vs through PgBouncer (transaction mode, pool 8) on a 1-core DB node.
# Usage: bench/pool_suite.sh v2w [RATE]
set -euo pipefail
V=$1; R=${2:-2000}
HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
infra/cluster.sh pgb-stop 2>/dev/null || true; infra/cluster.sh pgb-start; sleep 1
run() { # label port clients rate extra
  VER=$V PORT=$2 CLIENTS=$3 RATE=$4 EXTRA="${5:-}" DUR=30 WARM=10 OUT=results/pool_$1 bench/run.sh | head -1
}
run direct_c400_max   5432 400 0
run pgb_c400_max      6432 400 0
run direct_c400_rate  5432 400 $R
run pgb_c400_rate     6432 400 $R
run direct_reconnect  5432 32  0 "-C"
run pgb_reconnect     6432 32  0 "-C"
# backend count seen by PostgreSQL under 400 clients (sampled in the middle of a run)
infra/cluster.sh pgb-stop
