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
