// services/background_service.go
package services

import (
	"ddos-protection-api/db"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Функция для запуска фоновой службы
func StartBackgroundService() {
	go func() {
		for {
			// Выполняем каждые 5 минут
			time.Sleep(30 * time.Second)
			log.Println("Запуск фоновой службы для проверки активности IP-адресов...")

			err := reduceInactiveIPWeights()
			if err != nil {
				log.Printf("Ошибка при уменьшении веса неактивных IP: %v", err)
			}

			checkActiveWebSocketSessions()
		}
	}()
}

func checkActiveWebSocketSessions() {
	agentsMu.RLock()
	defer agentsMu.RUnlock()

	for userID, agentConn := range agents {
		err := agentConn.conn.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			log.Printf("Проблема с WebSocket соединением для пользователя %d: %v. Закрытие соединения...", userID, err)
			// Закрываем соединение и удаляем агента, так как он неактивен
			agentConn.conn.Close()

			agentsMu.Lock()
			delete(agents, userID)
			agentsMu.Unlock()

			log.Printf("Соединение для пользователя %d закрыто и удалено.", userID)
		} else {
			log.Printf("WebSocket соединение для пользователя %d активно.", userID)
		}
	}
}

// contains проверяет, содержится ли элемент в массиве
func contains(arr []int, item int) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

// Функция для уменьшения веса IP-адресов при отсутствии активности
func reduceInactiveIPWeights() error {
	// Время, после которого начинается уменьшение веса (3 минуты)
	inactivityThreshold := time.Now().Add(-3 * time.Minute)
	// Полное время уменьшения веса (1 час)
	const fullReductionTime = 1 * time.Hour
	// Шаг уменьшения веса
	const reductionStep = 10
	const minWeight = 10
	const maxWeight = 100

	// Выбираем все IP-адреса, которые неактивны дольше 3 минут и имеют вес больше минимального
	query := `
		SELECT user_id, ip, weight, last_updated, low_weight_notified
		FROM ip_weights
		WHERE last_updated < $1 AND weight > $2
	`
	rows, err := db.DB.Query(query, inactivityThreshold, minWeight)
	if err != nil {
		return err
	}
	defer rows.Close()
	rowCount := 0 // Счетчик строк

	activeSessions, err := GetActiveSessions()
	if err != nil {
		log.Printf("Ошибка при получении активных сессий: %v", err)
		activeSessions = []map[string]interface{}{} // Если ошибка, работаем с пустым списком
	}

	// Перебираем все найденные записи
	for rows.Next() {
		var userID int
		var ip string
		var weight int
		var lastUpdated time.Time
		var lowWeightNotified bool

		// Сканируем строку и проверяем ошибки
		err := rows.Scan(&userID, &ip, &weight, &lastUpdated, &lowWeightNotified)
		if err != nil {
			log.Printf("Ошибка при обработке строки: %v", err)
			continue
		}

		// Рассчитываем время с момента последнего обновления
		timeSinceLastUpdate := time.Since(lastUpdated)
		// Расчет количества шагов уменьшения, в зависимости от времени без активности
		reductionFactor := int(timeSinceLastUpdate / fullReductionTime * reductionStep)
		if reductionFactor < 1 {
			reductionFactor = 1
		}

		// Новый вес с учетом постепенного уменьшения
		newWeight := weight - reductionFactor
		if newWeight < minWeight {
			newWeight = minWeight
		}

		// Проверка на наличие активных сессий и отправка уведомления об уменьшении веса
		if newWeight <= 20 && !lowWeightNotified {
			if containsSession(activeSessions, userID) && newWeight == 20 {
				err := NotifyAgentLowWeight(ip, userID, newWeight)

				if err != nil {
					log.Printf("Ошибка при отправке уведомления об уменьшении веса: %v", err)
				} else {
					// Устанавливаем флаг отправки уведомления
					lowWeightNotified = true
				}
			}
		}

		// Обновляем вес IP и статус уведомления о низком весе в базе данных
		updateQuery := `
			UPDATE ip_weights
			SET weight = $1, last_updated = CURRENT_TIMESTAMP, low_weight_notified = $2
			WHERE user_id = $3 AND ip = $4
		`
		_, err = db.DB.Exec(updateQuery, newWeight, lowWeightNotified, userID, ip)
		if err != nil {
			log.Printf("Ошибка при обновлении веса %d для IP %s: %v", newWeight, ip, err)
		} else {
			log.Printf("Вес для IP %s уменьшен до %d", ip, newWeight)
		}

		rowCount++
	}

	log.Printf("Обработано строк: %d", rowCount)
	return nil
}

// Функция проверки наличия активной сессии
func containsSession(sessions []map[string]interface{}, userID int) bool {
	for _, session := range sessions {
		if sessionUserID, ok := session["user_id"].(int); ok && sessionUserID == userID {
			return true
		}
	}
	return false
}
