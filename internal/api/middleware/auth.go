package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"problem-search/internal/auth"
)

type contextKey string

const userIDKey contextKey = "user_id"

func RequireAuth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
				return
			}

			userID, err := authService.ValidateAccessToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid access token"}`, http.StatusUnauthorized)
				return
			}

			requestContext := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(requestContext))
		})
	}
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}
