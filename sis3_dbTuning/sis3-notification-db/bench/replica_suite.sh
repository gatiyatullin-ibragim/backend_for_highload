#!/usr/bin/env bash
# Read replica: same load, same single-core primary; reads either stay on the primary or go to the replica (port 5433, own core).
# Usage: bench/replica_suite.sh v2w TOTAL_RATE
set -euo pipefail
V=$1; X=${2:-3000}
HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
P="psql -h 127.0.0.1 -p 5432 -U postgres -d notif -Atq"
$P -c "CREATE TABLE IF NOT EXISTS lag_probe (id bigserial PRIMARY KEY, ts timestamptz DEFAULT now())"
if ! pg_isready -h 127.0.0.1 -p 5433 -q; then
  infra/cluster.sh replica-init; infra/cluster.sh replica-start
fi
sleep 3; $P -c "select client_addr, state, sync_state from pg_stat_replication"
RR=$(python3 -c "print(int($X*466/1073))"); WR=$((X-RR))     # read/write share of the mix (466 : 607)
two() { # label read_port
  rm -f /tmp/lag.csv; ( for i in $(seq 1 55); do $P -c "select now()::time(0), coalesce(extract(epoch from replay_lag)*1000,0)::int, pg_wal_lsn_diff(sent_lsn, replay_lsn)::bigint from pg_stat_replication" 2>/dev/null; sleep 1; done > /tmp/lag.csv ) &
  VER=$V MIX=read  PORT=$2   RATE=$RR CLIENTS=48 DUR=40 WARM=10 OUT=results/replica_$1_read  bench/run.sh > /tmp/r.out &
  VER=$V MIX=write PORT=5432 RATE=$WR CLIENTS=16 DUR=40 WARM=10 OUT=results/replica_$1_write bench/run.sh > /tmp/w.out &
  wait; cat /tmp/r.out /tmp/w.out | grep '^\['; cp /tmp/lag.csv results/replica_$1_lag.csv
}
two single_node 5432
two with_replica 5433
echo "--- staleness (idle replica)"; python3 bench/staleness.py 300 | tee results/staleness_idle.json
