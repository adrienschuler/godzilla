package main

import (
	"context"
	"log/slog"

	pb "github.com/adrienschuler/godzilla/gen/presence"
)

// server implements the PresenceService gRPC interface.
type server struct {
	pb.UnimplementedPresenceServiceServer
	store *store
}

func (s *server) UserConnected(ctx context.Context, req *pb.UserRequest) (*pb.OnlineUsersResponse, error) {
	users := s.store.connect(req.Username)
	slog.InfoContext(ctx, "user connected", "username", req.Username, "online_count", len(users))
	return &pb.OnlineUsersResponse{Usernames: users}, nil
}

func (s *server) UserDisconnected(ctx context.Context, req *pb.UserRequest) (*pb.Empty, error) {
	s.store.disconnect(req.Username)
	slog.InfoContext(ctx, "user disconnected", "username", req.Username)
	return &pb.Empty{}, nil
}

func (s *server) SetTyping(ctx context.Context, req *pb.SetTypingRequest) (*pb.Empty, error) {
	s.store.setTyping(req.Username, req.IsTyping)
	action := "started"
	if !req.IsTyping {
		action = "stopped"
	}
	slog.InfoContext(ctx, "user typing", "username", req.Username, "action", action)
	return &pb.Empty{}, nil
}

func (s *server) GetOnlineUsers(ctx context.Context, _ *pb.Empty) (*pb.OnlineUsersResponse, error) {
	users := s.store.onlineUsers()
	slog.DebugContext(ctx, "get online users", "count", len(users))
	return &pb.OnlineUsersResponse{Usernames: users}, nil
}

func (s *server) GetTypingUsers(ctx context.Context, _ *pb.Empty) (*pb.TypingUsersResponse, error) {
	users := s.store.typingUsers()
	slog.DebugContext(ctx, "get typing users", "count", len(users))
	return &pb.TypingUsersResponse{Usernames: users}, nil
}
