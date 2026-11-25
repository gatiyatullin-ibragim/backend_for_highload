#!/usr/bin/env bash
# v2 extras: (1) retention by DROP PARTITION, (2) read-watermark design vs per-row is_read
set -euo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
P="psql -h 127.0.0.1 -U postgres -d notif -Atq"
T0=$(grep T0= results/anchor.txt | cut -d= -f2)
# (1) retention: detach + drop the oldest day
L0=$($P -c "select pg_current_wal_lsn()"); S0=$($P -c "select pg_size_pretty(sum(pg_total_relation_size(inhrelid))) from pg_inherits where inhparent='notifications'::regclass")
PART=notifications_$(date -d "$T0" +%Y%m%d)
a=$(date +%s.%N)
$P -c "ALTER TABLE notifications DETACH PARTITION $PART" -c "DROP TABLE $PART"
b=$(date +%s.%N); L1=$($P -c "select pg_current_wal_lsn()")
echo "{\"method\":\"DROP PARTITION\",\"rows\":864000,\"drop_s\":$(printf "%.3f" $(echo "$b - $a" | bc)),\"wal_mb\":$($P -c "select round(pg_wal_lsn_diff('$L1','$L0')/1048576,3)"),\"dead_tuples_after\":0}" | tee results/retention_drop.json
# (2) watermark
$P -c "CREATE TABLE IF NOT EXISTS user_read_state (user_id bigint PRIMARY KEY, last_read_at timestamptz NOT NULL)"
$P -c "TRUNCATE user_read_state" -c "INSERT INTO user_read_state SELECT g, timestamptz '$T0' + interval '24.5 days' FROM generate_series(1,200000) g" -c "VACUUM ANALYZE user_read_state"
for s in unread_count_wm mark_read_wm; do VER=v2 MIX=one SCRIPT=$s CLIENTS=8 DUR=20 WARM=5 OUT=results/v2_one_$s bench/run.sh | head -1; done
