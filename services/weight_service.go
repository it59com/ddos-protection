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
		weight += 5
	} else if requestCount > 50 {
		weight += 3
	} else {
		weight += 1
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
	err := UpsertIPWeight(ip, userID, agentName, weight)
	if err != nil {
		return 0, err
	}

	return weight, nil
}

// UpsertIPWeight - функция для добавления или обновления веса IP-адреса
func UpsertIPWeight(ip string, userID int, agentName string, increment int) error {
	var currentWeight int

	// Проверяем текущий вес IP в таблице
	query := `SELECT weight FROM ip_weights WHERE user_id = $1 AND ip = $2`
	err := db.DB.QueryRow(query, userID, ip).Scan(&currentWeight)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если IP не найден, добавляем новую запись
			query = `INSERT INTO ip_weights (user_id, agent_name, ip, weight, last_updated) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)`
			_, err = db.DB.Exec(query, userID, agentName, ip, increment)
			if err != nil {
				return fmt.Errorf("UpsertIPWeight: ошибка при добавлении нового IP в ip_weights: %v", err)
			}
			log.Printf("UpsertIPWeight: Добавлен новый IP %s с весом %d для пользователя %d", ip, increment, userID)
			return nil
		}
		return fmt.Errorf("UpsertIPWeight: ошибка при проверке веса IP: %v", err)
	}

	// Если IP уже существует, увеличиваем вес
	newWeight := currentWeight + increment
	if newWeight > 100 {
		newWeight = 100
	}
	query = `UPDATE ip_weights SET weight = $1, last_updated = CURRENT_TIMESTAMP WHERE ip = $2 AND user_id = $3`
	_, err = db.DB.Exec(query, newWeight, ip, userID)
	if err != nil {
		return fmt.Errorf("UpsertIPWeight: ошибка при обновлении веса IP: %v", err)
	}
	log.Printf("UpsertIPWeight: Обновлен IP %s с новым весом %d для пользователя %d", ip, newWeight, userID)
	return nil
}

// UpdateTotalWeight - функция для обновления общего веса IP-адреса для всех пользователей
func UpdateTotalWeight(ip string, increment int) error {
	var currentTotalWeight int

	// Проверяем текущий общий вес IP в таблице
	query := `SELECT weight FROM total_weights WHERE ip = ?  AND deleted_at IS NULL`
	err := db.DB.QueryRow(query, ip).Scan(&currentTotalWeight)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если IP не найден, добавляем новую запись
			query = `INSERT INTO total_weights (ip, weight, last_updated) VALUES ($1, $2, CURRENT_TIMESTAMP)`
			_, err = db.DB.Exec(query, ip, increment)
			if err != nil {
				return fmt.Errorf("UpdateTotalWeight: ошибка при добавлении нового IP в total_weights: %v", err)
			}
			log.Printf("UpdateTotalWeight: Добавлен новый IP %s с общим весом %d", ip, increment)
			return nil
		}
		return fmt.Errorf("UpdateTotalWeight: ошибка при проверке общего веса IP: %v", err)
	}

	// Если IP уже существует, увеличиваем общий вес
	newTotalWeight := currentTotalWeight + increment
	if newTotalWeight > 100 {
		newTotalWeight = 100
	}
	query = `UPDATE total_weights SET weight = $1, last_updated = CURRENT_TIMESTAMP WHERE ip = $2`
	_, err = db.DB.Exec(query, newTotalWeight, ip)
	if err != nil {
		return fmt.Errorf("UpdateTotalWeight: ошибка при обновлении общего веса IP: %v", err)
	}
	log.Printf("UpdateTotalWeight: Обновлен общий вес IP %s с новым значением %d", ip, newTotalWeight)
	return nil
}

// AddOrUpdateIPWeight - общий метод для обновления user_weight и total_weight
func AddOrUpdateIPWeight(ip string, userID int, agentName string, increment int) error {
	err := UpsertIPWeight(ip, userID, agentName, increment)
	if err != nil {
		return err
	}
	return UpdateTotalWeight(ip, increment)
}

// GetTotalWeightForIP возвращает общий вес для данного IP из таблицы ip_weights
func GetTotalWeightForIP(ip string) (float64, error) {
	var totalWeight float64
	// Исправьте тип данных на float64, так как результат AVG() будет в формате decimal/float
	query := `SELECT AVG(weight) AS avg FROM ip_weights WHERE ip = $1`
	err := db.DB.QueryRow(query, ip).Scan(&totalWeight)
	if err != nil {
		return 0, fmt.Errorf("GetTotalWeightForIP: ошибка при получении общего веса для IP: %w", err)
	}
	return totalWeight, nil
}

// Получение веса IP для пользователя
func CheckIPWeight(ip string, userID int) (float64, error) {
	var weight float64
	query := `SELECT weight FROM ip_weights WHERE user_id = $1 AND ip = $2`
	err := db.DB.QueryRow(query, userID, ip).Scan(&weight)
	if err != nil {
		return 0, fmt.Errorf("CheckIPWeight: ошибка при проверке веса IP: %w", err)
	}
	return weight, nil
}
