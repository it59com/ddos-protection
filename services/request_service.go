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
	queryCheck := `SELECT request_count FROM requests WHERE user_id = ? AND ip = ? AND host = ? AND port = ?`
	err := db.DB.QueryRow(queryCheck, userID, ip, host, port).Scan(&requestCount)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("ошибка проверки записи: %w", err)
	}

	// Если запись существует, обновляем её
	if err == nil {
		queryUpdate := `UPDATE requests SET request_count = request_count + 1, last_request = CURRENT_TIMESTAMP WHERE user_id = ? AND ip = ? AND host = ? AND port = ?`
		_, err := db.DB.Exec(queryUpdate, userID, ip, host, port)
		if err != nil {
			return false, fmt.Errorf("ошибка обновления данных о запросе: %w", err)
		}
		requestCount++ // Увеличиваем для текущего вызова
	} else {
		// Иначе, вставляем новую запись
		queryInsert := `INSERT INTO requests (user_id, ip, host, request_count, last_request, firewall_source, port) 
                        VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, ?, ?)`
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
	weight, err := CalculateWeight(ip, userID, firewall, requestCount, isRepeatAttack)
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
