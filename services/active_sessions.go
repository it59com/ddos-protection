package services

import (
	"ddos-protection-api/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Обновление статуса активности пользователя в базе данных
func updateSessionStatus(userID int, token string, agentName string, status string) error {
	query := `
		INSERT INTO active_sessions (user_id, token, agent_name, status, last_active)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(token) DO UPDATE SET status = ?, last_active = CURRENT_TIMESTAMP
	`
	_, err := db.DB.Exec(query, userID, token, agentName, status, status)
	return err
}

// Удаление статуса активности при отключении
func removeSession(token string) error {
	query := `UPDATE active_sessions SET status = 'offline', last_active = CURRENT_TIMESTAMP WHERE token = ?`
	_, err := db.DB.Exec(query, token)
	return err
}

// GetActiveSessions возвращает список активных сессий
func GetActiveSessions(c *gin.Context) {
	rows, err := db.DB.Query("SELECT user_id, agent_name, status, last_active FROM active_sessions WHERE status = 'online'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении активных сессий"})
		return
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var userID int
		var agentName, status string
		var lastActive time.Time

		if err := rows.Scan(&userID, &agentName, &status, &lastActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке данных сессий"})
			return
		}

		sessions = append(sessions, map[string]interface{}{
			"user_id":     userID,
			"agent_name":  agentName,
			"status":      status,
			"last_active": lastActive,
		})
	}

	c.JSON(http.StatusOK, gin.H{"active_sessions": sessions})
}
