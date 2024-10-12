package agentpc

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConnect выполняет подключение к WebSocket-серверу
func WebSocketAgentConnect(url string, token string) {
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
		log.Fatalf("Ошибка подключения к WebSocket серверу: %v", err)
		return
	}
	defer conn.Close()

	log.Println("Подключение к WebSocket серверу установлено")

	for {
		err = conn.WriteMessage(websocket.TextMessage, []byte("Сообщение от агента"))
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
			break
		}
		time.Sleep(5 * time.Second) // Примерный интервал между отправками
	}
}
