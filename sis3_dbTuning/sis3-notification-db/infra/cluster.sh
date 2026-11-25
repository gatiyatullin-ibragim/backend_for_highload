#!/usr/bin/env bash
# Usage: infra/cluster.sh init|start|stop|replica-init|replica-start|replica-stop|pgb-start|pgb-stop
# CPU layout (simulates separate hosts on one 16-core box):
#   2-vCPU sandbox: primary = core 0, replica = core 1, PgBouncer + load generator = core 1 (see REPORT: hardware limits)
set -euo pipefail
BIN=/usr/lib/postgresql/16/bin
ROOT=/var/lib/sis3
PRIMARY=$ROOT/primary
REPLICA=$ROOT/replica
HERE="$(cd "$(dirname "$0")" && pwd)"
as_pg() { sudo -u postgres "$@"; }
case "${1:-}" in
  init)
    rm -rf "$PRIMARY"; mkdir -p "$ROOT"; chown postgres:postgres "$ROOT"
    as_pg $BIN/initdb -D "$PRIMARY" --data-checksums >/dev/null
    cat "$HERE/postgresql.sis3.conf" >> "$PRIMARY/postgresql.conf"
    echo "port = 5432" >> "$PRIMARY/postgresql.conf"
    echo "host replication all 127.0.0.1/32 trust" >> "$PRIMARY/pg_hba.conf"
    echo "host all all 127.0.0.1/32 trust" >> "$PRIMARY/pg_hba.conf"
    ;;
  start)  as_pg taskset -c 0 $BIN/pg_ctl -D "$PRIMARY" -l "$ROOT/primary.log" -w start ;;
  stop)   as_pg $BIN/pg_ctl -D "$PRIMARY" -m fast -w stop ;;
  replica-init)
    rm -rf "$REPLICA"
    as_pg $BIN/pg_basebackup -h 127.0.0.1 -p 5432 -D "$REPLICA" -R -X stream -c fast
    echo "port = 5433" >> "$REPLICA/postgresql.conf"
    ;;
  replica-start) as_pg taskset -c 1 $BIN/pg_ctl -D "$REPLICA" -l "$ROOT/replica.log" -w start ;;
  replica-stop)  as_pg $BIN/pg_ctl -D "$REPLICA" -m fast -w stop ;;
  pgb-start)
    mkdir -p /var/run/sis3 && chown postgres:postgres /var/run/sis3
    as_pg taskset -c 1 pgbouncer -d "$HERE/pgbouncer.ini" ;;
  pgb-stop) pkill -u postgres pgbouncer || true ;;
  *) echo "usage: $0 init|start|stop|replica-init|replica-start|replica-stop|pgb-start|pgb-stop"; exit 1 ;;
esac
