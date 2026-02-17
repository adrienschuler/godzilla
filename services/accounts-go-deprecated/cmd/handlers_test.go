package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func setupTestAPI(t *testing.T) *API {
	t.Helper()

	// Redis via miniredis
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// MongoDB via testcontainers
	ctx := context.Background()
	mongoC, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		t.Fatalf("failed to start mongodb container: %v", err)
	}
	t.Cleanup(func() {
		if err := mongoC.Terminate(ctx); err != nil {
			t.Logf("failed to terminate mongodb container: %v", err)
		}
	})

	connStr, err := mongoC.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get mongodb connection string: %v", err)
	}

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		t.Fatalf("failed to connect to mongodb: %v", err)
	}
	t.Cleanup(func() {
		mongoClient.Disconnect(ctx)
	})

	users := mongoClient.Database("testdb").Collection("users")
	_, err = users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	return &API{rdb: rdb, users: users}
}

func postJSON(url string, body any) *http.Request {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func TestRegister(t *testing.T) {
	api := setupTestAPI(t)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		api.register(w, postJSON("/user/register", credentials{Username: "alice", Password: "secret"}))

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
		resp := decodeResponse(t, w)
		if resp["status"] != "success" {
			t.Fatalf("expected status success, got %q", resp["status"])
		}
	})

	t.Run("duplicate user", func(t *testing.T) {
		w := httptest.NewRecorder()
		api.register(w, postJSON("/user/register", credentials{Username: "alice", Password: "other"}))

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		api.register(w, postJSON("/user/register", credentials{Username: "", Password: ""}))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestLoginLogout(t *testing.T) {
	api := setupTestAPI(t)

	// Register a user first.
	w := httptest.NewRecorder()
	api.register(w, postJSON("/user/register", credentials{Username: "bob", Password: "pass123"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("register failed: %d", w.Code)
	}

	t.Run("invalid credentials", func(t *testing.T) {
		w := httptest.NewRecorder()
		api.login(w, postJSON("/user/login", credentials{Username: "bob", Password: "wrong"}))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("successful login", func(t *testing.T) {
		w := httptest.NewRecorder()
		api.login(w, postJSON("/user/login", credentials{Username: "bob", Password: "pass123"}))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		cookies := w.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == cookieName && c.Value != "" {
				found = true
			}
		}
		if !found {
			t.Fatal("expected auth cookie to be set")
		}
	})

	t.Run("nonexistent user", func(t *testing.T) {
		w := httptest.NewRecorder()
		api.login(w, postJSON("/user/login", credentials{Username: "nobody", Password: "pass"}))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("logout", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/user/logout", nil)
		api.logout(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestMe(t *testing.T) {
	api := setupTestAPI(t)

	t.Run("authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
		req.Header.Set("X-Authenticated-User", "dave")
		api.me(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["username"] != "dave" {
			t.Fatalf("expected dave, got %v", resp["username"])
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
		api.me(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}
