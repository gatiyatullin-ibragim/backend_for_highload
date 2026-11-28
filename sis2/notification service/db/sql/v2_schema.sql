-- v2: v1 + RANGE partitioning of notifications by day (retention = DROP PARTITION).
-- PK must contain the partition key => (id, created_at).
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
CREATE SEQUENCE notif_id_seq START 25056001 CACHE 100;
CREATE TABLE notifications (
  id          bigint NOT NULL,
  user_id     bigint NOT NULL,
  created_at  timestamptz NOT NULL,
  channel     smallint NOT NULL,
  status      smallint NOT NULL,
  is_read     boolean NOT NULL DEFAULT false,
  title       text NOT NULL,
  body        text NOT NULL,
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- partition maintenance function: pre-creates [from_day, to_day) partitions of `step` days (1 = daily, 7 = weekly)
CREATE OR REPLACE FUNCTION make_partitions(from_day date, to_day date, step int DEFAULT 1) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE d date := from_day;
BEGIN
  WHILE d < to_day LOOP
    EXECUTE format(
      'CREATE TABLE IF NOT EXISTS notifications_%s PARTITION OF notifications
         FOR VALUES FROM (%L) TO (%L) WITH (fillfactor = 90)',
      to_char(d, 'YYYYMMDD'), d, d + step);
    d := d + step;
  END LOOP;
END $$;
