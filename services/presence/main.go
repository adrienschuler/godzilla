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
	cleanupDone chan struct{}
}

func newStore() *store {
	s := &store{
		online: make(map[string]int),
		typing: make(map[string]time.Time),
		cleanupDone: make(chan struct{}),
	}
	go s.cleanupTyping()
	return s
}

func (s *store) connect(username string) []string {
	s.mu.Lock()
	s.online[username]++
	users := s.onlineUsersLocked()
	s.mu.Unlock()
	return users
}

func (s *store) disconnect(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if count, exists := s.online[username]; exists && count <= 1 {
		delete(s.online, username)
		delete(s.typing, username)
	} else if exists {
		s.online[username]--
	}
}

func (s *store) setTyping(username string, isTyping bool) {
	s.mu.Lock()
	if isTyping {
		s.typing[username] = time.Now()
	} else {
		delete(s.typing, username)
	}
	s.mu.Unlock()
}

func (s *store) onlineUsers() []string {
	s.mu.RLock()
	users := s.onlineUsersLocked()
	s.mu.RUnlock()
	return users
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
	users := make([]string, 0, len(s.typing))
	for u := range s.typing {
		users = append(users, u)
	}
	s.mu.RUnlock()
	slices.Sort(users)
	return users
}

func (s *store) cleanupTyping() {
	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
		close(s.cleanupDone)
	}()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpiredTyping()
		case <-s.cleanupDone:
			return
		}
	}
}

func (s *store) cleanupExpiredTyping() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for u, t := range s.typing {
		if now.Sub(t) > 5*time.Second {
			delete(s.typing, u)
		}
	}
}

func (s *store) stopCleanup() {
	close(s.cleanupDone)
}

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

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	port := env("PORT", "50051")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	store := newStore()
	srv := grpc.NewServer()
	pb.RegisterPresenceServiceServer(srv, &server{store: store})

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
	
	// Clean up resources
	store.stopCleanup()
	srv.GracefulStop()
	slog.Info("server stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
