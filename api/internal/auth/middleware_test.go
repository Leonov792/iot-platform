package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromCtx(r.Context())
		if !ok {
			t.Fatal("user_id не положился в контекст")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userID))
	})

	handler := RequireAuth(next)

	token, err := GenerateToken("user-1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ждём 200, пришло %d", rec.Code)
	}
	if rec.Body.String() != "user-1" {
		t.Fatalf("ждём в теле user-1, пришло %q", rec.Body.String())
	}
}

func TestRequireAuthNoToken(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ждём 401, пришло %d", rec.Code)
	}
}
