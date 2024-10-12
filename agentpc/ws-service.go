package agentpc

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketAgent представляет подключение WebSocket
type WebSocketAgent struct {
	conn *websocket.Conn
}

// NewWebSocketAgent создает новое подключение к WebSocket
func NewWebSocketAgent(url string, token string) (*WebSocketAgent, error) {
	// Определение протокола для WebSocket
	if strings.HasPrefix(url, "https://") {
		url = strings.Replace(url, "https://", "wss://", 1) + "/ws"
	} else if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "ws://", 1) + "/ws"
	} else {
		url = "ws://" + url + "/ws"
	}

	log.Printf("URL для WebSocket: %s", url)
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		return nil, err
	}

	log.Println("Подключение к WebSocket серверу установлено")
	return &WebSocketAgent{conn: conn}, nil
}

// Close закрывает WebSocket соединение
func (agent *WebSocketAgent) Close() {
	if agent.conn != nil {
		agent.conn.Close()
	}
}

// SendMessage отправляет сообщение через WebSocket
func (agent *WebSocketAgent) SendMessage(message string) error {
	err := agent.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("Ошибка при отправке сообщения: %v", err)
		return err
	}
	return nil
}

// ReceiveMessages получает сообщения через WebSocket
func (agent *WebSocketAgent) ReceiveMessages() {
	for {
		_, message, err := agent.conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка при получении сообщения: %v", err)
			break
		}
		log.Printf("Получено сообщение от сервера: %s", message)
	}
}

// WebSocketAgentConnect выполняет подключение к WebSocket-серверу и обработку сообщений
func WebSocketAgentConnect(url string, token string) {
	agent, err := NewWebSocketAgent(url, token)
	if err != nil {
		log.Fatalf("Ошибка подключения к WebSocket серверу: %v", err)
		return
	}
	defer agent.Close()

	// Запуск обработчика входящих сообщений в отдельной горутине
	go agent.ReceiveMessages()

	// Отправка сообщений через интервал
	for {
		err := agent.SendMessage("Сообщение от агента")
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
			break
		}
		time.Sleep(5 * time.Second) // Интервал между отправками сообщений
	}
}
