-- "read watermark": mark-all-read = ONE row upsert instead of updating ~19 notification rows
\set uid random(1, 200000)
INSERT INTO user_read_state (user_id, last_read_at) VALUES (:uid, now()) ON CONFLICT (user_id) DO UPDATE SET last_read_at = now();
