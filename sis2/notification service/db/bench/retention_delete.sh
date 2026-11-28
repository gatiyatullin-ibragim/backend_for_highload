#!/usr/bin/env bash
# Retention on an unpartitioned table: delete the oldest day (864k rows) and measure time + WAL + bloat.
set -euo pipefail
P="psql -h 127.0.0.1 -U postgres -d notif -Atq"
T0=$(grep T0= results/anchor.txt | cut -d= -f2)
L0=$($P -c "select pg_current_wal_lsn()"); S0=$($P -c "select pg_total_relation_size('notifications')")
a=$(date +%s.%N)
N=$($P -c "with d as (delete from notifications where created_at < timestamptz '$T0' + interval '1 day' returning 1) select count(*) from d")
b=$(date +%s.%N)
L1=$($P -c "select pg_current_wal_lsn()")
$P -c "checkpoint"; DEAD=$($P -c "select n_dead_tup from pg_stat_user_tables where relname='notifications'")
c=$(date +%s.%N); $P -c "vacuum notifications"; d=$(date +%s.%N)
S1=$($P -c "select pg_total_relation_size('notifications')")
echo "{\"method\":\"DELETE\",\"rows\":$N,\"delete_s\":$(echo "$b - $a" | bc),\"wal_mb\":$($P -c "select round(pg_wal_lsn_diff('$L1','$L0')/1048576,1)"),\"dead_tuples_after\":$DEAD,\"vacuum_s\":$(echo "$d - $c" | bc),\"size_mb_before\":$((S0/1048576)),\"size_mb_after_vacuum\":$((S1/1048576))}" | tee results/retention_delete.json
