#!/usr/bin/env bash
# Usage: VER=v1 PORT=5432 RATE=10480 CLIENTS=64 DUR=60 WARM=20 MIX=all OUT=results/v1_peak bench/run.sh
# Workload = production ratios from SIS#1 (per 1048 txns: 300 prefs, 70 inbox, 25 inbox_older, 70 unread, 7 mark_read, 300 insert, 300 status, 1 fanout)
# RATE=0 -> closed loop (max throughput). RATE>0 -> open loop, latency includes queueing (schedule lag).
set -euo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
: "${VER:?}" "${OUT:?}"; PORT=${PORT:-5432}; RATE=${RATE:-0}; CLIENTS=${CLIENTS:-64}; DUR=${DUR:-60}; WARM=${WARM:-20}; MIX=${MIX:-all}
DB=${DB:-notif}; HOST=${HOST:-127.0.0.1}
BASE=$(date -d "$(grep T0= results/anchor.txt | cut -d= -f2)" +%s)
if [ "$VER" = v0 ]; then CH=push; ST0=accepted; ST2=delivered; else CH=1; ST0=1; ST2=3; fi
S=bench/scripts
case "$MIX" in
  all)   FILES="-f $S/prefs_lookup.sql@300 -f $S/inbox_page.sql@70 -f $S/inbox_older.sql@25 -f $S/unread_count.sql@70 -f $S/mark_read.sql@7 -f $S/insert.sql@300 -f $S/status_update.sql@300 -f $S/fanout.sql@1"
         NAMES="prefs_lookup inbox_page inbox_older unread_count mark_read insert status_update fanout";;
  read)  FILES="-f $S/prefs_lookup.sql@300 -f $S/inbox_page.sql@70 -f $S/inbox_older.sql@25 -f $S/unread_count.sql@70 -f $S/fanout.sql@1"
         NAMES="prefs_lookup inbox_page inbox_older unread_count fanout";;
  one)   FILES="-f $S/$SCRIPT.sql@1"; NAMES="$SCRIPT";;
  write) FILES="-f $S/mark_read.sql@7 -f $S/insert.sql@300 -f $S/status_update.sql@300"
         NAMES="mark_read insert status_update";;
esac
COMMON="${EXTRA:-} -h $HOST -p $PORT -U postgres -M prepared -D CH=$CH -D ST0=$ST0 -D ST2=$ST2 -D base=$BASE -c $CLIENTS -j 4"
RATEARG=""; [ "$RATE" != 0 ] && RATEARG="-R $RATE"
rm -rf "$OUT"; mkdir -p "$OUT"
PSQLQ="psql -h 127.0.0.1 -p ${STATPORT:-5432} -U postgres -d $DB -Atq"
snap() { $PSQLQ -c "select pg_current_wal_lsn(), coalesce(sum(n_tup_upd),0), coalesce(sum(n_tup_hot_upd),0), coalesce(sum(n_tup_ins),0) from pg_stat_user_tables where relname like 'notifications%'"; }
# hygiene: flush dirty pages of the previous run so a checkpoint storm is not billed to this one
psql -h 127.0.0.1 -p 5432 -U postgres -d $DB -qc "CHECKPOINT" >/dev/null 2>&1 || true; sleep 3
[ "$WARM" != 0 ] && taskset -c ${LOADCPU:-1} pgbench $COMMON $RATEARG -T "$WARM" $FILES $DB >/dev/null 2>&1 || true
S0=$(snap)
taskset -c ${LOADCPU:-1} pgbench $COMMON $RATEARG -T "$DUR" -l --log-prefix="$OUT/log" -r $FILES $DB > "$OUT/pgbench.out" 2>&1 || true
S1=$(snap)
$PSQLQ -c "select round(pg_wal_lsn_diff('$(echo $S1|cut -d'|' -f1)','$(echo $S0|cut -d'|' -f1)')/1024/1024,1) as wal_mb, \
  $(echo $S1|cut -d'|' -f2) - $(echo $S0|cut -d'|' -f2) as upd, $(echo $S1|cut -d'|' -f3) - $(echo $S0|cut -d'|' -f3) as hot_upd, $(echo $S1|cut -d'|' -f4) - $(echo $S0|cut -d'|' -f4) as ins" > "$OUT/dbstats.txt"
echo "$NAMES" > "$OUT/names.txt"
echo "{\"ver\":\"$VER\",\"port\":$PORT,\"rate\":$RATE,\"clients\":$CLIENTS,\"dur\":$DUR,\"mix\":\"$MIX\"}" > "$OUT/meta.json"
python3 bench/analyze.py "$OUT"
