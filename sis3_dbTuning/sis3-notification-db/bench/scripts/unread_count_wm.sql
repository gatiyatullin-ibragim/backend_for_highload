\set uid random(1, 200000)
SELECT count(*) FROM notifications n WHERE n.user_id = :uid
  AND n.created_at > coalesce((SELECT last_read_at FROM user_read_state WHERE user_id = :uid), '-infinity');
