-- :lo / :hi = id range, {{START}} = timestamp of id 0, 10 notifications per second (= 864k/day = 1:100 of 86M/day)
SET synchronous_commit = off;
INSERT INTO notifications (id, user_id, created_at, channel, title, body, status, is_read)
SELECT g,
  (hashint8(g) & 2147483647) % 200000 + 1,
  timestamptz '{{START}}' + g * interval '100 milliseconds',
  {{CHANNEL}},
  'Notification #' || g,
  md5(g::text) || md5((g + 1)::text),
  {{STATUS}},
  (hashint8(g + 7) & 2147483647) % 100 < 85
FROM generate_series(:lo, :hi) g;
