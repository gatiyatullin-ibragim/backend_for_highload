-- SIS1: worker needs prefs + push tokens on cache miss (worst case: Redis down -> 3000 QPS hit PG)
\set uid random(1, 200000)
SELECT user_id, push_on, email_on, sms_on, quiet_from, quiet_to, tz, locale FROM user_preferences WHERE user_id = :uid;
SELECT platform, token FROM devices WHERE user_id = :uid;
