package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/yourusername/incident-ticketing/pkg/jwtutil"
)

type contextKey string

const (
	contextUserIDKey contextKey = "user_id"
	contextRoleKey   contextKey = "role"
)

// Auth verifies Bearer token and stores claims in request context.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header")
			return
		}

		claims, err := jwtutil.VerifyToken(parts[1])
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), contextUserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, contextRoleKey, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext returns authenticated user id from request context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(contextUserIDKey)
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return id, id != uuid.Nil
}

// RoleFromContext returns authenticated role from request context.
func RoleFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(contextRoleKey)
	role, ok := v.(string)
	if !ok || role == "" {
		return "", false
	}
	return role, true
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
