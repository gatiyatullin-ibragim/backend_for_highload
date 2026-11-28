SET synchronous_commit = off;
INSERT INTO user_preferences (user_id, push_on, email_on, sms_on, quiet_from, quiet_to, tz, locale)
SELECT g,
  (hashint8(g)   & 2147483647) % 100 < 92,
  (hashint8(g+1) & 2147483647) % 100 < 80,
  (hashint8(g+2) & 2147483647) % 100 < 10,
  time '22:00', time '08:00',
  (ARRAY['Asia/Almaty','Europe/Moscow','Europe/Berlin','America/New_York'])[1 + g % 4],
  (ARRAY['ru','en','kk'])[1 + g % 3]
FROM generate_series(1, 200000) g;
INSERT INTO devices (device_id, user_id, platform, token)
SELECT g, (g - 1) % 200000 + 1,
  CASE WHEN g % 3 = 0 THEN 'ios' ELSE 'android' END,
  md5(g::text) || md5((g * 7)::text)
FROM generate_series(1, 400000) g;
