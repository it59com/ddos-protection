package agentpc

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketAgent представляет подключение WebSocket
type WebSocketAgent struct {
	conn      *websocket.Conn
	url       string
	token     string
	agentName string
}

// NewWebSocketAgent создает новое подключение WebSocketAgent
func NewWebSocketAgent(url, token, agentName string) *WebSocketAgent {
	if strings.HasPrefix(url, "https://") {
		url = strings.Replace(url, "https://", "wss://", 1) + "/ws"
	} else if strings.HasPrefix(url, "http://") {
		url = strings.Replace(url, "http://", "ws://", 1) + "/ws"
	} else {
		url = "ws://" + url + "/ws"
	}

	log.Printf("URL для WebSocket: %s", url)

	agent := &WebSocketAgent{
		url:       url,
		token:     token,
		agentName: agentName,
	}

	go agent.Connect() // Запускаем соединение в отдельной горутине
	return agent
}

// Connect устанавливает подключение к WebSocket-серверу с автоматическим переподключением
func (agent *WebSocketAgent) Connect() {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+agent.token)

	retryDelay := 5 * time.Second

	for {
		conn, _, err := websocket.DefaultDialer.Dial(agent.url, headers)
		if err != nil {
			log.Printf("Ошибка подключения к WebSocket серверу: %v. Повторная попытка через %v...", err, retryDelay)
			time.Sleep(retryDelay)
			if retryDelay < 60*time.Second {
				retryDelay *= 2
			}
			continue
		}

		agent.conn = conn
		retryDelay = 5 * time.Second
		log.Println("Подключение к WebSocket серверу установлено")

		// Установка тайм-аутов
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		/*

			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(120 * time.Second))
				return nil
			})
		*/

		// Отправка имени агента
		log.Printf("Отправка имени агента: %s", agent.agentName)
		err = agent.SendMessage(agent.agentName)
		if err != nil {
			log.Printf("Ошибка при отправке имени агента: %v", err)
			conn.Close()
			continue
		}

		// Прослушивание сообщений
		go agent.ReceiveMessages()

		// Ожидание завершения соединения перед переподключением
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Соединение закрыто: %v", err)
				break
			}
		}

		conn.Close()
		log.Println("Соединение с WebSocket сервером закрыто. Переподключение через 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}

// Close закрывает WebSocket соединение
func (agent *WebSocketAgent) Close() {
	if agent.conn != nil {
		agent.conn.Close()
	}
}

// SendMessage отправляет сообщение через WebSocket
func (agent *WebSocketAgent) SendMessage(message string) error {
	if agent.conn == nil {
		return fmt.Errorf("WebSocket соединение не установлено")
	}
	agent.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)) // Тайм-аут на запись
	err := agent.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("WriteMessage: Ошибка при отправке сообщения: %v", err)
		return err
	}
	return nil
}

// ReceiveMessages получает сообщения через WebSocket
func (agent *WebSocketAgent) ReceiveMessages() {
	for {
		//agent.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		messageType, message, err := agent.conn.ReadMessage()
		if err != nil {

			log.Printf("ReceiveMessages: Ошибка при получении сообщения: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			log.Printf("Получено сообщение от сервера: %s", message)
			if err := agent.SendMessage(fmt.Sprintf("CONFIRM %s", message)); err != nil {
				log.Printf("Ошибка при отправке подтверждения: %v", err)
				break
			}
		} else if messageType == websocket.CloseMessage {
			log.Println("Получено сообщение о закрытии соединения")
			break
		}
	}
}

// Отправка ping-сообщений
func (agent *WebSocketAgent) sendPingMessages() {
	for {
		time.Sleep(50 * time.Second)
		if agent.conn == nil {
			return
		}
		if err := agent.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("Ошибка при отправке ping: %v", err)
			return
		}
	}
}
