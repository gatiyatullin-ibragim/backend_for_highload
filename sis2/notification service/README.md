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

