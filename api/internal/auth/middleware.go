package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const userIDKey ctxKey = "userID"

// RequireAuth проверяет Bearer-токен и кладёт user_id в контекст.
// если токена нет или он кривой — сразу 401, дальше не пускаем
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, `{"error":"нет токена"}`, http.StatusUnauthorized)
			return
		}

		raw := strings.TrimPrefix(header, "Bearer ")
		userID, err := ParseToken(raw)
		if err != nil {
			http.Error(w, `{"error":"токен протух или кривой"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromCtx вытаскивает user_id, который middleware туда положил
func UserIDFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}
