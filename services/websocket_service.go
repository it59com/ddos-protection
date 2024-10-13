package services

import (
	"ddos-protection-api/auth"
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

type AgentConnection struct {
	conn           *websocket.Conn
	userID         int
	agentName      string
	mu             sync.Mutex
	blockRules     map[string]bool
	weightSent     map[int]bool
	confirmChannel map[string]chan bool
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
	if err != nil {
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
		conn.Close()
	}()

	agentConnection := &AgentConnection{
		conn:           conn,
		userID:         claims.UserID,
		blockRules:     make(map[string]bool),
		weightSent:     make(map[int]bool),
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

	// Отправляем приветственное сообщение агенту после подключения
	if err := agentConnection.SendMessage("Привет от сервера!"); err != nil {
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
