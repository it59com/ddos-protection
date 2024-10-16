package services

import (
	"ddos-protection-api/db"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Обновление статуса активности пользователя в базе данных
func updateSessionStatus(userID int, token string, agentName string, status string) error {
	query := `
		INSERT INTO active_sessions (user_id, token, agent_name, status, last_active)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT(token) DO UPDATE SET status = $4, last_active = CURRENT_TIMESTAMP
	`
	_, err := db.DB.Exec(query, userID, token, agentName, status)
	return err
}

// Удаление статуса активности при отключении
func removeSession(token string) error {
	query := `UPDATE active_sessions SET status = 'offline', last_active = CURRENT_TIMESTAMP WHERE token = $1`
	_, err := db.DB.Exec(query, token)
	return err
}

// GetActiveSessions возвращает список активных сессий
func GetActiveSessions() ([]map[string]interface{}, error) {
	query := `
		SELECT user_id, agent_name, status, last_active 
		FROM active_sessions 
		WHERE status = 'online' AND deleted_at IS NULL
	`
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var userID int
		var agentName, status string
		var lastActive time.Time

		if err := rows.Scan(&userID, &agentName, &status, &lastActive); err != nil {
			return nil, err
		}

		sessions = append(sessions, map[string]interface{}{
			"user_id":     userID,
			"agent_name":  agentName,
			"status":      status,
			"last_active": lastActive,
		})
	}

	return sessions, nil
}

// Обработчик для маршрута /active_sessions
func GetActiveSessionsHandler(c *gin.Context) {
	sessions, err := GetActiveSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении активных сессий"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_sessions": sessions})
}

// UpdateAgentSession обновляет или вставляет сессию агента
func UpdateAgentSession(userID int, token, agentName string) error {
	query := `
		INSERT INTO agent_sessions (user_id, token, agent_name, last_active)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
	`
	_, err := db.DB.Exec(query, userID, token, agentName)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении сессии агента: %w", err)
	}

	return nil
}
