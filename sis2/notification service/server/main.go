package main

import (
	context "context"
	fmt "fmt"
	log "log"
	net "net"
	sync "sync"

	pb "notification-service/pb"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedNotificationServiceServer

	mu            sync.RWMutex
	notifications map[string]string

	processedKeys map[string]string
}

func (s *Server) SendNotification(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//idempotency check
	if clientKey != ""{
		if existingNotifyID, found := processedKeys[clientKey]; found{
			log.Pringf("[gRPC] Idempotent hit! Key %s -> Returning existing ID: %s, clientKey, existingNotifID")
			return &pb.SendResponse{
				NorificationId: existingNotifID,
				Status: "accepted (idempotent)",
			},nil
		}
	}

	notifID := fmt.Sprintf("notif-%d", len(s.notifications)+1)
	s.notifications[notifID] = "accepted"

	log.Printf("[gRPC] Received notification for user: %s, channel: %s -> Assigned ID: %s", req.GetUserId(), req.GetChannel(), notifID)

	return &pb.SendResponse{
		NotificationId: notifID,
		Status:         "accepted",
	}, nil
}

func (s *Server) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
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
		log.Fatalf("failed to Listen: %v", err)
	}

	grpcServer := grpc.NewServer()

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
