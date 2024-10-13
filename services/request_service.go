// services/request_service.go
package services

import (
	"context"
	"database/sql"
	"ddos-protection-api/db"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

const (
	requestLimit = 10               // Лимит запросов от одного IP
	timeWindow   = 60 * time.Second // Временное окно в секундах
)

func trackIPRequests(userID int, ip, host, firewall string, port int) (bool, error) {
	// Сначала проверяем, существует ли запись
	var requestCount int
	queryCheck := `SELECT request_count FROM requests WHERE user_id = $1 AND ip = $2 AND host = $3 AND port = $4 AND deleted_at IS NULL`
	err := db.DB.QueryRow(queryCheck, userID, ip, host, port).Scan(&requestCount)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("ошибка проверки записи: %w", err)
	}

	// Если запись существует, обновляем её
	if err == nil {
		queryUpdate := `UPDATE requests SET request_count = request_count + 1, last_request = CURRENT_TIMESTAMP WHERE user_id = $1 AND ip = $2 AND host = $3 AND port = $4`
		_, err := db.DB.Exec(queryUpdate, userID, ip, host, port)
		if err != nil {
			return false, fmt.Errorf("ошибка обновления данных о запросе: %w", err)
		}
		requestCount++ // Увеличиваем для текущего вызова
	} else {
		// Иначе, вставляем новую запись
		queryInsert := `INSERT INTO requests (user_id, ip, host, request_count, last_request, firewall_source, port) 
                        VALUES ($1, $2, $3, 1, CURRENT_TIMESTAMP, $4, $5)`
		_, err := db.DB.Exec(queryInsert, userID, ip, host, firewall, port)
		if err != nil {
			return false, fmt.Errorf("ошибка вставки новой записи: %w", err)
		}
		requestCount = 1
	}

	// Проверка, если IP ранее был заблокирован, чтобы определить повторную атаку
	isRepeatAttack, err := db.CheckIfRepeatAttack(userID, ip)
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке повторной атаки: %w", err)
	}

	// Определение веса IP
	weight, err := CalculateWeightRequest(ip, userID, firewall, requestCount, isRepeatAttack)
	if err != nil {
		return false, fmt.Errorf("ошибка при расчете веса: %w", err)
	}

	// Проверка лимита веса для блокировки
	if weight >= 90 {
		err := db.AddToDatabase(ip, firewall, requestCount, userID, port)
		if err != nil {
			return false, fmt.Errorf("ошибка добавления IP в базу данных: %w", err)
		}
		log.Printf("IP %s заблокирован, текущий вес: %d", ip, weight)
		return true, nil
	} else if weight < 60 {
		err := db.RemoveFromDatabase(ip, firewall, userID, port)
		if err != nil {
			return false, fmt.Errorf("ошибка удаления IP из базы данных: %w", err)
		}
		log.Printf("IP %s разблокирован, текущий вес: %d", ip, weight)
	}

	return false, nil
}

// AddToDatabase добавляет информацию о заблокированном IP-адресе в базу данных
func AddToDatabase(ip, firewall string, requestCount, userID, port int) error {
	query := `INSERT INTO ip_addresses (user_id, ip, blocked_at, request_count, weight, firewall_source, port) 
	          VALUES ($1, $2, CURRENT_TIMESTAMP, $3, 1, $4, $5)`
	_, err := db.DB.Exec(query, userID, ip, requestCount, firewall, port)
	if err != nil {
		return fmt.Errorf("ошибка добавления IP в базу данных: %w", err)
	}
	return nil
}

// RemoveFromDatabase удаляет запись IP-адреса из таблицы ip_addresses
func RemoveFromDatabase(ip, firewall string, userID, port int) error {
	query := `DELETE FROM ip_addresses WHERE ip = $1 AND firewall_source = $2 AND user_id = $3 AND port = $4`
	_, err := db.DB.Exec(query, ip, firewall, userID, port)
	if err != nil {
		return fmt.Errorf("ошибка удаления IP из базы данных: %w", err)
	}
	return nil
}

// CheckIfRepeatAttack проверяет, была ли повторная атака с этого IP
func CheckIfRepeatAttack(userID int, ip string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM ip_addresses WHERE user_id = $1 AND ip = $2`
	err := db.DB.QueryRow(query, userID, ip).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке повторной атаки: %w", err)
	}
	return count > 0, nil
}

// CalculateWeight определяет вес IP-адреса
func CalculateWeightRequest(ip string, userID int, firewall string, requestCount int, repeatAttack bool) (int, error) {
	weight := requestCount // Пример, измените логику расчета по необходимости

	if repeatAttack {
		weight += 30
	}

	if weight > 100 {
		weight = 100
	}

	query := `INSERT INTO ip_weights (user_id, agent_name, ip, weight, last_updated) 
	          VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	          ON CONFLICT (user_id, ip) DO UPDATE SET weight = EXCLUDED.weight, last_updated = CURRENT_TIMESTAMP`
	_, err := db.DB.Exec(query, userID, firewall, ip, weight)
	if err != nil {
		return 0, fmt.Errorf("ошибка при обновлении веса IP: %w", err)
	}
	return weight, nil
}
