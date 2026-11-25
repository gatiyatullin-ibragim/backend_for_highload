-- unread badge (in SIS1 this lives in Redis; here the PG fallback when Redis is cold)
\set uid random(1, 200000)
SELECT count(*) FROM notifications WHERE user_id = :uid AND NOT is_read;
