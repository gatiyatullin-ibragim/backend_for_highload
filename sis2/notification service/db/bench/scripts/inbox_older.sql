-- keyset pagination: "older than cursor" (cursor = timestamp of last item on previous page)
\set uid random(1, 200000)
\set days random(1, 25)
SELECT id, created_at, channel, title, is_read FROM notifications
 WHERE user_id = :uid AND created_at < now() - :days * interval '1 day' ORDER BY created_at DESC, id DESC LIMIT 20;
