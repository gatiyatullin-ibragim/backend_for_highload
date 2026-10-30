package main

import (
	"context"
	"encoding/json"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	pb "notification-service/pb"
)

const benchmarkBufferSize = 1024 * 1024

const (
	targetGRPCRequestsPerSecond = 3000
	grpcLoadTestDuration        = time.Second
)

type benchmarkNotificationServer struct {
	pb.UnimplementedNotificationServiceServer
}

func (benchmarkNotificationServer) SendNotification(
	context.Context,
	*pb.SendRequest,
) (*pb.SendResponse, error) {
	return &pb.SendResponse{
		NotificationId: "notif-benchmark",
		Status:         "accepted",
	}, nil
}

func benchmarkRequest() *pb.SendRequest {
	return &pb.SendRequest{
		IdempotencyKey: "key-12345",
		UserId:         "user-999",
		TemplateId:     "order_shipped",
		Channel:        "push",
		Payload:        "Hello via gRPC",
	}
}

func benchmarkRESTRequest() RestSendRequest {
	return RestSendRequest{
		IdempotencyKey: "key-12345",
		UserId:         "user-999",
		TemplateID:     "order_shipped",
		Channel:        "push",
		Payload:        "Hello via REST",
	}
}

func TestSerializedRequestSizes(t *testing.T) {
	grpcBytes, err := proto.Marshal(benchmarkRequest())
	if err != nil {
		t.Fatalf("marshal protobuf request: %v", err)
	}

	jsonBytes, err := json.Marshal(benchmarkRESTRequest())
	if err != nil {
		t.Fatalf("marshal JSON request: %v", err)
	}

	t.Logf("protobuf request size: %d bytes", len(grpcBytes))
	t.Logf("JSON request size: %d bytes", len(jsonBytes))
}

func BenchmarkProtobufMarshal(b *testing.B) {
	req := benchmarkRequest()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := proto.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(data)))
	}
}

func BenchmarkJSONMarshal(b *testing.B) {
	req := benchmarkRESTRequest()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(data)))
	}
}

func newBenchmarkGRPCClient(t testing.TB) (pb.NotificationServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(benchmarkBufferSize)
	grpcServer := grpc.NewServer()
	pb.RegisterNotificationServiceServer(grpcServer, benchmarkNotificationServer{})

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Errorf("serve benchmark gRPC server: %v", err)
		}
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("dial benchmark gRPC server: %v", err)
	}

	cleanup := func() {
		conn.Close()
		grpcServer.Stop()
		listener.Close()
	}

	return pb.NewNotificationServiceClient(conn), cleanup
}

func BenchmarkGRPCSendNotification(b *testing.B) {
	client, cleanup := newBenchmarkGRPCClient(b)
	defer cleanup()

	req := benchmarkRequest()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := client.SendNotification(ctx, req); err != nil {
			b.Fatal(err)
		}
	}
}

func TestGRPC3000RPS(t *testing.T) {
	client, cleanup := newBenchmarkGRPCClient(t)
	defer cleanup()

	workers := runtime.GOMAXPROCS(0) * 4
	if workers < 16 {
		workers = 16
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcLoadTestDuration+time.Second)
	defer cancel()

	var successfulRequests atomic.Int64
	errCh := make(chan error, workers)
	start := time.Now()
	deadline := start.Add(grpcLoadTestDuration)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()

			for time.Now().Before(deadline) {
				if _, err := client.SendNotification(ctx, benchmarkRequest()); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				successfulRequests.Add(1)
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	close(errCh)

	if err, ok := <-errCh; ok {
		t.Fatalf("gRPC load test request failed: %v", err)
	}

	requests := successfulRequests.Load()
	actualRPS := float64(requests) / elapsed.Seconds()
	t.Logf(
		"gRPC load: %d workers, %d successful requests in %v (%.0f RPS)",
		workers,
		requests,
		elapsed,
		actualRPS,
	)

	if actualRPS < targetGRPCRequestsPerSecond {
		t.Fatalf(
			"gRPC throughput %.0f RPS is below target %d RPS",
			actualRPS,
			targetGRPCRequestsPerSecond,
		)
	}
}
