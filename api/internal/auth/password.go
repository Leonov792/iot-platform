package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword — bcrypt с дефолтным cost. Медленно, но так и задумано
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword сверяет пароль с хэшем
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
