package auth

import "testing"

func TestHashAndCheck(t *testing.T) {
	hash, err := HashPassword("секрет")
	if err != nil {
		t.Fatalf("хэш не сделался: %v", err)
	}
	if hash == "секрет" {
		t.Fatal("пароль не должен лежать открытым")
	}
	if !CheckPassword(hash, "секрет") {
		t.Fatal("правильный пароль не прошёл проверку")
	}
	if CheckPassword(hash, "неверный") {
		t.Fatal("левак почему-то прошёл")
	}
}
