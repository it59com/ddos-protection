package services

import (
	"ddos-protection-api/db"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Обработчик блокировки IP-адреса с информацией о порте
func BlockIPHandler(c *gin.Context) {
	ip := c.Param("ip")
	host := c.ClientIP()
	firewall := c.DefaultQuery("firewall", "unknown")
	portStr := c.DefaultQuery("port", "0")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный порт"})
		return
	}

	// Получение user_id на основе токена
	token := c.GetHeader("Authorization")
	userID, err := db.GetUserIDByToken(token)
	if err != nil {
		log.Printf("Ошибка получения пользователя по токену: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
		return
	}

	// Получение имени агента из параметра firewall
	agentName := c.DefaultQuery("firewall", "unknown")

	// Получение текущего количества запросов для этого IP от данного пользователя и агента
	requestCount, err := db.GetRequestCount(userID, ip, host, port)
	if err != nil {
		log.Printf("Ошибка при получении количества запросов: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении количества запросов"})
		return
	}

	// Проверка, если IP ранее был заблокирован, чтобы определить повторную атаку
	isRepeatAttack, err := db.CheckIfRepeatAttack(userID, ip)
	if err != nil {
		log.Printf("Ошибка при проверке повторной атаки: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке повторной атаки"})
		return
	}

	// Передача userID и порта в trackIPRequests для учета пользователя
	blocked, err := trackIPRequests(userID, ip, host, firewall, port)
	if err != nil {
		log.Printf("Ошибка в trackIPRequests: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if blocked {
		// Определяем вес IP с учетом userID и agentName
		weight, err := CalculateWeight(ip, userID, agentName, requestCount, isRepeatAttack)
		if err != nil {
			log.Printf("Ошибка при расчете веса: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("IP %s заблокирован", ip), "weight": weight})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Запрос зафиксирован"})
	}
}

// Новый маршрут для формирования отчета по блокировкам
func BlockReportHandler(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userID, err := db.GetUserIDByToken(token)

	rows, err := db.DB.Query(`
        SELECT ip, host, request_count, last_request, firewall_source 
        FROM requests 
        WHERE request_count >= ? 
		AND user_id = ? 
        ORDER BY last_request DESC
    `, requestLimit, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения отчета"})
		return
	}
	defer rows.Close()

	var report []map[string]interface{}
	for rows.Next() {
		var ip, host, firewallSource string
		var requestCount int
		var lastRequest time.Time

		if err := rows.Scan(&ip, &host, &requestCount, &lastRequest, &firewallSource); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при сканировании данных отчета"})
			return
		}

		report = append(report, map[string]interface{}{
			"ip":              ip,
			"host":            host,
			"request_count":   requestCount,
			"last_request":    lastRequest,
			"firewall_source": firewallSource,
		})
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}
