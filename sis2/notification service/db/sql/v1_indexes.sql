-- inbox page + keyset pagination: equality on user_id, then already sorted newest-first
CREATE INDEX notifications_inbox_idx ON notifications (user_id, created_at DESC, id DESC);
-- unread counter / mark-all-read: tiny partial index (only ~15% of rows are unread)
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE NOT is_read;
-- push token lookup: covering index -> index-only scan, no heap visit
CREATE INDEX devices_user_idx ON devices (user_id) INCLUDE (platform, token);
