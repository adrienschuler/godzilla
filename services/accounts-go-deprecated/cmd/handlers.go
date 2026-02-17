package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/adrienschuler/godzilla/internal/httputil"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type user struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

const (
	bcryptCost    = 12
	sessionTTL    = 24 * time.Hour
	cookieName    = "auth_token"
	sessionPrefix = "session:"
	redisTimeout  = 3 * time.Second
	mongoTimeout  = 5 * time.Second
)

func decodeCreds(r *http.Request) (credentials, error) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Username == "" || c.Password == "" {
		return c, fmt.Errorf("invalid credentials")
	}
	return c, nil
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	creds, err := decodeCreds(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcryptCost)
	if err != nil {
		slog.Error("password hashing failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), mongoTimeout)
	defer cancel()

	_, err = a.users.InsertOne(ctx, user{
		Username: creds.Username,
		Password: string(hashed),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httputil.WriteError(w, http.StatusConflict, "Username already exists")
			return
		}
		slog.Error("mongo insert failed", "error", err, "username", creds.Username)
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	slog.Info("user registered", "username", creds.Username)
	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "success", "message": "User registered successfully"})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	creds, err := decodeCreds(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), mongoTimeout)
	defer cancel()

	var u user
	err = a.users.FindOne(ctx, bson.M{"username": creds.Username}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			httputil.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		slog.Error("mongo find failed", "error", err, "username", creds.Username)
		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(creds.Password)) != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	sessionID, err := generateSessionID()
	if err != nil {
		slog.Error("failed to generate session ID", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	redisCtx, redisCancel := context.WithTimeout(r.Context(), redisTimeout)
	defer redisCancel()

	if err := a.rdb.Set(redisCtx, sessionPrefix+sessionID, creds.Username, sessionTTL).Err(); err != nil {
		slog.Error("failed to store session", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	setAuthCookie(w, cookieName, sessionID, int(sessionTTL.Seconds()))
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Logged in successfully"})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), redisTimeout)
		defer cancel()
		a.rdb.Del(ctx, sessionPrefix+cookie.Value)
	}
	setAuthCookie(w, cookieName, "", -1)
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Logged out"})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-Authenticated-User")
	if user == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":    "success",
		"message":   fmt.Sprintf("Hello %s!", user),
		"username":  user,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func setAuthCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
