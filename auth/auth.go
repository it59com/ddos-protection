package auth

import (
	"ddos-protection-api/db"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("your_secret_key") // Замените на более надежный секретный ключ

type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func RegisterUser(email, password string) error {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	query := `INSERT INTO users (email, password_hash) VALUES (?, ?);`
	_, err = db.DB.Exec(query, email, passwordHash)
	if err != nil {
		return fmt.Errorf("ошибка при регистрации пользователя: %w", err)
	}

	return nil
}

func LoginUser(email, password string) (string, error) {
	var passwordHash string
	query := `SELECT password_hash FROM users WHERE email = ?;`
	err := db.DB.QueryRow(query, email).Scan(&passwordHash)
	if err != nil {
		return "", errors.New("пользователь не найден")
	}

	if !CheckPasswordHash(password, passwordHash) {
		return "", errors.New("неверный пароль")
	}

	// Создание токена
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("недействительный токен")
	}

	return claims, nil
}
