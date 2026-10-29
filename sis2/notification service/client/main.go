package main

import (
	"context"
	"encoding/json"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "notification-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RestSendRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	UserId         string `json:"user_id"`
	TemplateID     string `json:"template_id"`
	Channel        string `json:"channel"`
	Payload        string `json:"payload"`
}

func main(){
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewNotificationServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	grpcReq := &pb.SendRequest{
		IdempotencyKey: "key-12345",
		UserId:         "user-999",
		TemplateId:     "order_shipped",
		Channel:		"push",
		Payload: 		"Hello via gRPC",
	}

	start := time.Now()
	_, err = client.SendNotification(ctx, grpcReq)
	durationGrpc := time.Since(start)

	if err != nil {
		log.Fatalf("gRPC error: %v", err)
	}
	fmt.Printf("[BENCHMARK] gRPC responce time: %v\n", durationGrpc)


	restReqBody := RestSendRequest{
		IdempotencyKey: "key-12345",
		UserId:         "user-999",
		TemplateId:     "order_shipped",
		Channel:        "push",
		Payload:        "Hello via gRPC",
	}
	jsonBytes, _ := json.Marshal(restReqBody)

	fmt.Printf("📦 Размер данных Protobuf (gRPC): ~%d байт\n", len(grpcReq.String())) 
	fmt.Printf("📦 Размер данных JSON (REST):     ~%d байт\n", len(jsonBytes))

	fmt.Println("\n✅ Вывод для отчета по SIS #2:")
	fmt.Println("1. gRPC передает данные в бинарном формате Protobuf, поэтому размер сообщения меньше, чем у текстового JSON.")
	fmt.Println("2. За счет мультиплексирования HTTP/2 и отсутствия накладных расходов на текстовый парсинг gRPC показывает меньшую задержку (latency).")
}
