-- Расширение схемы SIS 3 под сервис из SIS 2 (идемпотентность SendNotification).
-- Применять ПОСЛЕ v2_schema.sql + v2_indexes.sql, на свежей базе:
--   createdb notif
--   psql notif -f sql/v2_schema.sql -f sql/v2_indexes.sql -f sql/sis2_extension.sql

-- Ключ идемпотентности -> id уведомления. PK даёт быстрый поиск и защиту от параллельных дублей.
-- Ключи нужны ограниченное время (например 24 ч): чистить cron-ом, а при большом объёме партиционировать так же, как notifications.
CREATE TABLE IF NOT EXISTS idempotency_keys (
  key             text PRIMARY KEY,
  notification_id bigint NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now()
);

-- Недельные партиции (как v2w) от прошлой недели на 4 недели вперёд.
-- Без партиции под now() INSERT в notifications падает; в проде это делает задание по расписанию.
SELECT make_partitions((date_trunc('week', now()))::date - 7, (date_trunc('week', now()))::date + 28, 7);
