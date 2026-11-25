
### A. Isolated query, warm cache, 8 clients: p99 latency / max throughput (TPS)

| query | v0 baseline | v1 types+indexes | v2 daily partitions | v2w weekly partitions |
|---|---|---|---|---|
| prefs_lookup | 270.392 ms / 52 | 1.118 ms / 14651 | 1.066 ms / 14376 | 0.984 ms / 15198 |
| inbox_page | 17.611 ms / 1414 | 2.051 ms / 10743 | 4.429 ms / 5156 | 2.256 ms / 10036 |
| inbox_older | 12.581 ms / 1744 | 5.468 ms / 4259 | 8.514 ms / 2636 | 5.448 ms / 3521 |
| unread_count | 11.587 ms / 1734 | 0.719 ms / 26654 | 5.543 ms / 4218 | 1.313 ms / 14104 |
| mark_read | 77.866 ms / 184 | 58.978 ms / 252 | 95.512 ms / 151 | 67.924 ms / 189 |
| insert | 1.473 ms / 15205 | 6.374 ms / 2629 | 1.315 ms / 14148 | 1.352 ms / 12375 |
| status_update | 1.431 ms / 14024 | 1.433 ms / 14248 | 2.957 ms / 6844 | 2.849 ms / 7181 |
| fanout | 2790.61 ms / 8 | 236.933 ms / 78 | 232.899 ms / 76 | 211.239 ms / 80 |

### B. Isolated query, COLD cache (restart + drop_caches), 8 clients: p99 / TPS

| query | v0 baseline | v1 types+indexes | v2 daily partitions | v2w weekly partitions |
|---|---|---|---|---|
| prefs_lookup | 270.467 ms / 48 | 1.113 ms / 15547 | 1.172 ms / 14065 | 0.96 ms / 16152 |
| inbox_page | 57.263 ms / 338 | 10.588 ms / 5744 | 13.525 ms / 3316 | 9.783 ms / 5717 |
| inbox_older | 33.22 ms / 738 | 12.517 ms / 1498 | 13.854 ms / 1170 | 9.675 ms / 1919 |
| unread_count | 28.804 ms / 965 | 3.694 ms / 14557 | 13.438 ms / 2987 | 4.904 ms / 8228 |

### C. Production mix (all 8 queries, SIS#1 ratios)

| version | @100 TPS p95 / p99 ms | max TPS (16 clients) | at max: p95 / p99 ms | @2000 TPS p95 / p99 ms |
|---|---|---|---|---|
| v0 baseline | 218.721 / 310.919 | 156 | 321.483 / 364.809 | n/a (cannot sustain) |
| v1 types+indexes | 1.894 / 3.659 | 4016 | 8.982 / 14.79 | 4.181 / 17.184 |
| v2 daily partitions | 2.916 / 6.952 | 3459 | 9.737 / 17.599 | 579.792 / 875.577 |
| v2w weekly partitions | 1.929 / 4.203 | 4263 | 7.784 / 13.737 | 3.752 / 38.022 |

### D. EXPLAIN (ANALYZE, BUFFERS), warm: execution time / 8 KB buffers touched

| query | v0 | v1 | v2w |
|---|---|---|---|
| prefs_lookup_1 | 0.053 ms / 7 buf | 0.069 ms / 7 buf | 0.060 ms / 7 buf |
| prefs_lookup_2_devices | 23.759 ms / 6350 buf | 0.085 ms / 7 buf | 0.051 ms / 7 buf |
| inbox_page | 0.292 ms / 125 buf | 0.166 ms / 27 buf | 0.267 ms / 33 buf |
| inbox_older | 0.270 ms / 125 buf | 0.214 ms / 27 buf | 0.187 ms / 26 buf |
| unread_count | 0.255 ms / 119 buf | 0.130 ms / 14 buf | 0.307 ms / 47 buf |
| mark_read | 0.434 ms / 323 buf | 0.578 ms / 238 buf | 0.750 ms / 260 buf |
| insert | 0.437 ms / 67 buf | 0.386 ms / 93 buf | 0.399 ms / 107 buf |
| status_update | 0.141 ms / 21 buf | 0.149 ms / 9 buf | 0.178 ms / 17 buf |
| fanout | 152.748 ms / 7859 buf | 27.915 ms / 2698 buf | 27.032 ms / 2698 buf |

### E. Retention of one day (864k rows)

| method | time | WAL | dead tuples | VACUUM after |
|---|---|---|---|---|
| DELETE (v1, unpartitioned) | 6.15 s | 97.0 MB | 1,016,362 | 78 s (size 5560→5560 MB: not returned to OS) |
| DROP PARTITION (v2) | 0.115 s | 0.087 MB | 0 | not needed |

### F. Connection pooling (v2w, 1-core DB node)

| scenario | TPS | p50 ms | p95 ms | p99 ms |
|---|---|---|---|---|
| 400 clients → PostgreSQL directly, closed loop | 2652 | 112.551 | 308.605 | 1160.166 |
| 400 clients → PgBouncer (pool 8), closed loop | 4485 | 70.37 | 160.3 | 203.88 |
| 400 clients → PostgreSQL, 2000 TPS open loop | 1950 | 2.677 | 1924.68 | 2528.054 |
| 400 clients → PgBouncer, 2000 TPS open loop | 2000 | 0.562 | 2.677 | 8.704 |
| reconnect per transaction → PostgreSQL | 211 | 91.737 | 180.472 | 215.869 |
| reconnect per transaction → PgBouncer | 3907 | 5.009 | 11.732 | 19.277 |

### G. Read replica, same total load 3500 TPS (reads 1520 + writes 1980)

| setup | read TPS | read p95 / p99 ms | write TPS | write p95 / p99 ms |
|---|---|---|---|---|
| reads + writes on primary | 1507 | 9.87 / 65.098 | 1979 | 839.927 / 1007.121 |
| reads → replica, writes → primary | 1516 | 5.136 / 19.485 | 1978 | 11.034 / 36.677 |

### H. Write amplification

| version | HOT share of status UPDATEs | WAL per status UPDATE | WAL per INSERT |
|---|---|---|---|
| v0 baseline | 98% | 0.20 KB | 0.43 KB |
| v1 types+indexes | 100% | 0.19 KB | 3.61 KB |
| v2 daily partitions | 100% | 0.29 KB | 0.49 KB |
| v2w weekly partitions | 100% | 0.53 KB | 0.53 KB |

### I. Read-state design (v2 daily partitions, warm)

| variant | p95 ms | p99 ms | TPS |
|---|---|---|---|
| mark_read (per-row is_read) | 78.935 | 95.512 | 151 |
| mark_read (watermark upsert) | 0.888 | 1.141 | 13183 |
| unread_count (partial index) | 4.0 | 5.543 | 4218 |
| unread_count (watermark) | 2.808 | 4.554 | 5904 |
