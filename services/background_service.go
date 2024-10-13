// services/background_service.go
package services

import (
	"ddos-protection-api/db"
	"log"
	"time"
)

// Функция для запуска фоновой службы
func StartBackgroundService() {
	go func() {
		for {
			// Выполняем каждые 5 минут
			time.Sleep(1 * time.Minute)
			log.Println("Запуск фоновой службы для проверки активности IP-адресов...")

			err := reduceInactiveIPWeights()
			if err != nil {
				log.Printf("Ошибка при уменьшении веса неактивных IP: %v", err)
			}
		}
	}()
}

// Функция для уменьшения веса IP-адресов при отсутствии активности
func reduceInactiveIPWeights() error {
	// Время, после которого начинается уменьшение веса (10 минут)
	inactivityThreshold := time.Now().Add(-10 * time.Minute)
	// Время, за которое вес должен быть снижен до 20 (1 час)
	fullReductionTime := time.Hour
	// Минимальный вес, до которого уменьшается значение (20)
	minWeight := 20

	// Выбираем все IP-адреса, которые неактивны дольше 10 минут
	query := `
		SELECT user_id, ip, weight, last_updated
		FROM ip_weights
		WHERE last_updated < $1 AND weight > $2
	`
	rows, err := db.DB.Query(query, inactivityThreshold, minWeight)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Перебираем все найденные записи
	for rows.Next() {
		var userID int
		var ip string
		var weight int
		var lastUpdated time.Time

		err := rows.Scan(&userID, &ip, &weight, &lastUpdated)
		if err != nil {
			log.Printf("Ошибка при обработке строки: %v", err)
			continue
		}

		// Рассчитываем время с момента последнего обновления
		timeSinceLastUpdate := time.Since(lastUpdated)

		// Рассчитываем коэффициент уменьшения веса
		reductionRatio := float64(timeSinceLastUpdate) / float64(fullReductionTime)
		if reductionRatio > 1 {
			reductionRatio = 1 // Ограничение, чтобы не снижалось ниже минимального веса
		}

		// Новый вес, уменьшенный с учетом времени
		newWeight := weight - int(float64(weight-minWeight)*reductionRatio)
		if newWeight < minWeight {
			newWeight = minWeight
		}

		// Обновляем вес IP в базе данных
		updateQuery := `
			UPDATE ip_weights
			SET weight = $1, last_updated = CURRENT_TIMESTAMP
			WHERE user_id = $2 AND ip = $3
		`
		_, err = db.DB.Exec(updateQuery, newWeight, userID, ip)
		if err != nil {
			log.Printf("Ошибка при обновлении веса для IP %s: %v", ip, err)
		} else {
			log.Printf("Вес для IP %s уменьшен до %d", ip, newWeight)
		}
	}

	return nil
}
