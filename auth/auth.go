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

// Обновленная структура Claims с полем UserID

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

// Функция для хеширования пароля
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// Функция для проверки пароля
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Функция для регистрации пользователя
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

// Функция для логина пользователя
// auth/auth.go
func LoginUser(email, password string) (int, string, error) {
	var userID int
	var passwordHash string
	query := `SELECT id, password_hash FROM users WHERE email = ?`
	err := db.DB.QueryRow(query, email).Scan(&userID, &passwordHash)
	if err != nil {
		return 0, "", errors.New("пользователь не найден")
	}

	if !CheckPasswordHash(password, passwordHash) {
		return 0, "", errors.New("неверный пароль")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return 0, "", err
	}

	return userID, tokenString, nil
}

// Функция для валидации токена
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
