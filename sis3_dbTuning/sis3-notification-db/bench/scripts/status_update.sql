-- provider webhook: delivery status of a RECENT notification (id encodes creation second => partition pruning)
-- ids of the last loaded day; created_at(id) = base + id/10 seconds
\set id random(24192001, 25056000)
\set ts :base + :id / 10
UPDATE notifications SET status = :ST2 WHERE id = :id AND created_at >= to_timestamp(:ts) AND created_at < to_timestamp(:ts + 1);
