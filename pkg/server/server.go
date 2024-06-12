package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"test-authservice/pkg/errs"
	"test-authservice/pkg/models"
)

type Core interface {
	IssueTokens(ctx context.Context, id string, ip string) (models.Tokens, error)
	RefreshTokens(ctx context.Context, rToken string, ip string) (models.Tokens, error)
}

func NewServer(ctx context.Context, port int, core Core) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tokens", getTokens(core))
	mux.HandleFunc("GET /refresh", refreshTokens(core))
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}

func getTokens(c Core) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			slog.Info("Invalid token pair request with empty id")
			http.Error(w, "You must provide user id in query params", http.StatusBadRequest)
			return
		}
		slog.Info("New token pair request", "id", id)
		tokens, err := c.IssueTokens(r.Context(), id, strings.Split(r.RemoteAddr, ":")[0])
		if err != nil {
			if errors.Is(err, errs.ErrNoSuchUser) {
				slog.Info("No user found", "id", id)
				http.Error(w, "No such user", http.StatusNotFound)
				return
			}
			slog.Error("Error issuing tokens", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		tokens.Refresh = base64.StdEncoding.EncodeToString([]byte(tokens.Refresh))

		response, err := json.Marshal(tokens)
		if err != nil {
			slog.Error("Error encoding JSON", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

func refreshTokens(c Core) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rBase64 := r.Header.Get("Authorization")
		if rBase64 == "" {
			slog.Info("Invalid refresh request with empty Authorization header")
			http.Error(w, "You must provide Authorization header", http.StatusBadRequest)
			return
		}
		rBytes, err := base64.StdEncoding.DecodeString(rBase64)
		if err != nil {
			slog.Error("Error decoding refresh token", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("New refresh request")

		refresh, _ := strings.CutPrefix(string(rBytes), "Bearer ")
		tokens, err := c.RefreshTokens(r.Context(), refresh, strings.Split(r.RemoteAddr, ":")[0])

		switch {
		case errors.Is(err, errs.ErrInvalidRefresh):
			slog.Error("Invalid refresh token", "token", refresh)
			http.Error(w, "Invalid refresh token", http.StatusForbidden)
			return
		case errors.Is(err, errs.ErrNoSuchUser):
			slog.Error("Refresh for non existing user", "token", refresh)
			http.Error(w, "No such user", http.StatusForbidden)
			return
		case errors.Is(err, errs.ErrWrongRefresh):
			slog.Error("Wrong token", "token", refresh)
			http.Error(w, "Use last refresh token", http.StatusForbidden)
			return
		case err != nil:
			slog.Error("Error refreshing tokens", "error", err, "token", refresh)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		tokens.Refresh = base64.StdEncoding.EncodeToString([]byte(tokens.Refresh))

		response, err := json.Marshal(tokens)
		if err != nil {
			slog.Error("Error encoding JSON", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}
