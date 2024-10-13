// services/auth_service.go
package services

import (
	"ddos-protection-api/auth"
	"ddos-protection-api/db"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type Claims struct {
	UserID    int    `json:"user_id"`
	AgentName string `json:"agent_name"`
	Email     string `json:"email"`
	jwt.StandardClaims
}

// CreateSession создает запись в таблице sessions при аутентификации
func CreateSession(email, token, host string) error {
	query := `INSERT INTO sessions (email, token, created_at, host) VALUES ($1, $2, CURRENT_TIMESTAMP, $3)`
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
	query := `SELECT email FROM sessions WHERE token = $1 AND deleted_at IS NULL`
	err := db.DB.QueryRow(query, token).Scan(&email)
	if err != nil {
		return 0, fmt.Errorf("токен не найден или неактивен: %v", err)
	}

	// Получение userID из таблицы users на основе email
	var userID int
	query = `SELECT id FROM users WHERE email = $1`
	err = db.DB.QueryRow(query, email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден: %v", err)
	}

	return userID, nil
}

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

	// Вызов функции LoginUser, чтобы получить userID и токен
	userID, token, err := auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Запись токена в таблицу sessions
	host := c.ClientIP()
	if err := CreateSession(req.Email, token, host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании сессии"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID, "token": token})
}

// DeleteUserHandler помечает пользователя и все связанные с ним записи как удаленные
func DeleteUserHandler(c *gin.Context) {
	// Получение user_id на основе токена
	token := c.GetHeader("Authorization")
	userID, err := db.GetUserIDByToken(token)
	email, err := db.GetEmailByToken(token)

	if err != nil {
		log.Printf("Ошибка получения пользователя по токену: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
		return
	}

	// Логируем удаление пользователя
	log.Printf("Помечаем пользователя id: %d как удаленного", userID)

	// Устанавливаем метку deleted_at для всех записей, связанных с пользователем
	deletedAt := time.Now()

	// Обновление deleted_at в таблице запросов
	queryUpdateRequests := `UPDATE requests SET deleted_at = $1 WHERE user_id = $2`
	_, err = db.DB.Exec(queryUpdateRequests, deletedAt, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при мягком удалении запросов пользователя"})
		return
	}

	// Обновление deleted_at в таблице активных сессий
	queryUpdateActiveSessions := `UPDATE active_sessions SET deleted_at = $1 WHERE user_id = $2`
	_, err = db.DB.Exec(queryUpdateActiveSessions, deletedAt, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при мягком удалении активных сессий пользователя"})
		return
	}

	// Обновление deleted_at в таблице сессий
	queryUpdateSessions := `UPDATE sessions SET deleted_at = $1 WHERE email = $2`
	_, err = db.DB.Exec(queryUpdateSessions, deletedAt, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при мягком удалении сессий пользователя"})
		return
	}

	// Обновление deleted_at в таблице веса IP
	queryUpdateIPWeights := `UPDATE ip_weights SET deleted_at = $1 WHERE user_id = $2`
	_, err = db.DB.Exec(queryUpdateIPWeights, deletedAt, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при мягком удалении данных о весе IP"})
		return
	}

	// Пометка пользователя как удаленного
	queryUpdateUser := `UPDATE users SET deleted_at = $1 WHERE id = $2`
	_, err = db.DB.Exec(queryUpdateUser, deletedAt, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при мягком удалении пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь и связанные записи успешно помечены как удаленные"})
}
