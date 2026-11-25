# SIS #3 — Database tuning (Notification Service)

`REPORT.md` is the short report. This repo reproduces every number in it.

```
 clients ──► PgBouncer :6432 (transaction pool, 8) ──► PostgreSQL primary :5432  ──WAL──► replica :5433 (reads)
 sql/    v0 baseline · v1 types+indexes · v2 daily partitions · v2w weekly partitions (same DDL as v2) + deterministic data generator
 bench/  build.sh · run.sh (pgbench mix) · analyze.py (p95/p99 from per-transaction logs) · suite.sh · pool_suite.sh · replica_suite.sh · extras_v2.sh · make_tables.py
 infra/  postgresql.sis3.conf (identical for all runs) · cluster.sh · pgbouncer.ini
 explain/  EXPLAIN (ANALYZE, BUFFERS) of all 9 statements for v0, v1, v2, v2w
 results/  summary.json per run + pgbench.out + TABLES.md   (raw per-transaction logs are omitted: ~250 MB)
 charts/
```

## Reproduce (Ubuntu 24.04, PostgreSQL 16, pgbouncer, Python 3 with numpy/matplotlib/psycopg2)
```bash
infra/cluster.sh init && infra/cluster.sh start
sudo -u postgres createdb -h 127.0.0.1 notif && sudo -u postgres psql -h 127.0.0.1 -d notif -c "create extension pg_stat_statements"
for v in v0 v1 v2w; do bench/build.sh $v && bench/suite.sh $v 100 2000; done   # ~12 min per version, ~6 GB disk
bench/extras_v2.sh        # on v2 (daily): DROP PARTITION retention + read-watermark   (build v2 first)
bench/retention_delete.sh # on v1: DELETE retention                                     (build v1 first)
bench/pool_suite.sh v2w 2000 && bench/replica_suite.sh v2w 3500    # on v2w
python3 bench/make_tables.py
```
Rules for clean numbers: do **not** edit scripts while a suite runs (bash reads them incrementally); nothing else may use the CPU; the sandbox used 2 vCPU (see REPORT §9). Absolute TPS depend on hardware; compare ratios.
