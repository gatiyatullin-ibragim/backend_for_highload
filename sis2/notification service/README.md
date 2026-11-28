#SIS #2 : Notification service(Protocol implemetnation)
Interservice protocol and base logic implementation

##Architecture choice: gRPC or REST/WebSocket

In this task in case of interservice(Service-to-Service) communication i chose gRPC contract + Protobuf + HTTP/2

Why i chose gRPC?
1.We get the requests from ither services like "Orders" not from browser. And gRPC it's best for interservice communication
2.Multiplexin. one HTTP/2 connection allows transmit parallerly several requests even on peak load 3000RPC
3.Protobuf. Json is more heavier than binary format of serealization in protobuf.
4.Strict deadline support(timeouts)on context level and configuration of retries.
5.Why not WebSocket? WebSocket is great for delivering client/browser data initially, but it's not designed for the classic request-response model between microservices.
Architectural tradeoff: The internal circuit between microservices is implemented using gRPC (high-speed radio and 3,000 RPS), while the external public API for clients remains REST/JSON.


how to run project?
go run server/main.go

## Load test: 3000 RPS

The load test uses a real TCP gRPC connection to the running notification
service. It sends 3000 requests per second with unique idempotency keys and
reports successful requests, achieved RPS, and p95 latency.

From this directory, in two terminals:

```bash
go run server/main.go
RUN_3000_RPS=1 LOAD_TEST_DURATION=10 \
  go test ./client -run '^TestGRPC3000RPS$' -count=1 -v
```

`LOAD_TEST_DURATION` is specified in seconds and defaults to 10. Set
`NOTIFICATION_GRPC_ADDR` when the service listens on a different address.
Regular `go test ./client` skips this load test so unit tests do not require a
running service.

## SIS #3 extension: PostgreSQL storage

SIS 3 is part of this service, not a separate project:

```
notification service/
├── notification.proto, pb/      SIS 2: gRPC contract
├── server/, client/             SIS 2: gRPC server, client, 3000 RPS load test
├── store/                       SIS 3: Store interface, memory (SIS 2) and postgres implementations
└── db/                          SIS 3: database tuning
    ├── REPORT.md                report with before/after measurements (p95/p99)
    ├── sql/                     schemas v0-v2w, indexes, SIS 2 extension (idempotency)
    ├── bench/, infra/           reproducible benchmark, PostgreSQL/PgBouncer/replica setup
    ├── explain/, results/, charts/   EXPLAIN plans, raw results, charts
    └── DEFENSE_NOTES_RU.md      notes for the defense
```


The in-memory maps from SIS 2 now sit behind a `store.Store` interface (`store/`).
`STORE=memory` (default) keeps the SIS 2 behaviour; `STORE=postgres` uses the database tuned in
[`db/`](db/) (schema v2w: weekly partitions, covering/composite indexes).

How the SIS 3 decisions show up in the code (`store/postgres.go`):

| SIS 3 decision | Where |
|---|---|
| PgBouncer, transaction pooling | `DATABASE_URL` points to `:6432`; pgx runs in `QueryExecModeCacheDescribe` (no named prepared statements) |
| Read replica | `DATABASE_RO_URL` (`notif_ro`) is used by `GetStatus` |
| Read-your-writes | statuses younger than 3 s are read from the primary (replica lag measured up to 124 ms) |
| Partition pruning | `notification_id = "<id>-<unix seconds>"`, the lookup filters `created_at` to that second |
| Idempotency | table `idempotency_keys` (`db/sql/sis2_extension.sql`), one short transaction with the inbox insert |

Run:

```bash
# 1. database (fresh one)
createdb notif
psql notif -f db/sql/v2_schema.sql \
           -f db/sql/v2_indexes.sql \
           -f db/sql/sis2_extension.sql
# 2. dependencies (first time only)
go get github.com/jackc/pgx/v5@latest && go mod tidy
# 3. service
STORE=postgres DATABASE_URL='postgres://postgres@127.0.0.1:5432/notif?sslmode=disable' go run server/main.go
```

Without PgBouncer/replica point `DATABASE_URL` to Postgres directly and leave `DATABASE_RO_URL` empty.
The 3000 RPS load test from the previous section works unchanged against either store.
