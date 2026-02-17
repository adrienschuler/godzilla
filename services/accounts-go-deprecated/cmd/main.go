package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adrienschuler/godzilla/internal/httputil"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type API struct {
	rdb   *redis.Client
	users *mongo.Collection
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Redis (sessions)
	redisHost := env("REDIS_SERVICE_SERVICE_HOST", "127.0.0.1")
	redisPort := env("REDIS_SERVICE_SERVICE_PORT", "6379")
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisHost + ":" + redisPort,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("redis connection failed", "error", err)
	} else {
		slog.Info("redis connected")
	}

	// MongoDB (users)
	mongoURI := env("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		slog.Error("mongo connection failed", "error", err)
		os.Exit(1)
	}
	if err := mongoClient.Ping(context.Background(), nil); err != nil {
		slog.Error("mongo ping failed", "error", err)
	} else {
		slog.Info("mongo connected")
	}

	dbName := env("MONGO_DB", "godzilla")
	users := mongoClient.Database(dbName).Collection("users")

	// Ensure unique index on username
	_, err = users.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		slog.Error("failed to create username index", "error", err)
		os.Exit(1)
	}

	api := &API{rdb: rdb, users: users}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.healthz)
	mux.HandleFunc("POST /user/register", api.register)
	mux.HandleFunc("POST /user/login", api.login)
	mux.HandleFunc("POST /user/logout", api.logout)
	mux.HandleFunc("GET /user/me", api.me)

	listenAddr := ":" + env("PORT", "8081")
	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      httputil.WithLogging(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	if err := mongoClient.Disconnect(ctx); err != nil {
		slog.Error("mongo close error", "error", err)
	}
	if err := rdb.Close(); err != nil {
		slog.Error("redis close error", "error", err)
	}
	slog.Info("server stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), mongoTimeout)
	defer cancel()
	if err := a.rdb.Ping(ctx).Err(); err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "redis unavailable")
		return
	}
	if err := a.users.Database().Client().Ping(ctx, nil); err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "mongo unavailable")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
