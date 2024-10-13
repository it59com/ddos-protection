package tests

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

var agentToken string

const (
	baseURL = "http://172.18.74.78:8080"
)

// Параметры для тестов
var (
	paramI      = flag.Int("i", 2, "Параметр i для теста")
	numUsers    = flag.Int("users", 1, "Количество пользователей")
	emailPrefix = "testuser"
	password    = "password123testuser"
)

func registerUser(email, password string, t *testing.T) string {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	return email
}

func loginUser(email, password string, t *testing.T) string {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	token := result["token"]
	log.Printf("Токен для %s: %s", email, token)
	assert.NotEmpty(t, token)
	return token
}

func blockRequests(token string, t *testing.T) {
	for i := 1; i <= *paramI; i++ {
		for j := 1; j <= 2; j++ {
			req, err := http.NewRequest("POST", baseURL+"/block/192.168."+strconv.Itoa(i)+"."+strconv.Itoa(j)+"?firewall=Agent-test&port=80", nil)
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, _ := ioutil.ReadAll(resp.Body)
			t.Logf("Ответ на запрос пользователя [%s]: %s", token, string(body))
			resp.Body.Close()
		}

		time.Sleep(time.Duration(rand.Intn(4)+1) * time.Second)
	}
}

// Функция для удаления всех пользователей
/*
func deleteUser(t *testing.T, token string) {
	deleteURL := baseURL + "/user/delete"
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Читаем и анализируем ответ
	body, _ := ioutil.ReadAll(resp.Body)
	t.Logf("Ответ на запрос удаления пользователя : %s", string(body))
	resp.Body.Close()
}
*/

func TestParallelUsers(t *testing.T) {
	flag.Parse() // Разбор флагов командной строки

	var userTokens []string

	for i := 1; i <= *numUsers; i++ {
		email := fmt.Sprintf("%s%d@example.com", emailPrefix, i)
		registerUser(email, password, t)
	}

	for i := 1; i <= *numUsers; i++ {
		email := fmt.Sprintf("%s%d@example.com", emailPrefix, i)
		log.Printf("email: %s, password %s", email, password)
		token := loginUser(email, password, t)
		userTokens = append(userTokens, token)
		//time.Sleep(500 * time.Millisecond)
	}

	// Запуск блокировок параллельно
	var wg sync.WaitGroup
	for _, token := range userTokens {
		wg.Add(1)
		go func(tok string) {
			defer wg.Done()
			log.Printf("Start block request Token: %s", tok)
			blockRequests(tok, t)
		}(token)
		//time.Sleep(100 * time.Millisecond)
	}

	// Ожидаем завершения всех горутин
	wg.Wait()

	/*
		for _, token := range userTokens {
			go deleteUser(t, token)
			time.Sleep(1 * time.Second)
		}
	*/

}
