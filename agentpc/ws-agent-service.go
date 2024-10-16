package agentpc

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ResetColor  = "\033[0m"  // Сброс цвета
	PurpleColor = "\033[35m" // Ярко фиолетовый цвет
)

type WebSocketAgent struct {
	conn             *websocket.Conn
	url              string
	token            string
	agentName        string
	lastDisconnected time.Time
}

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

	go agent.Connect()
	return agent
}

func (agent *WebSocketAgent) Connect() {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+agent.token)
	//headers.Add("Origin", "http://localhost") // Проверьте правильность Origin
	retryDelay := 5 * time.Second
	firstConnection := true

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
		if firstConnection {
			log.Println(PurpleColor + "WS-AGENT: Первое установление соединения с WebSocket сервером" + ResetColor)
			firstConnection = false
		} else {
			disconnectedDuration := time.Since(agent.lastDisconnected)
			log.Printf(PurpleColor+"Соединение с WebSocket сервером восстановлено после %v"+ResetColor, disconnectedDuration)
			agent.OnReconnect(disconnectedDuration)
		}
		log.Println(PurpleColor + "WS-AGENT: Подключение к WebSocket серверу установлено" + ResetColor)

		// Отправка имени агента
		log.Printf(PurpleColor+"Отправка имени агента: %s"+ResetColor, agent.agentName)
		err = agent.SendMessage(agent.agentName)
		if err != nil {
			log.Printf("Ошибка при отправке имени агента: %v", err)
			conn.Close()
			continue
		}

		// Прослушивание сообщений
		agent.ReceiveMessages()

		agent.conn.Close()
		agent.lastDisconnected = time.Now()
		log.Println("WS-AGENT: Соединение с WebSocket сервером закрыто. Переподключение через 5 секунд...")
		time.Sleep(5 * time.Second)
	}
}

// OnReconnect вызывается после восстановления соединения
func (agent *WebSocketAgent) OnReconnect(disconnectedDuration time.Duration) {
	log.Printf(PurpleColor+"Метод OnReconnect вызван. Время отсутствия связи: %v"+ResetColor, disconnectedDuration)
}

func (agent *WebSocketAgent) SendMessage(message string) error {
	if agent.conn == nil {
		return fmt.Errorf(PurpleColor + "WS-AGENT: WebSocket соединение не установлено" + ResetColor)
	}
	agent.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := agent.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("WS-AGENT: Ошибка при отправке сообщения: %v", err)
		return err
	}
	return nil
}

func (agent *WebSocketAgent) ReceiveMessages() {
	for {
		messageType, message, err := agent.conn.ReadMessage()
		if err != nil {
			log.Printf("WS-AGENT: Ошибка при получении сообщения: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			log.Printf(PurpleColor+"WS-AGENT: Получено сообщение от сервера: %s"+ResetColor, message)
			/*
				if err := agent.SendMessage(fmt.Sprintf("CONFIRM %s", message)); err != nil {
					log.Printf("WS-AGENT: Ошибка при отправке подтверждения: %v", err)
					break
				} else {

				}
			*/
		} else if messageType == websocket.CloseMessage {
			log.Println("WS-AGENT: Получено сообщение о закрытии соединения")
			break
		}
	}
}
