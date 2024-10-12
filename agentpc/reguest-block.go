package agentpc

import (
	"bytes"
	"fmt"
	"net/http"
)

// Функция для отправки запроса на блокировку IP с указанием порта
func blockIP(ip string, port int, config *AgentConfig) error {
	const blockEndpoint = "/block"
	url := fmt.Sprintf("%s%s/%s?firewall=%s&port=%d", config.ServerURL, blockEndpoint, ip, config.AgentName, port)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Установка заголовка авторизации
	req.Header.Set("Authorization", "Bearer "+config.Token)

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("не удалось заблокировать IP, статус код: %d", resp.StatusCode)
	}

	fmt.Printf("Запрос на блокировку IP %s на порту %d успешно выполнен\n", ip, port)
	return nil
}
