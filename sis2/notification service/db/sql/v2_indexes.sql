-- local (per-partition) indexes: each is ~1/30 of the size => hot part stays in RAM
CREATE INDEX notifications_inbox_idx  ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE NOT is_read;
CREATE INDEX devices_user_idx ON devices (user_id) INCLUDE (platform, token);
