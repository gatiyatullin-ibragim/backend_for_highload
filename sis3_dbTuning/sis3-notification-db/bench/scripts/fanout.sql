-- campaign fan-out: next batch of 1000 push recipients (keyset on user_id)
\set last random(1, 199000)
SELECT p.user_id, d.token FROM user_preferences p JOIN devices d ON d.user_id = p.user_id
 WHERE p.push_on AND p.user_id > :last ORDER BY p.user_id LIMIT 1000;
