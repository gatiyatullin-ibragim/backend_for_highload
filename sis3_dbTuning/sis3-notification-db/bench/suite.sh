#!/usr/bin/env bash
# Usage: bench/suite.sh v0|v1|v2 [R_LOW] [R_CAL]   full measurement ladder for one schema version on the primary
set -euo pipefail
V=$1; RLOW=${2:-100}; RCAL=${3:-0}
HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
bench/explain.sh $V
for s in prefs_lookup inbox_page inbox_older unread_count mark_read insert status_update fanout; do
  VER=$V MIX=one SCRIPT=$s CLIENTS=8 DUR=20 WARM=5 OUT=results/${V}_one_$s bench/run.sh | head -1
done
# COLD cache: restart + drop OS page cache -> every first touch of a page is a disk read (production: 2.6 TB data >> RAM)
for s in prefs_lookup inbox_page inbox_older unread_count; do
  infra/cluster.sh stop >/dev/null; sync; echo 3 > /proc/sys/vm/drop_caches; infra/cluster.sh start >/dev/null
  VER=$V MIX=one SCRIPT=$s CLIENTS=8 DUR=20 WARM=0 OUT=results/${V}_cold_$s bench/run.sh | head -1
done
VER=$V RATE=$RLOW CLIENTS=32 DUR=45 WARM=15 OUT=results/${V}_mix_low bench/run.sh
VER=$V RATE=0 CLIENTS=16 DUR=30 WARM=10 OUT=results/${V}_mix_max bench/run.sh
if [ "$RCAL" != 0 ]; then VER=$V RATE=$RCAL CLIENTS=64 DUR=45 WARM=15 OUT=results/${V}_mix_cal bench/run.sh; fi
