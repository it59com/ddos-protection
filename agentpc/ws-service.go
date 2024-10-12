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

// WebSocketAgentConnect выполняет подключение к WebSocket-серверу с переподключением и отправкой AgentName
func WebSocketAgentConnect(url, token, agentName string) {
	// Определение протокола для WebSocket
	if strings.HasPrefix(url, "https://") {
		url = strings.Replace(url, "https://", "wss://", 1) + "/ws"
	} else if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "ws://", 1) + "/ws"
	} else {
		url = "ws://" + url + "/ws"
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)

	for {
		conn, _, err := websocket.DefaultDialer.Dial(url, headers)
		if err != nil {
			log.Printf("Ошибка подключения к WebSocket серверу: %v. Повторная попытка через 5 секунд...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Подключение к WebSocket серверу установлено")

		// Отправка `AgentName` сразу после подключения
		err = conn.WriteMessage(websocket.TextMessage, []byte(agentName))
		if err != nil {
			log.Printf("Ошибка при отправке имени агента: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Обработка сообщений WebSocket
		for {
			err := conn.WriteMessage(websocket.TextMessage, []byte("Сообщение от агента"))
			if err != nil {
				log.Printf("Ошибка при отправке сообщения: %v", err)
				break
			}

			// Чтение ответа от сервера
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Ошибка при получении сообщения: %v", err)
				break
			}
			log.Printf("Ответ от сервера: %s", message)

			// Пример задержки между сообщениями
			time.Sleep(5 * time.Second)
		}

		// Закрытие соединения перед переподключением
		conn.Close()
		log.Println("Соединение с WebSocket сервером закрыто. Переподключение через 5 секунд...")
		time.Sleep(5 * time.Second) // Ждем перед переподключением
	}
}
