// routes/web_socket_handler.go
package services

import (
	"ddos-protection-api/auth"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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
	userID, err := auth.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный или истекший токен"})
		return
	}

	// Обновление WebSocket соединения
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Ошибка при обновлении до WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Успешное подключение WebSocket для пользователя ID: %d", userID)

	// Обработка сообщений WebSocket
	for {
		// Ожидание сообщений от клиента
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения сообщения: %v", err)
			break
		}

		log.Printf("Получено сообщение от агента пользователя %d: %s", userID, string(message))

		// Здесь можно обработать данные от агента и ответить при необходимости
		response := []byte("Сообщение получено")
		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			break
		}
	}
}
