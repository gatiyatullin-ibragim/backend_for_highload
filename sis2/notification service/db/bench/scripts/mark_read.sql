\set uid random(1, 200000)
UPDATE notifications SET is_read = true WHERE user_id = :uid AND NOT is_read;
