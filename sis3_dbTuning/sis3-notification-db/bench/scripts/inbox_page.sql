-- SIS1: GET /users/{id}/inbox?limit=20  (first page, newest first)
\set uid random(1, 200000)
SELECT id, created_at, channel, title, is_read FROM notifications WHERE user_id = :uid ORDER BY created_at DESC, id DESC LIMIT 20;
