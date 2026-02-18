package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/adrienschuler/godzilla/gen/presence"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

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
