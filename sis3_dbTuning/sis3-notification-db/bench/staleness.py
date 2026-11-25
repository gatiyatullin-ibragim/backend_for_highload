#!/usr/bin/env python3
"""Replica staleness probe: commit on primary -> how long until the row is visible on the replica (read-your-writes gap)."""
import sys, time, json, psycopg2, numpy as np
N = int(sys.argv[1]) if len(sys.argv) > 1 else 300
p = psycopg2.connect(host="127.0.0.1", port=5432, user="postgres", dbname="notif"); p.autocommit = True
r = psycopg2.connect(host="127.0.0.1", port=5433, user="postgres", dbname="notif"); r.autocommit = True
pc, rc = p.cursor(), r.cursor(); gaps = []
for _ in range(N):
    pc.execute("INSERT INTO lag_probe DEFAULT VALUES RETURNING id"); i = pc.fetchone()[0]; t = time.perf_counter()
    while True:
        rc.execute("SELECT 1 FROM lag_probe WHERE id = %s", (i,))
        if rc.fetchone(): break
    gaps.append((time.perf_counter() - t) * 1000); time.sleep(0.01)
g = np.array(gaps)
print(json.dumps({"n": N, "p50_ms": round(float(np.percentile(g, 50)), 2), "p95_ms": round(float(np.percentile(g, 95)), 2),
                  "p99_ms": round(float(np.percentile(g, 99)), 2), "max_ms": round(float(g.max()), 2)}))
