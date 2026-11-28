#!/usr/bin/env python3
"""Percentiles from pgbench -l logs. Log columns: client txn_no time_us script_no epoch epoch_us [schedule_lag_us]
With -R the *response* latency = time + schedule_lag (what a caller of an open-loop system actually sees)."""
import sys, glob, json, os
import numpy as np
d = sys.argv[1]
meta = json.load(open(f"{d}/meta.json")); names = open(f"{d}/names.txt").read().split()
rows = []
for f in glob.glob(f"{d}/log*"):
    for line in open(f):
        p = line.split()
        if len(p) < 6: continue
        t = int(p[2]); s = int(p[3]); lag = int(p[6]) if len(p) > 6 else 0
        if p[2] == "failed": continue
        rows.append((s, t + lag, t, lag))
a = np.array(rows, dtype=np.int64)
out = {"meta": meta, "scripts": {}}
def stats(x_ms, dur):
    return {"n": int(len(x_ms)), "tps": round(len(x_ms)/dur, 1), "avg": round(float(x_ms.mean()), 3),
            "p50": round(float(np.percentile(x_ms, 50)), 3), "p95": round(float(np.percentile(x_ms, 95)), 3),
            "p99": round(float(np.percentile(x_ms, 99)), 3), "max": round(float(x_ms.max()), 1)}
dur = meta["dur"]
out["all"] = stats(a[:, 1]/1000.0, dur)
out["all_service"] = stats(a[:, 2]/1000.0, dur)
for i, n in enumerate(names):
    m = a[a[:, 0] == i]
    if len(m): out["scripts"][n] = stats(m[:, 1]/1000.0, dur)
tgt = meta["rate"]
out["sustained"] = (tgt == 0) or (out["all"]["tps"] >= 0.97 * tgt and out["all"]["p99"] < 1000)
json.dump(out, open(f"{d}/summary.json", "w"), indent=1)
print(f"[{os.path.basename(d.rstrip('/'))}] tps={out['all']['tps']} p50={out['all']['p50']} p95={out['all']['p95']} p99={out['all']['p99']} ms  sustained={out['sustained']}")
for n, s in out["scripts"].items():
    print(f"   {n:14s} tps={s['tps']:8.1f} p50={s['p50']:8.2f} p95={s['p95']:8.2f} p99={s['p99']:8.2f} max={s['max']:8.1f}")
