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
// It manages online users and their typing status.
type store struct {
	mu           sync.RWMutex
	online       map[string]int       // username -> connection count
	typing       map[string]time.Time // username -> last typing timestamp
	cleanupDone  chan struct{}       // channel to signal cleanup goroutine to stop
}

// newStore creates and initializes a new store instance.
// It starts the background cleanup goroutine for typing status.
// Returns a pointer to the initialized store.
func newStore() *store {
	s := &store{
		online: make(map[string]int),
		typing: make(map[string]time.Time),
		cleanupDone: make(chan struct{}),
	}
	go s.cleanupTyping()
	return s
}

// connect increments the connection count for a user and returns the current list of online users.
// Parameters:
//   username: The username of the connecting user.
// Returns:
//   A sorted slice of all currently online usernames.
func (s *store) connect(username string) []string {
	s.mu.Lock()
	s.online[username]++
	users := s.onlineUsersLocked()
	s.mu.Unlock()
	return users
}

// disconnect decrements the connection count for a user and removes them if no connections remain.
// Parameters:
//   username: The username of the disconnecting user.
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

// setTyping updates the typing status for a user.
// Parameters:
//   username: The username of the user.
//   isTyping: Boolean indicating if the user is typing (true) or not (false).
func (s *store) setTyping(username string, isTyping bool) {
	s.mu.Lock()
	if isTyping {
		s.typing[username] = time.Now()
	} else {
		delete(s.typing, username)
	}
	s.mu.Unlock()
}

// onlineUsers returns a sorted slice of all currently online usernames.
// Returns:
//   A sorted slice of online usernames.
func (s *store) onlineUsers() []string {
	s.mu.RLock()
	users := s.onlineUsersLocked()
	s.mu.RUnlock()
	return users
}

// onlineUsersLocked returns a sorted slice of online usernames.
// Must be called with s.mu held in read or write mode.
// Returns:
//   A sorted slice of online usernames.
func (s *store) onlineUsersLocked() []string {
	users := make([]string, 0, len(s.online))
	for u := range s.online {
		users = append(users, u)
	}
	slices.Sort(users)
	return users
}

// typingUsers returns a sorted slice of usernames who are currently typing.
// Returns:
//   A sorted slice of usernames who are typing.
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

// cleanupTyping runs in a background goroutine to periodically clean up expired typing statuses.
// It stops when the cleanupDone channel is closed.
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

// cleanupExpiredTyping removes typing statuses that haven't been updated in the last 5 seconds.
// Must be called with s.mu held in write mode.
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

// stopCleanup signals the cleanup goroutine to stop by closing the cleanupDone channel.
func (s *store) stopCleanup() {
	close(s.cleanupDone)
}

// server implements the PresenceService gRPC interface.
// It handles gRPC requests and interacts with the store for presence data.
type server struct {
	pb.UnimplementedPresenceServiceServer
	store *store
}

// UserConnected handles a user connection event.
// Parameters:
//   ctx: The context for the gRPC call.
//   req: The user request containing the username.
// Returns:
//   OnlineUsersResponse containing the current list of online users.
//   error: Any error that occurred during processing.
func (s *server) UserConnected(ctx context.Context, req *pb.UserRequest) (*pb.OnlineUsersResponse, error) {
	users := s.store.connect(req.Username)
	slog.InfoContext(ctx, "user connected", "username", req.Username, "online_count", len(users))
	return &pb.OnlineUsersResponse{Usernames: users}, nil
}

// UserDisconnected handles a user disconnection event.
// Parameters:
//   ctx: The context for the gRPC call.
//   req: The user request containing the username.
// Returns:
//   Empty response.
//   error: Any error that occurred during processing.
func (s *server) UserDisconnected(ctx context.Context, req *pb.UserRequest) (*pb.Empty, error) {
	s.store.disconnect(req.Username)
	slog.InfoContext(ctx, "user disconnected", "username", req.Username)
	return &pb.Empty{}, nil
}

// SetTyping updates the typing status for a user.
// Parameters:
//   ctx: The context for the gRPC call.
//   req: The set typing request containing username and typing status.
// Returns:
//   Empty response.
//   error: Any error that occurred during processing.
func (s *server) SetTyping(ctx context.Context, req *pb.SetTypingRequest) (*pb.Empty, error) {
	s.store.setTyping(req.Username, req.IsTyping)
	action := "started"
	if !req.IsTyping {
		action = "stopped"
	}
	slog.InfoContext(ctx, "user typing", "username", req.Username, "action", action)
	return &pb.Empty{}, nil
}

// GetOnlineUsers retrieves the current list of online users.
// Parameters:
//   ctx: The context for the gRPC call.
//   _: Empty request (unused).
// Returns:
//   OnlineUsersResponse containing the list of online users.
//   error: Any error that occurred during processing.
func (s *server) GetOnlineUsers(ctx context.Context, _ *pb.Empty) (*pb.OnlineUsersResponse, error) {
	users := s.store.onlineUsers()
	slog.DebugContext(ctx, "get online users", "count", len(users))
	return &pb.OnlineUsersResponse{Usernames: users}, nil
}

// GetTypingUsers retrieves the current list of users who are typing.
// Parameters:
//   ctx: The context for the gRPC call.
//   _: Empty request (unused).
// Returns:
//   TypingUsersResponse containing the list of typing users.
//   error: Any error that occurred during processing.
func (s *server) GetTypingUsers(ctx context.Context, _ *pb.Empty) (*pb.TypingUsersResponse, error) {
	users := s.store.typingUsers()
	slog.DebugContext(ctx, "get typing users", "count", len(users))
	return &pb.TypingUsersResponse{Usernames: users}, nil
}

// main is the entry point for the presence service.
// It initializes the gRPC server, sets up health checks, and handles graceful shutdown.
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

// env retrieves an environment variable with a fallback value.
// Parameters:
//   key: The name of the environment variable.
//   fallback: The default value to use if the variable is not set.
// Returns:
//   The value of the environment variable or the fallback if not set.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
