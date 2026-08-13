package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestRegister(t *testing.T) {
	h, _, _, _, _ := newTestRouter()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": "a@b.c", "password": "secret1"}, nil)

	if rec.Code != http.StatusCreated {
		t.Fatalf("ждём 201, пришло %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["token"] == "" {
		t.Fatal("токен не вернулся")
	}
}

func TestRegisterShortPassword(t *testing.T) {
	h, _, _, _, _ := newTestRouter()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": "a@b.c", "password": "123"}, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ждём 400 на короткий пароль, пришло %d", rec.Code)
	}
}

func TestLogin(t *testing.T) {
	h, _, _, _, _ := newTestRouter()

	doJSON(t, h, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": "a@b.c", "password": "secret1"}, nil)

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "a@b.c", "password": "secret1"}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("ждём 200, пришло %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h, _, _, _, _ := newTestRouter()

	doJSON(t, h, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": "a@b.c", "password": "secret1"}, nil)

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "a@b.c", "password": "wrong"}, nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ждём 401 на неверный пароль, пришло %d", rec.Code)
	}
}
