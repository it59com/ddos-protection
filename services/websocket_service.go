package services

import (
	"ddos-protection-api/auth"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type AgentConnection struct {
	conn           *websocket.Conn
	userID         int
	agentName      string
	mu             sync.Mutex
	blockRules     map[string]bool
	weightSent     map[int]bool
	confirmChannel map[string]chan bool
}

type Notification struct {
	IPAddress     string `json:"ip_address"`
	BlockTime     string `json:"block_time"`
	CurrentWeight int    `json:"weight"`
}

var agents = make(map[int]*AgentConnection)
var agentsMu sync.RWMutex

// Метод для отправки сообщения агенту
func (a *AgentConnection) SendMessage(message string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("Ошибка при отправке сообщения агенту %s: %v", a.agentName, err)
		return err
	}
	log.Printf("Сообщение отправлено агенту %s: %s", a.agentName, message)
	return nil
}

func WebSocketHandler(c *gin.Context) {
	// Аутентификация
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не указан токен"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := auth.ValidateToken(token)
	log.Printf("WebSocket Аутентификация: %v", token)
	if err != nil {
		log.Printf("WebSocket Аутентификация Ошибка: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный или истекший токен"})
		return
	}

	// Обработка WebSocket-соединения
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Ошибка при обновлении до WebSocket: %v", err)
		return
	}
	defer func() {
		log.Println("Закрытие WebSocket соединения")
		updateSessionStatus(claims.UserID, token, claims.Email, "offline")
		conn.Close()
	}()
	// Уникальный ключ для хранения сессии агента
	//key := fmt.Sprintf("%d:%s", claims.UserID, token)

	agentConnection := &AgentConnection{
		conn:       conn,
		userID:     claims.UserID,
		blockRules: make(map[string]bool),
		weightSent: make(map[int]bool),

		confirmChannel: make(map[string]chan bool),
	}

	agentsMu.Lock()
	agents[claims.UserID] = agentConnection
	agentsMu.Unlock()
	defer func() {
		agentsMu.Lock()
		delete(agents, claims.UserID)
		agentsMu.Unlock()
	}()

	// Получение имени агента
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Ошибка при чтении имени агента: %v", err)
		return
	}
	agentConnection.agentName = string(message)
	log.Printf("Агент %s подключен для пользователя %d", agentConnection.agentName, claims.UserID)
	//UpdateAgentSession(claims.UserID, token, agentConnection.agentName)
	updateSessionStatus(claims.UserID, token, agentConnection.agentName, "online")

	// Отправляем приветственное сообщение агенту после подключения
	if err := agentConnection.SendMessage("server-ok"); err != nil {
		log.Printf("Не удалось отправить приветственное сообщение агенту %s", agentConnection.agentName)
	}

	// Слушаем сообщения от агента
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка при чтении сообщения или отключение агента: %v", err)
			break // Выходим из цикла при ошибке (например, отключении клиента)
		}

		// Обрабатываем полученные данные
		log.Printf("Получено сообщение: %s", message)
	}
}

func NotifyAgent(ip string, userID int, weight int) error {
	agentsMu.RLock()
	defer agentsMu.RUnlock()

	agent, exists := agents[userID]
	if !exists {
		return fmt.Errorf("агент не найден для пользователя %d", userID)
	}

	notification := Notification{
		IPAddress:     ip,
		BlockTime:     time.Now().Format(time.RFC3339),
		CurrentWeight: weight,
	}
	message, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("ошибка при кодировании JSON: %v", err)
	}

	err = agent.SendMessage(string(message))
	if err != nil {
		log.Printf("Ошибка при отправке уведомления: %v", err)
		return err
	}

	log.Printf("Отправлено уведомление агенту: %s", message)
	return nil
}

// Обновленный метод для проверки активной сессии перед отправкой уведомления
func NotifyAgentLowWeight(ip string, userID int, weight int) error {
	agentsMu.RLock()
	agent, exists := agents[userID]
	agentsMu.RUnlock()

	if !exists {
		log.Printf("Активный агент не найден для пользователя %d", userID)
		return fmt.Errorf("активный агент не найден")
	}

	if err := agent.SendLowWeightNotification(ip, weight); err != nil {
		log.Printf("Ошибка при отправке уведомления для агента пользователя %d: %v", userID, err)
		return err
	}

	return nil
}

// NotifyAgentWeightDrop - отправляет сообщение о падении веса
func NotifyAgentWeightDrop(ip string, userID int, weight int) {
	agentsMu.RLock()
	defer agentsMu.RUnlock()

	agent, ok := agents[userID]
	if ok && weight <= 20 {
		msg := fmt.Sprintf(`{"type": "weight_drop", "ip": "%s", "weight": %d}`, ip, weight)
		if err := agent.SendMessage(msg); err != nil {
			log.Printf("Ошибка отправки сообщения о снижении веса через WebSocket для пользователя %d: %v", userID, err)
		} else {
			log.Printf("Сообщение о снижении веса отправлено агенту пользователя %d для IP %s", userID, ip)
		}
	}
}

func updateAgentSession(agent *AgentConnection, userID int) {
	agentsMu.Lock()
	defer agentsMu.Unlock()

	agentSession := agents[userID]
	if agentSession != nil {
		// Обновление времени последней активности
		agentSession.blockRules = agent.blockRules
		agentSession.weightSent = agent.weightSent
		agentSession.confirmChannel = agent.confirmChannel
	}
}

// Метод для отправки сообщения об уменьшении веса
func (a *AgentConnection) SendLowWeightNotification(ip string, weight int) error {
	message := map[string]interface{}{
		"type":      "low_weight_warning",
		"ip":        ip,
		"weight":    weight,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка при маршалинге JSON сообщения: %v", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("ошибка при отправке уведомления о низком весе: %v", err)
	}

	log.Printf("Отправлено уведомление о низком весе для IP %s с весом %d", ip, weight)
	return nil
}
