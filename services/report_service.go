// services/report_service.go
package services

import (
	"ddos-protection-api/db"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// BlockReport содержит структуру для отчета по блокировкам
type BlockReport struct {
	IP             string    `json:"ip"`
	Host           string    `json:"host"`
	RequestCount   int       `json:"request_count"`
	LastRequest    time.Time `json:"last_request"`
	FirewallSource string    `json:"firewall_source"`
}

// GenerateBlockReport создает отчет по блокировкам для определенного пользователя
func GenerateBlockReport(userID, requestLimit int) ([]BlockReport, error) {
	query := `
        SELECT ip, host, request_count, last_request, firewall_source 
        FROM requests 
        WHERE request_count >= ? AND user_id = ? 
        ORDER BY last_request DESC`

	rows, err := db.DB.Query(query, requestLimit, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных отчета: %w", err)
	}
	defer rows.Close()

	var reports []BlockReport
	for rows.Next() {
		var report BlockReport
		if err := rows.Scan(&report.IP, &report.Host, &report.RequestCount, &report.LastRequest, &report.FirewallSource); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании данных отчета: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func IPWeightReportHandler(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userID, err := db.GetUserIDByToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
		return
	}

	rows, err := db.DB.Query(`
        SELECT ip, agent_name, weight, last_updated 
        FROM ip_weights 
        WHERE user_id = ? 
        ORDER BY weight DESC
    `, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении отчета"})
		return
	}
	defer rows.Close()

	var report []map[string]interface{}
	for rows.Next() {
		var ip, agentName string
		var weight int
		var lastUpdated time.Time

		if err := rows.Scan(&ip, &agentName, &weight, &lastUpdated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сканировании данных отчета"})
			return
		}

		report = append(report, map[string]interface{}{
			"ip":           ip,
			"agent_name":   agentName,
			"weight":       weight,
			"last_updated": lastUpdated,
		})
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}
