package services

import (
	"database/sql"
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
	err := AddOrUpdateIPWeight(ip, userID, agentName, weight)
	if err != nil {
		return 0, err
	}

	return weight, nil
}

// AddOrUpdateIPWeight - функция для обновления или добавления веса IP-адреса в таблице ip_weights
func AddOrUpdateIPWeight(ip string, userID int, agentName string, weight int) error {
	var currentWeight int

	// Проверяем текущий вес IP в таблице
	query := `SELECT weight FROM ip_weights WHERE ip = ? AND user_id = ?`
	err := db.DB.QueryRow(query, ip, userID).Scan(&currentWeight)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если IP не найден, добавляем новую запись
			query = `INSERT INTO ip_weights (user_id, agent_name, ip, weight, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`
			_, err = db.DB.Exec(query, userID, agentName, ip, weight)
			if err != nil {
				return fmt.Errorf("ошибка при добавлении нового IP в ip_weights: %v", err)
			}
			log.Printf("Добавлен новый IP %s с весом %d", ip, weight)
			return nil
		}
		return fmt.Errorf("ошибка при проверке веса IP: %v", err)
	}

	// Если IP уже существует, обновляем вес
	newWeight := currentWeight + weight
	if newWeight > 100 {
		newWeight = 100
	}
	query = `UPDATE ip_weights SET weight = ?, last_updated = CURRENT_TIMESTAMP WHERE ip = ? AND user_id = ?`
	_, err = db.DB.Exec(query, newWeight, ip, userID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении веса IP: %v", err)
	}
	log.Printf("Обновлен IP %s с новым весом %d", ip, newWeight)
	return nil
}
