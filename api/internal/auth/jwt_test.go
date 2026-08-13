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
	userID, err := ParseToken(token)
	if err != nil {
		t.Fatalf("токен не распарсился: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("ждём user-1, пришло %s", userID)
	}
}

func TestExpiredToken(t *testing.T) {
	token, err := GenerateToken("user-1", -time.Minute)
	if err != nil {
		t.Fatalf("токен не подписался: %v", err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Fatal("протухший токен должен не проходить")
	}
}
