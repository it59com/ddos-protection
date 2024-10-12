// routes/web_socket_handler.go

package services

import (
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
	blockRules map[string]bool
	mu         sync.Mutex
}

var agents = make(map[int]*AgentConnection)
var agentsMu sync.Mutex

// WebSocketHandler обрабатывает WebSocket-соединение и проверяет Bearer-токен
// WebSocketHandler обрабатывает WebSocket-соединение и проверяет Bearer-токен
func WebSocketHandler(c *gin.Context) {
	// Извлечение токена из заголовка Authorization
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не указан токен"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Проверка токена
	claims, err := GetUserIDByToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный или истекший токен"})
		return
	}

	// Извлекаем userID из claims
	userID := claims

	// Обновление WebSocket соединения
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Ошибка при обновлении до WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Успешное подключение WebSocket для пользователя ID: %d", userID)

	// Сохранение подключения
	agent := &AgentConnection{
		conn:       conn,
		userID:     userID,
		blockRules: make(map[string]bool),
	}
	agentsMu.Lock()
	agents[userID] = agent
	agentsMu.Unlock()

	defer func() {
		agentsMu.Lock()
		delete(agents, userID)
		agentsMu.Unlock()
	}()

	// Обработка сообщений WebSocket
	for {
		// Ожидание сообщений от клиента
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения сообщения: %v", err)
			break
		}

		log.Printf("Получено сообщение от агента пользователя %d: %s", userID, string(message))

		// Ответ на сообщение
		response := []byte("Сообщение получено User_id $userID")
		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			break
		}
	}
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
}
