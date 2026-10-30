package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "notification-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server реализует интерфейс NotificationServiceServer
type Server struct {
	//заглушка
	pb.UnimplementedNotificationServiceServer

	mu            sync.RWMutex
	notifications map[string]string // Хранилище статусов
	processedKeys map[string]string // Хранилище для идемпотентности (ключ -> id уведомления)
}

// 1. Метод принятия уведомления с поддержкой идемпотентности
func (s *Server) SendNotification(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	//лок блокирует доступ к мапе для остальных горутин обеспечивая потокобезопасность
	s.mu.Lock()
	defer s.mu.Unlock()

	// Получаем ключ идемпотентности из запроса
	clientKey := req.GetIdempotencyKey()

	// Проверяем, обрабатывали ли мы этот запрос ранее
	if clientKey != "" {
		if existingNotifID, found := s.processedKeys[clientKey]; found {
			log.Printf("[gRPC] Idempotent hit! Key: %s -> Returning existing ID: %s", clientKey, existingNotifID)
			return &pb.SendResponse{
				NotificationId: existingNotifID,
				Status:         "accepted (idempotent)",
			}, nil
		}
	}

	// Генерируем новый ID уведомления
	notifID := fmt.Sprintf("notif-%d", len(s.notifications)+1)

	// Сохраняем в «базу»
	s.notifications[notifID] = "accepted"

	// Запоминаем ключ, чтобы повторный запрос сработал идемпотентно
	if clientKey != "" {
		s.processedKeys[clientKey] = notifID
	}

	log.Printf("[gRPC] Created new notification: %s for user: %s", notifID, req.GetUserId())

	return &pb.SendResponse{
		NotificationId: notifID,
		Status:         "accepted",
	}, nil
}

// 2. Метод получения статуса
func (s *Server) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	//ReadLock потому что только читаем данные и ничего не записываем
	s.mu.RLock()
	defer s.mu.RUnlock()

	notifID := req.GetNotificationId()
	currentStatus, exists := s.notifications[notifID]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "notification %s not found", notifID)
	}

	return &pb.StatusResponse{
		NotificationId: notifID,
		Status:         currentStatus,
	}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Инициализируем сервер и обе мапы (для статусов и для идемпотентности)
	notificationServer := &Server{
		notifications: make(map[string]string),
		processedKeys: make(map[string]string),
	}

	pb.RegisterNotificationServiceServer(grpcServer, notificationServer)

	log.Println("gRPC server is running on port :50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
