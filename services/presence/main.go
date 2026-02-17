package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	pb "github.com/adrienschuler/godzilla/gen/presence"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// store holds in-memory presence state.
type store struct {
	mu     sync.RWMutex
	online map[string]int       // username -> connection count
	typing map[string]time.Time // username -> last typing timestamp
}

func newStore() *store {
	s := &store{
		online: make(map[string]int),
		typing: make(map[string]time.Time),
	}
	go s.cleanupTyping()
	return s
}

func (s *store) connect(username string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.online[username]++
	return s.onlineUsersLocked()
}

func (s *store) disconnect(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.online[username] <= 1 {
		delete(s.online, username)
		delete(s.typing, username)
	} else {
		s.online[username]--
	}
}

func (s *store) setTyping(username string, isTyping bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if isTyping {
		s.typing[username] = time.Now()
	} else {
		delete(s.typing, username)
	}
}

func (s *store) onlineUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.onlineUsersLocked()
}

func (s *store) onlineUsersLocked() []string {
	users := make([]string, 0, len(s.online))
	for u := range s.online {
		users = append(users, u)
	}
	slices.Sort(users)
	return users
}

func (s *store) typingUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]string, 0, len(s.typing))
	for u := range s.typing {
		users = append(users, u)
	}
	slices.Sort(users)
	return users
}

func (s *store) cleanupTyping() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for u, t := range s.typing {
			if time.Since(t) > 5*time.Second {
				delete(s.typing, u)
			}
		}
		s.mu.Unlock()
	}
}

// server implements the PresenceService gRPC interface.
type server struct {
	pb.UnimplementedPresenceServiceServer
	store *store
}

func (s *server) UserConnected(_ context.Context, req *pb.UserRequest) (*pb.OnlineUsersResponse, error) {
	slog.Info("user connected", "username", req.Username)
	users := s.store.connect(req.Username)
	return &pb.OnlineUsersResponse{Usernames: users}, nil
}

func (s *server) UserDisconnected(_ context.Context, req *pb.UserRequest) (*pb.Empty, error) {
	slog.Info("user disconnected", "username", req.Username)
	s.store.disconnect(req.Username)
	return &pb.Empty{}, nil
}

func (s *server) SetTyping(_ context.Context, req *pb.SetTypingRequest) (*pb.Empty, error) {
	s.store.setTyping(req.Username, req.IsTyping)
	return &pb.Empty{}, nil
}

func (s *server) GetOnlineUsers(_ context.Context, _ *pb.Empty) (*pb.OnlineUsersResponse, error) {
	return &pb.OnlineUsersResponse{Usernames: s.store.onlineUsers()}, nil
}

func (s *server) GetTypingUsers(_ context.Context, _ *pb.Empty) (*pb.TypingUsersResponse, error) {
	return &pb.TypingUsersResponse{Usernames: s.store.typingUsers()}, nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	port := env("PORT", "50051")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	pb.RegisterPresenceServiceServer(srv, &server{store: newStore()})

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	go func() {
		slog.Info("listening", "addr", ":"+port)
		if err := srv.Serve(lis); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())
	srv.GracefulStop()
	slog.Info("server stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
