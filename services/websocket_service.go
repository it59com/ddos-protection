package services

import (
	"ddos-protection-api/auth"
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
	blockRules     map[string]bool
	weightSent     map[int]bool
	mu             sync.Mutex
	confirmChannel map[string]chan bool
}

var agents = make(map[int]*AgentConnection)
var agentsMu sync.RWMutex

func (a *AgentConnection) SendAndConfirm(ip, command string) {
	confirmation := make(chan bool, 1)
	a.mu.Lock()
	a.confirmChannel[ip] = confirmation
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.confirmChannel, ip)
		a.mu.Unlock()
	}()

	const maxRetries = 3
	retryCount := 0

	for {
		a.mu.Lock()
		err := a.conn.WriteMessage(websocket.TextMessage, []byte(command))
		a.mu.Unlock()

		if err != nil {
			log.Printf("Ошибка отправки команды для IP %s: %v", ip, err)
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Прекращаем отправку, если соединение закрыто
				return
			}
			retryCount++
			if retryCount >= maxRetries {
				log.Printf("Максимальное количество попыток отправки команды для IP %s достигнуто", ip)
				return
			}
			time.Sleep(5 * time.Second)
			continue
		}

		select {
		case <-confirmation:
			log.Printf("Подтверждение получено для IP %s", ip)
			return
		case <-time.After(10 * time.Second):
			retryCount++
			if retryCount >= maxRetries {
				log.Printf("Повторные попытки отправки команды для IP %s исчерпаны", ip)
				return
			}
			log.Printf("Повторная отправка команды для IP %s (попытка %d)", ip, retryCount+1)
		}
	}
}

func WebSocketHandler(c *gin.Context) {
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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Ошибка при обновлении до WebSocket: %v", err)
		return
	}
	defer conn.Close()

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

	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Ошибка при чтении имени агента: %v", err)
		return
	}
	agentConnection.agentName = string(message)
	log.Printf("Агент %s подключен для пользователя %d", agentConnection.agentName, claims.UserID)

	/*
		if err := sendBlockedIPs(agentConnection); err != nil {
			log.Printf("Ошибка при отправке заблокированных IP агенту %s: %v", agentConnection.agentName, err)
		}
	*/

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Ошибка при чтении сообщения или отключение агента: %v", err)
				break
			}

			parts := strings.Split(string(message), " ")
			if len(parts) == 2 && parts[0] == "CONFIRM" {
				ip := parts[1]
				agentConnection.mu.Lock()
				if ch, exists := agentConnection.confirmChannel[ip]; exists {
					ch <- true
				}
				agentConnection.mu.Unlock()
			}
		}
	}()
}
