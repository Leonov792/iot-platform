package auth

import (
	"testing"
	"time"
)

func TestTokenRoundtrip(t *testing.T) {
	token, err := GenerateToken("user-1", time.Hour)
	if err != nil {
		t.Fatalf("токен не подписался: %v", err)
	}
	userID, role, homeID, err := ParseToken(token)
	if err != nil {
		t.Fatalf("токен не распарсился: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("ждём user-1, пришло %s", userID)
	}
	if role != RoleOwner {
		t.Fatalf("ждём роль owner, пришло %s", role)
	}
	if homeID != "user-1" {
		t.Fatalf("ждём home_id user-1, пришло %s", homeID)
	}
}

func TestTokenWithRole(t *testing.T) {
	token, err := GenerateTokenWithRole("staff-1", RoleStaff, "owner-1", time.Hour)
	if err != nil {
		t.Fatalf("токен не подписался: %v", err)
	}
	userID, role, homeID, err := ParseToken(token)
	if err != nil {
		t.Fatalf("токен не распарсился: %v", err)
	}
	if userID != "staff-1" || role != RoleStaff || homeID != "owner-1" {
		t.Fatalf("claims не сошлись: %s/%s/%s", userID, role, homeID)
	}
}

func TestExpiredToken(t *testing.T) {
	token, err := GenerateToken("user-1", -time.Minute)
	if err != nil {
		t.Fatalf("токен не подписался: %v", err)
	}
	if _, _, _, err := ParseToken(token); err == nil {
		t.Fatal("протухший токен должен не проходить")
	}
}
