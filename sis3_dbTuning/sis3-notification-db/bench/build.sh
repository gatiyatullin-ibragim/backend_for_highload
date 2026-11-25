#!/usr/bin/env bash
# Usage: bench/build.sh v0|v1|v2     (re)builds database "notif" with the given schema version + identical data
set -euo pipefail
V=${1:?v0|v1|v2}
HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
PSQL="sudo -u postgres psql -h 127.0.0.1 -p 5432 -d notif -v ON_ERROR_STOP=1 -q"
N=25056000; CHUNK=2500000
T0=$($PSQL -Atc "select date_trunc('day', now()) - interval '29 days'")   # timestamp of id 0
echo "T0=$T0" | tee results/anchor.txt
[ "$V" = v0 ] && ENVF=sql/expr_text.env || ENVF=sql/expr_smallint.env
SQLV=$V; [ "$V" = v2w ] && SQLV=v2     # v2w = same DDL as v2, weekly partitions
source "$ENVF"
t0=$(date +%s)
$PSQL -f sql/${SQLV}_schema.sql
[ "$V" = v2 ] && $PSQL -c "select make_partitions((date '$T0')::date, (date_trunc('day', now()) + interval '4 days')::date)"
[ "$V" = v2w ] && $PSQL -c "select make_partitions((date '$T0')::date, (date_trunc('day', now()) + interval '14 days')::date, 7)"
$PSQL -f sql/load_small_tables.sql
python3 - "$ENVF" > /tmp/load_notifications.sql <<PY
import sys
tpl=open("sql/load_notifications.sql.tpl").read()
env={}
for l in open(sys.argv[1]):
    k,v=l.rstrip().split("=",1); env[k]=v.strip('"')
print(tpl.replace("{{START}}","$T0").replace("{{CHANNEL}}",env["CHANNEL"]).replace("{{STATUS}}",env["STATUS"]))
PY
lo=1
while [ $lo -le $N ]; do
  hi=$((lo+CHUNK-1)); [ $hi -gt $N ] && hi=$N
  $PSQL -v lo=$lo -v hi=$hi -f /tmp/load_notifications.sql; echo "  loaded ids $lo..$hi"
  lo=$((hi+1))
done
$PSQL -f sql/${SQLV}_indexes.sql
for t in notifications devices user_preferences; do $PSQL -c "VACUUM (ANALYZE) $t"; done
$PSQL -c "CHECKPOINT"
$PSQL -Atc "select 'rows', count(*) from notifications"
$PSQL -Atc "select 'table+indexes size', pg_size_pretty(pg_total_relation_size('notifications'))"
echo "build $V done in $(( $(date +%s) - t0 )) s"
