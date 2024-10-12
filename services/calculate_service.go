package services

import (
	"ddos-protection-api/db"
	"fmt"
	"log"
)

// CalculateWeight - функция для расчета веса IP с учетом userID и agentName
func CalculateWeight(ip string, userID int, agentName string, requestCount int, repeatAttack bool) (int, error) {
	// Начальный вес
	weight := 0

	// Повышение веса за частоту запросов
	if requestCount > 100 {
		weight += 50
	} else if requestCount > 50 {
		weight += 30
	} else {
		weight += 10
	}

	// Увеличение веса, если атака повторяется
	if repeatAttack {
		weight += 30
	}

	// Ограничение максимального веса
	if weight > 100 {
		weight = 100
	}

	// Обновление веса в базе данных
	err := UpdateIPWeight(ip, userID, agentName, weight)
	if err != nil {
		return 0, err
	}

	return weight, nil
}

// UpdateIPWeight - функция для обновления веса IP-адреса в таблице ip_weights
func UpdateIPWeight(ip string, userID int, agentName string, weight int) error {
	query := `INSERT INTO ip_weights (user_id, agent_name, ip, weight, last_updated) 
              VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP);`

	_, err := db.DB.Exec(query, ip, userID, agentName, weight)
	if err != nil {
		return fmt.Errorf("ошибка обновления веса IP: %w", err)
	}

	log.Printf("Вес для IP %s (пользователь %d, агент %s) установлен на %d", ip, userID, agentName, weight)
	return nil
}
