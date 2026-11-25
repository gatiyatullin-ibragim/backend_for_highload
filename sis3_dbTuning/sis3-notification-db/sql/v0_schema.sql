-- v0: "first draft" schema -- what a student writes before measuring anything.
-- text enums, PK-only on devices (no user_id index), single user_id index on the big table.
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
CREATE SEQUENCE notif_id_seq START 25056001;
CREATE TABLE notifications (
  id          bigint PRIMARY KEY,
  user_id     bigint NOT NULL,
  created_at  timestamptz NOT NULL,
  channel     text NOT NULL,
  title       text NOT NULL,
  body        text NOT NULL,
  status      text NOT NULL,
  is_read     boolean NOT NULL DEFAULT false
);
