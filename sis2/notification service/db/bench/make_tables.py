#!/usr/bin/env python3
"""Builds results/TABLES.md and charts/*.png from raw results (no hand-typed numbers)."""
import json, re, os
import matplotlib; matplotlib.use("Agg")
import matplotlib.pyplot as plt
R = "results"; V = ["v0", "v1", "v2", "v2w"]
LAB = {"v0": "v0 baseline", "v1": "v1 types+indexes", "v2": "v2 daily partitions", "v2w": "v2w weekly partitions"}
COL = {"v0": "#257de0", "v1": "#f47230", "v2": "#00b894", "v2w": "#f2a600"}
Q = ["prefs_lookup", "inbox_page", "inbox_older", "unread_count", "mark_read", "insert", "status_update", "fanout"]
def S(d):
    p = f"{R}/{d}/summary.json"
    return json.load(open(p)) if os.path.exists(p) else None
def one(v, q, kind="one"):
    s = S(f"{v}_{kind}_{q}"); return s["scripts"][q] if s else None
out = []
def table(title, head, rows):
    out.append(f"\n### {title}\n"); out.append("| " + " | ".join(head) + " |"); out.append("|" + "---|" * len(head))
    for r in rows: out.append("| " + " | ".join(str(x) for x in r) + " |")
f = lambda x, k: "–" if x is None else x[k]
# A warm isolated
rows = []
for q in Q:
    r = [q]
    for v in V:
        s = one(v, q); r.append("–" if not s else f"{s['p99']} ms / {s['tps']:.0f}")
    rows.append(r)
table("A. Isolated query, warm cache, 8 clients: p99 latency / max throughput (TPS)", ["query"] + [LAB[v] for v in V], rows)
rows = []
for q in ["prefs_lookup", "inbox_page", "inbox_older", "unread_count"]:
    r = [q]
    for v in V:
        s = one(v, q, "cold"); r.append("–" if not s else f"{s['p99']} ms / {s['tps']:.0f}")
    rows.append(r)
table("B. Isolated query, COLD cache (restart + drop_caches), 8 clients: p99 / TPS", ["query"] + [LAB[v] for v in V], rows)
# C mix
rows = []
for v in V:
    lo, mx, ca = S(f"{v}_mix_low"), S(f"{v}_mix_max"), S(f"{v}_mix_cal")
    rows.append([LAB[v], f"{lo['all']['p95']} / {lo['all']['p99']}", f"{mx['all']['tps']:.0f}", f"{mx['all']['p95']} / {mx['all']['p99']}",
                 "n/a (cannot sustain)" if v == "v0" else (f"{ca['all']['p95']} / {ca['all']['p99']}")])
table("C. Production mix (all 8 queries, SIS#1 ratios)", ["version", "@100 TPS p95 / p99 ms", "max TPS (16 clients)", "at max: p95 / p99 ms", "@2000 TPS p95 / p99 ms"], rows)
# D explain
rows = []
def parse(v):
    t = open(f"explain/{v}.txt").read(); res = {}
    for sec in re.split(r"=+ (\w+) \(\w+\) =+", t)[1:][::1]:
        pass
    parts = re.split(r"=+ (\S+) \(\w+\) =+\n", t)
    for i in range(1, len(parts), 2):
        name, body = parts[i], parts[i+1]
        ex = re.search(r"Execution Time: ([\d.]+)", body).group(1)
        b = re.search(r"Buffers: shared ([^\n]*)", body); hit = 0
        if b:
            for k, n in re.findall(r"(hit|read)=(\d+)", b.group(1)): hit += int(n)
        res[name] = (ex, hit)
    return res
E = {v: parse(v) for v in ["v0", "v1", "v2w"]}
for q in E["v0"]:
    rows.append([q] + [f"{E[v][q][0]} ms / {E[v][q][1]} buf" for v in ["v0", "v1", "v2w"]])
table("D. EXPLAIN (ANALYZE, BUFFERS), warm: execution time / 8 KB buffers touched", ["query", "v0", "v1", "v2w"], rows)
# E retention
rd, rp = json.load(open(f"{R}/retention_delete.json")), json.load(open(f"{R}/retention_drop.json"))
table("E. Retention of one day (864k rows)", ["method", "time", "WAL", "dead tuples", "VACUUM after"],
      [["DELETE (v1, unpartitioned)", f"{rd['delete_s']:.2f} s", f"{rd['wal_mb']} MB", f"{rd['dead_tuples_after']:,}", f"{rd['vacuum_s']:.0f} s (size {rd['size_mb_before']}→{rd['size_mb_after_vacuum']} MB: not returned to OS)"],
       ["DROP PARTITION (v2)", f"{rp['drop_s']:.3f} s", f"{rp['wal_mb']} MB", "0", "not needed"]])
# F pooling
rows = []
for k, lab in [("direct_c400_max", "400 clients → PostgreSQL directly, closed loop"), ("pgb_c400_max", "400 clients → PgBouncer (pool 8), closed loop"),
               ("direct_c400_rate", "400 clients → PostgreSQL, 2000 TPS open loop"), ("pgb_c400_rate", "400 clients → PgBouncer, 2000 TPS open loop"),
               ("direct_reconnect", "reconnect per transaction → PostgreSQL"), ("pgb_reconnect", "reconnect per transaction → PgBouncer")]:
    s = S(f"pool_{k}"); a = s["all"]; rows.append([lab, f"{a['tps']:.0f}", a["p50"], a["p95"], a["p99"]])
table("F. Connection pooling (v2w, 1-core DB node)", ["scenario", "TPS", "p50 ms", "p95 ms", "p99 ms"], rows)
# G replica
rows = []
for k, lab in [("single_node", "reads + writes on primary"), ("with_replica", "reads → replica, writes → primary")]:
    r_, w_ = S(f"replica_{k}_read")["all"], S(f"replica_{k}_write")["all"]
    rows.append([lab, f"{r_['tps']:.0f}", f"{r_['p95']} / {r_['p99']}", f"{w_['tps']:.0f}", f"{w_['p95']} / {w_['p99']}"])
table("G. Read replica, same total load 3500 TPS (reads 1520 + writes 1980)", ["setup", "read TPS", "read p95 / p99 ms", "write TPS", "write p95 / p99 ms"], rows)
# H HOT / WAL
rows = []
for v in V:
    d = open(f"{R}/{v}_one_status_update/dbstats.txt").read().strip().split("|"); i = open(f"{R}/{v}_one_insert/dbstats.txt").read().strip().split("|")
    rows.append([LAB[v], f"{int(d[2])/max(1,int(d[1]))*100:.0f}%", f"{float(d[0])/max(1,int(d[1]))*1024:.2f} KB", f"{float(i[0])/max(1,int(i[3]))*1024:.2f} KB"])
table("H. Write amplification", ["version", "HOT share of status UPDATEs", "WAL per status UPDATE", "WAL per INSERT"], rows)
# watermark
wm = [("mark_read (per-row is_read)", one("v2", "mark_read")), ("mark_read (watermark upsert)", S("v2_one_mark_read_wm")["scripts"]["mark_read_wm"]),
      ("unread_count (partial index)", one("v2", "unread_count")), ("unread_count (watermark)", S("v2_one_unread_count_wm")["scripts"]["unread_count_wm"])]
table("I. Read-state design (v2 daily partitions, warm)", ["variant", "p95 ms", "p99 ms", "TPS"], [[n, s["p95"], s["p99"], f"{s['tps']:.0f}"] for n, s in wm])
open(f"{R}/TABLES.md", "w").write("\n".join(out) + "\n"); print("\n".join(out))
# ---- charts ----
plt.rcParams.update({"font.size": 10, "axes.spines.top": False, "axes.spines.right": False})
fig, ax = plt.subplots(figsize=(11, 4.6)); w = 0.2
for j, v in enumerate(V):
    vals = [(one(v, q) or {"p99": float("nan")})["p99"] for q in Q]
    bars = ax.bar([i + (j - 1.5) * w for i in range(len(Q))], vals, w * 0.9, color=COL[v], label=LAB[v])
    for b, x in zip(bars, vals):
        if x == x and (v in ("v0", "v2w")): ax.text(b.get_x() + b.get_width()/2, x * 1.12, f"{x:.3g}", ha="center", va="bottom", fontsize=7, color="#191919")
ax.set_yscale("log"); ax.set_xticks(range(len(Q))); ax.set_xticklabels(Q, rotation=20); ax.set_ylabel("p99 latency, ms (log scale, lower is better)")
ax.set_title("Per-query p99, warm cache, 8 clients", loc="left"); ax.legend(frameon=False, ncol=4, loc="upper center", bbox_to_anchor=(0.5, -0.22)); ax.grid(axis="y", color="#e5e5e5", lw=0.6); ax.set_axisbelow(True)
fig.tight_layout(); fig.savefig("charts/per_query_p99.png", dpi=150); plt.close(fig)
fig, ax = plt.subplots(figsize=(7.5, 3.8))
vals = [S(f"{v}_mix_max")["all"]["tps"] for v in V]
bars = ax.bar([LAB[v] for v in V], vals, color=[COL[v] for v in V], width=0.6)
for b, v, x in zip(bars, V, vals):
    ax.text(b.get_x() + b.get_width()/2, x + 60, f"{x:.0f} TPS\np99 {S(v+'_mix_max')['all']['p99']} ms", ha="center", fontsize=9, color="#191919")
ax.set_ylim(0, max(vals) * 1.25); ax.set_ylabel("max TPS of the production mix"); ax.set_title("Capacity of one PostgreSQL core, same mix", loc="left")
plt.xticks(rotation=12); ax.grid(axis="y", color="#e5e5e5", lw=0.6); ax.set_axisbelow(True); fig.tight_layout(); fig.savefig("charts/mix_capacity.png", dpi=150)
