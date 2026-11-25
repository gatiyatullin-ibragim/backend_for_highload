-- v1: same workload, schema + indexes designed from the query list (see REPORT.md, section 2).
DROP TABLE IF EXISTS notifications, devices, user_preferences CASCADE;
DROP SEQUENCE IF EXISTS notif_id_seq;

CREATE TABLE user_preferences (
  user_id     bigint PRIMARY KEY,
  push_on     boolean NOT NULL,
  email_on    boolean NOT NULL,
  sms_on      boolean NOT NULL,
  quiet_from  time,
  quiet_to    time,
  tz          text,
  locale      text,
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE devices (
  device_id   bigint PRIMARY KEY,
  user_id     bigint NOT NULL,
  platform    text NOT NULL,
  token       text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
-- CACHE 100: each backend takes 100 ids at once, nextval() stops being a hot lock
CREATE SEQUENCE notif_id_seq START 25056001 CACHE 100;
-- 8-byte columns first (no alignment padding), smallint enums instead of text,
-- fillfactor 90 leaves room on the page so status UPDATEs can be HOT
CREATE TABLE notifications (
  id          bigint NOT NULL PRIMARY KEY,
  user_id     bigint NOT NULL,
  created_at  timestamptz NOT NULL,
  channel     smallint NOT NULL,   -- 1 push, 2 email, 3 sms, 4 in_app
  status      smallint NOT NULL,   -- 1 accepted, 2 sent, 3 delivered, 4 failed
  is_read     boolean NOT NULL DEFAULT false,
  title       text NOT NULL,
  body        text NOT NULL
) WITH (fillfactor = 90);
