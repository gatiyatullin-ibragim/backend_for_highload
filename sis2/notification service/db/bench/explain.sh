#!/usr/bin/env bash
# Usage: bench/explain.sh v0|v1|v2  -> explain/<ver>.txt   (EXPLAIN (ANALYZE, BUFFERS) of every workload query, warm cache, 2nd execution)
set -euo pipefail
V=$1; HERE="$(cd "$(dirname "$0")/.." && pwd)"; cd "$HERE"
BASE=$(date -d "$(grep T0= results/anchor.txt | cut -d= -f2)" +%s)
[ "$V" = v0 ] && { CH="'push'"; ST0="'accepted'"; ST2="'delivered'"; } || { CH=1; ST0=1; ST2=3; }
ID=25000000; TS=$((BASE + ID/10)); UID_=12345
PSQL="psql -h 127.0.0.1 -p ${PORT:-5432} -U postgres -d notif -q"
q() { # name, sql, rollback?
  echo "=================== $1 ($V) ==================="; echo "$2"; echo "-------------------"
  $PSQL -c "BEGIN" -c "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) $2" -c "ROLLBACK" >/dev/null 2>&1   # warm up
  $PSQL -At -c "BEGIN" -c "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) $2" -c "ROLLBACK" | grep -v '^BEGIN$\|^ROLLBACK$'
  echo
}
{
q prefs_lookup_1 "SELECT user_id, push_on, email_on, sms_on, quiet_from, quiet_to, tz, locale FROM user_preferences WHERE user_id = $UID_;"
q prefs_lookup_2_devices "SELECT platform, token FROM devices WHERE user_id = $UID_;"
q inbox_page "SELECT id, created_at, channel, title, is_read FROM notifications WHERE user_id = $UID_ ORDER BY created_at DESC, id DESC LIMIT 20;"
q inbox_older "SELECT id, created_at, channel, title, is_read FROM notifications WHERE user_id = $UID_ AND created_at < now() - 10 * interval '1 day' ORDER BY created_at DESC, id DESC LIMIT 20;"
q unread_count "SELECT count(*) FROM notifications WHERE user_id = $UID_ AND NOT is_read;"
q mark_read "UPDATE notifications SET is_read = true WHERE user_id = $UID_ AND NOT is_read;"
q insert "INSERT INTO notifications (id, user_id, created_at, channel, title, body, status, is_read) VALUES (nextval('notif_id_seq'), $UID_, now(), $CH, 'Your order has shipped', 'Track your parcel in the app', $ST0, false);"
q status_update "UPDATE notifications SET status = $ST2 WHERE id = $ID AND created_at >= to_timestamp($TS) AND created_at < to_timestamp($TS + 1);"
q fanout "SELECT p.user_id, d.token FROM user_preferences p JOIN devices d ON d.user_id = p.user_id WHERE p.push_on AND p.user_id > 100000 ORDER BY p.user_id LIMIT 1000;"
} > explain/$V.txt 2>&1
grep -E "^=====|Execution Time|Planning Time" explain/$V.txt
