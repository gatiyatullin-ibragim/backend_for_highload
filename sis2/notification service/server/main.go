package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	pb "notification-service/pb"
	"notification-service/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server реализует NotificationServiceServer; всё состояние лежит в store.Store.
type Server struct {
	pb.UnimplementedNotificationServiceServer
	store store.Store
}

// 1. Приём уведомления с идемпотентностью (ключ хранится в БД, см. db/sql/sis2_extension.sql)
func (s *Server) SendNotification(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	id, created, err := s.store.Create(ctx, store.NewNotification{
		IdempotencyKey: req.GetIdempotencyKey(),
		UserID:         req.GetUserId(),
		TemplateID:     req.GetTemplateId(),
		Channel:        req.GetChannel(),
		Payload:        req.GetPayload(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	if !created {
		log.Printf("[gRPC] Idempotent hit! Key: %s -> existing ID: %s", req.GetIdempotencyKey(), id)
		return &pb.SendResponse{NotificationId: id, Status: "accepted (idempotent)"}, nil
	}
	return &pb.SendResponse{NotificationId: id, Status: "accepted"}, nil
}

// 2. Статус уведомления
func (s *Server) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	st, err := s.store.Status(ctx, req.GetNotificationId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.StatusResponse{NotificationId: req.GetNotificationId(), Status: st}, nil
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return status.Error(codes.NotFound, "notification not found")
	case errors.Is(err, store.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return status.FromContextError(err).Err()
	default:
		log.Printf("[store] error: %v", err)
		return status.Error(codes.Internal, "internal error")
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// STORE=memory (по умолчанию, как в SIS 2) или STORE=postgres (БД из SIS 3).
func newStore(ctx context.Context) (store.Store, error) {
	if env("STORE", "memory") != "postgres" {
		return store.NewMemory(), nil
	}
	return store.NewPostgres(ctx,
		env("DATABASE_URL", "postgres://postgres@127.0.0.1:6432/notif?sslmode=disable"),
		os.Getenv("DATABASE_RO_URL"), // пусто => читаем с primary
		3*time.Second)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	st, err := newStore(ctx)
	cancel()
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterNotificationServiceServer(grpcServer, &Server{store: st})

	log.Printf("gRPC server is running on :50051 (STORE=%s)", env("STORE", "memory"))
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
