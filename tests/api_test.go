package tests

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var agentToken string

const (
	baseURL   = "http://172.18.74.78:8080"
	wsBaseURL = "ws://172.18.74.78:8080/ws" // URL для WebSocket соединения
)

var (
	paramI      = flag.Int("i", 2, "Параметр i для теста")
	numUsers    = flag.Int("users", 1, "Количество пользователей")
	emailPrefix = "testuser"
	password    = "password123testuser"
)

type BlockResponse struct {
	Message     string  `json:"message"`
	TotalWeight float64 `json:"total_weight"`
	UserWeight  float64 `json:"user_weight"`
}

func checkIfUserAuthorized(email, password string, t *testing.T) (bool, string) {
	token := loginUser(email, password, t)
	if token != "" {
		log.Printf("Пользователь %s уже авторизован, пропускаем регистрацию", email)
		return true, token
	}
	return false, ""
}

func registerUser(email, password string, t *testing.T) string {
	authorized, token := checkIfUserAuthorized(email, password, t)
	if authorized {
		return token
	}

	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	return loginUser(email, password, t)
}

func loginUser(email, password string, t *testing.T) string {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Log("Ошибка авторизации:", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Logf("Ошибка авторизации, статус: %d", resp.StatusCode)
		return ""
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Log("Ошибка декодирования ответа авторизации:", err)
		return ""
	}

	token, ok := result["token"].(string)
	if !ok {
		t.Log("Ошибка извлечения токена из ответа")
		return ""
	}

	log.Printf("Токен для %s: %s", email, token)
	return token
}

func testWebSocketConnection(token string, t *testing.T) {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)

	// Подключение к WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsBaseURL, headers)
	assert.NoError(t, err, "Ошибка при подключении к WebSocket")
	defer conn.Close()

	// Отправляем тестовое сообщение
	testMessage := "ping"
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	assert.NoError(t, err, "Ошибка при отправке сообщения через WebSocket")

	// Получаем ответ
	_, message, err := conn.ReadMessage()
	assert.NoError(t, err, "Ошибка при чтении сообщения через WebSocket")
	t.Logf("Ответ от WebSocket сервера: %s", message)
	assert.Equal(t, "server-ok", string(message), "Неверный ответ от WebSocket сервера")
}

func blockRequests(token string, t *testing.T) {
	for i := 1; i <= *paramI; i++ {
		for j := 1; j <= 100; j++ {
			req, err := http.NewRequest("POST", baseURL+"/block/192.168."+strconv.Itoa(i)+"."+strconv.Itoa(i)+"?firewall=Agent-test&port=80", nil)
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var response BlockResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				t.Log("Ошибка декодирования ответа блокировки:", err)
			} else {
				t.Logf("Ответ на запрос блокировки для токена [%s]: %+v", token, response)
			}
		}

		//time.Sleep(time.Duration(rand.Intn(4)+1) * time.Second)
	}
}

func TestParallelUsers(t *testing.T) {
	flag.Parse() // Разбор флагов командной строки

	var userTokens []string

	for i := 1; i <= *numUsers; i++ {
		email := fmt.Sprintf("%s%d@example.com", emailPrefix, i)
		token := registerUser(email, password, t)
		userTokens = append(userTokens, token)
	}

	// Запуск блокировок и проверок WebSocket параллельно
	var wg sync.WaitGroup
	for _, token := range userTokens {
		wg.Add(1)
		go func(tok string) {
			defer wg.Done()
			log.Printf("Start WebSocket test and block request for Token: %s", tok)
			testWebSocketConnection(tok, t) // Тест WebSocket соединения
			blockRequests(tok, t)           // Тест блокировок
		}(token)
	}

	// Ожидаем завершения всех горутин
	wg.Wait()
}
