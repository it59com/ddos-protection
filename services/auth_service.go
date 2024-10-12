// services/auth_service.go
package services

import (
	"ddos-protection-api/auth"
	"ddos-protection-api/db"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateSession создает запись в таблице sessions при аутентификации
func CreateSession(email, token, host string) error {
	query := `INSERT INTO sessions (email, token, created_at, host) VALUES (?, ?, CURRENT_TIMESTAMP, ?)`
	_, err := db.DB.Exec(query, email, token, host)
	if err != nil {
		return fmt.Errorf("не удалось создать сессию: %v", err)
	}
	return nil
}

// GetUserIDByToken получает userID на основе токена из таблицы sessions
func GetUserIDByToken(token string) (int, error) {
	// Удаление префикса "Bearer " из токена, если он присутствует
	token = strings.TrimPrefix(token, "Bearer ")

	// Получение email из таблицы sessions на основе token
	var email string
	query := `SELECT email FROM sessions WHERE token = ?`
	err := db.DB.QueryRow(query, token).Scan(&email)
	if err != nil {
		return 0, fmt.Errorf("токен не найден или неактивен: %v", err)
	}

	// Получение userID из таблицы users на основе email
	var userID int
	query = `SELECT id FROM users WHERE email = ?`
	err = db.DB.QueryRow(query, email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден: %v", err)
	}

	return userID, nil
}

// GetUserIDByToken получает userID на основе токена из таблицы sessions
func GetUserIDByToken_old(token string) (int, error) {
	// Удаление префикса "Bearer " из токена, если он присутствует
	token = strings.TrimPrefix(token, "Bearer ")

	// Получение email из таблицы sessions на основе token
	var email string
	query := `SELECT email FROM sessions WHERE token = ?`
	err := db.DB.QueryRow(query, token).Scan(&email)
	if err != nil {
		return 0, fmt.Errorf("токен не найден или неактивен: %v", err)
	}

	// Получение userID из таблицы users на основе email
	var userID int
	query = `SELECT id FROM users WHERE email = ?`
	err = db.DB.QueryRow(query, email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден: %v", err)
	}

	return userID, nil
}

// Функция для аутентификации пользователя
// LoginHandler обработчик для аутентификации пользователя
func LoginHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат данных"})
		return
	}

	// Проверка учетных данных пользователя и генерация токена
	token, err := auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Запись токена в таблицу sessions
	host := c.ClientIP()
	if err := db.CreateSession(req.Email, token, host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании сессии"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
