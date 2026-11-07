package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	pb "notification-service/pb"
)

const benchmarkBufferSize = 1024 * 1024

const (
	targetGRPCRequestsPerSecond = 3000
	defaultGRPCLoadTestDuration = 10 * time.Second
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
	if os.Getenv("RUN_3000_RPS") != "1" {
		t.Skip("set RUN_3000_RPS=1 to run the real-server load test")
	}

	address := os.Getenv("NOTIFICATION_GRPC_ADDR")
	if address == "" {
		address = "127.0.0.1:50051"
	}
	duration := loadTestDuration(t)

	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("connect to notification service at %s: %v", address, err)
	}
	defer conn.Close()

	client := pb.NewNotificationServiceClient(conn)
	readyCtx, cancelReady := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelReady()
	if _, err := client.SendNotification(readyCtx, benchmarkRequestWithKey("load-test-ready")); err != nil {
		t.Fatalf("notification service is not ready at %s: %v", address, err)
	}

	workers := runtime.GOMAXPROCS(0) * 4
	if workers < 16 {
		workers = 16
	}
	totalRequests := targetGRPCRequestsPerSecond * int(duration/time.Second)
	jobs := make(chan int, totalRequests)
	latencies := make(chan time.Duration, totalRequests)
	errs := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), duration+10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for requestNumber := range jobs {
				start := time.Now()
				requestCtx, requestCancel := context.WithTimeout(ctx, 5*time.Second)
				_, err := client.SendNotification(
					requestCtx,
					benchmarkRequestWithKey(fmt.Sprintf("load-test-%d", requestNumber)),
				)
				requestCancel()
				if err != nil {
					select {
					case errs <- err:
					default:
					}
					continue
				}
				latencies <- time.Since(start)
			}
		}()
	}

	start := time.Now()
	interval := time.Second / targetGRPCRequestsPerSecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for requestNumber := 0; requestNumber < totalRequests; requestNumber++ {
		if requestNumber > 0 {
			<-ticker.C
		}
		jobs <- requestNumber
	}
	close(jobs)
	wg.Wait()
	close(latencies)

	elapsed := time.Since(start)
	latencyValues := make([]time.Duration, 0, totalRequests)
	for latency := range latencies {
		latencyValues = append(latencyValues, latency)
	}
	successfulRequests := len(latencyValues)
	completionRPS := float64(successfulRequests) / elapsed.Seconds()
	offeredRPS := float64(totalRequests) / duration.Seconds()
	p95 := percentile(latencyValues, 0.95)
	t.Logf(
		"gRPC load: address=%s workers=%d offered=%.0f RPS duration=%v successful=%d/%d completion=%.0f RPS p95=%v",
		address,
		workers,
		offeredRPS,
		duration,
		successfulRequests,
		totalRequests,
		completionRPS,
		p95,
	)

	select {
	case err := <-errs:
		t.Fatalf("gRPC load test request failed: %v", err)
	default:
	}
	if successfulRequests != totalRequests {
		t.Fatalf("gRPC load test completed %d/%d requests", successfulRequests, totalRequests)
	}
}

func benchmarkRequestWithKey(key string) *pb.SendRequest {
	request := benchmarkRequest()
	request.IdempotencyKey = key
	return request
}

func loadTestDuration(t *testing.T) time.Duration {
	t.Helper()

	value := os.Getenv("LOAD_TEST_DURATION")
	if value == "" {
		return defaultGRPCLoadTestDuration
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 1 {
		t.Fatalf("LOAD_TEST_DURATION must be a positive number of seconds, got %q", value)
	}
	return time.Duration(seconds) * time.Second
}

func percentile(values []time.Duration, quantile float64) time.Duration {
	sorted := append([]time.Duration(nil), values...)
	if len(sorted) == 0 {
		return 0
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted)-1) * quantile)
	return sorted[index]
}
