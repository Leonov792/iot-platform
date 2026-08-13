package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret — TODO: вынести в env, а то светить секрет в коде последнее дело
var JWTSecret = []byte("dev-secret-tut-potom-zamenyu")

// claims — что кладём в токен
type claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string, ttl time.Duration) (string, error) {
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(JWTSecret)
}

// ParseToken возвращает user_id из валидного токена
func ParseToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			// комплятору пофиг на метод подписи, а вот продю не
			return nil, errors.New("не тот метод подписи")
		}
		return JWTSecret, nil
	})
	if err != nil {
		return "", err
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return "", errors.New("токен не валиден")
	}
	return c.UserID, nil
}
