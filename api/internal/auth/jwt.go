package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret — секрет подписи. Дефолт только для локальной разработки и тестов;
// в проде обязан задаваться через env JWT_SECRET (см. config.Load -> SetSecret).
var JWTSecret = []byte("dev-secret-tut-potom-zamenyu")

// SetSecret переопределяет секрет из конфига. Вызывается в main до первого
// использования GenerateToken/ParseToken.
func SetSecret(s []byte) {
	if len(s) > 0 {
		JWTSecret = s
	}
}

// claims — что кладём в токен
type claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	HomeID string `json:"home_id"`
	jwt.RegisteredClaims
}

// GenerateToken — токен для владельца (роль owner, home_id = userID).
// для остальных ролей используй GenerateTokenWithRole.
func GenerateToken(userID string, ttl time.Duration) (string, error) {
	return GenerateTokenWithRole(userID, RoleOwner, userID, ttl)
}

func GenerateTokenWithRole(userID, role, homeID string, ttl time.Duration) (string, error) {
	if role == "" {
		role = RoleOwner
	}
	if homeID == "" {
		homeID = userID
	}
	c := claims{
		UserID: userID,
		Role:   role,
		HomeID: homeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(JWTSecret)
}

// ParseToken возвращает user_id, роль и home_id из валидного токена
func ParseToken(tokenString string) (userID, role, homeID string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			// комплятору пофиг на метод подписи, а вот продю не
			return nil, errors.New("не тот метод подписи")
		}
		return JWTSecret, nil
	})
	if err != nil {
		return "", "", "", err
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return "", "", "", errors.New("токен не валиден")
	}
	return c.UserID, c.Role, c.HomeID, nil
}
