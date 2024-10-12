package services

import (
	"ddos-protection-api/auth"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Структура для хранения подключений агентов
type AgentConnection struct {
	conn       *websocket.Conn
	userID     int
	agentName  string
	blockRules map[string]bool
	weightSent map[int]bool // Отслеживание отправленных сообщений по весу
	mu         sync.Mutex
}

var agents = make(map[int]*AgentConnection)
var agentsMu sync.Mutex

// WebSocketHandler обрабатывает WebSocket-соединение и проверяет Bearer-токен
func WebSocketHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не указан токен"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Проверка токена
	claims, err := auth.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный или истекший токен"})
		return
	}

	// Установление WebSocket-соединения
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Ошибка при обновлении до WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Обновление статуса на "online"
	if err := updateSessionStatus(claims.UserID, token, "", "online"); err != nil { // agentName пусто на старте
		log.Printf("Ошибка обновления статуса на 'online': %v", err)
	}

	agentConnection := &AgentConnection{
		conn:       conn,
		userID:     claims.UserID,
		blockRules: make(map[string]bool),
	}

	agentsMu.Lock()
	agents[claims.UserID] = agentConnection
	agentsMu.Unlock()

	// Ожидание первого сообщения с `AgentName`
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Ошибка при чтении имени агента: %v", err)
		return
	}

	agentName := string(message)
	agentConnection.agentName = agentName
	log.Printf("Агент %s подключен для пользователя %d", agentName, claims.UserID)

	// Обновление статуса с учетом `AgentName`
	if err := updateSessionStatus(claims.UserID, token, agentName, "online"); err != nil {
		log.Printf("Ошибка обновления статуса на 'online' с именем агента: %v", err)
	}

	// Обработка сообщений WebSocket
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения сообщения или отключение: %v", err)
			break
		}

		// Обновление метки времени активности
		if err := updateSessionStatus(claims.UserID, token, agentName, "online"); err != nil {
			log.Printf("Ошибка обновления времени активности: %v", err)
		}
	}

	// При разрыве соединения обновляем статус на "offline"
	if err := removeSession(token); err != nil {
		log.Printf("Ошибка обновления статуса на 'offline': %v", err)
	}

	// Удаляем агент при отключении
	agentsMu.Lock()
	delete(agents, claims.UserID)
	agentsMu.Unlock()
}

// SendIPWeightMessage отправляет команду блокировки или разблокировки в зависимости от веса IP
func SendIPWeightMessage(userID int, ip string, weight int, interfaceName string) {
	agentsMu.Lock()
	agent, exists := agents[userID]
	agentsMu.Unlock()
	if !exists {
		log.Printf("Агент для пользователя ID %d не найден", userID)
		return
	}

	agent.mu.Lock()
	defer agent.mu.Unlock()

	// Проверяем блокировку и отправляем команду блокировки или разблокировки
	if weight > 90 && !agent.blockRules[ip] {
		command := "IPTABLES -A INPUT -i " + interfaceName + " -s " + ip + " -j DROP"
		if err := agent.conn.WriteMessage(websocket.TextMessage, []byte(command)); err != nil {
			log.Printf("Ошибка при отправке команды блокировки IP %s: %v", ip, err)
		} else {
			log.Printf("Отправлено правило блокировки для IP %s", ip)
			agent.blockRules[ip] = true
		}
	} else if weight < 60 && agent.blockRules[ip] {
		command := "IPTABLES -D INPUT -i " + interfaceName + " -s " + ip + " -j DROP"
		if err := agent.conn.WriteMessage(websocket.TextMessage, []byte(command)); err != nil {
			log.Printf("Ошибка при отправке команды разблокировки IP %s: %v", ip, err)
		} else {
			log.Printf("Удалено правило блокировки для IP %s", ip)
			delete(agent.blockRules, ip)
		}
	}

	// Проверка отправки сообщений о весе
	if agent.weightSent == nil {
		agent.weightSent = make(map[int]bool)
	}

	// Перебираем веса 10, 20, 30 и т.д. и отправляем сообщения
	for w := 10; w <= weight; w += 10 {
		if !agent.weightSent[w] {
			message := fmt.Sprintf("Для адреса: %s вес стал %d", ip, w)
			sendWsServiceMessage(agent, message)
			agent.weightSent[w] = true
		}
	}
}

func sendWsServiceMessage(agent *AgentConnection, message string) {
	if err := agent.conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		log.Printf("Ошибка при отправке сообщения: %s для user_id: %d, %v", message, agent.userID, err)
	} else {
		log.Printf("Сообщение: %s от пользователя %d", message, agent.userID)
	}
}
