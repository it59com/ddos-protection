// services/request_service.go
package services

import (
	"context"
	"ddos-protection-api/db"
	"fmt"
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

// TrackIPRequests обрабатывает запросы и блокирует IP при превышении лимита
func TrackIPRequests(userID int, ip, host, firewall string, port int) (bool, error) {
	// SQL-запрос для вставки или обновления записи в таблице requests с учетом userID и порта
	query := `INSERT INTO requests (user_id, ip, host, request_count, last_request, firewall_source, port) 
              VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, ?, ?)
              ON CONFLICT(user_id, ip, host, port) DO UPDATE SET 
              request_count = request_count + 1,
              last_request = CURRENT_TIMESTAMP;`

	_, err := db.DB.Exec(query, userID, ip, host, firewall, port)
	if err != nil {
		return false, fmt.Errorf("ошибка обновления данных о запросе: %w", err)
	}

	// Учет лимита запросов с использованием Redis
	key := fmt.Sprintf("req_count:%d:%s:%d", userID, ip, port)
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("ошибка инкремента счетчика в Redis: %w", err)
	}

	// Устанавливаем TTL для ключа, если это первый запрос
	if count == 1 {
		err = rdb.Expire(ctx, key, timeWindow).Err()
		if err != nil {
			return false, fmt.Errorf("ошибка установки TTL в Redis: %w", err)
		}
	}

	// Проверка лимита запросов
	if count > requestLimit {
		// Добавляем заблокированный IP в таблицу ip_addresses
		err := db.AddToDatabase(ip, firewall, int(count), userID, port)
		if err != nil {
			return false, fmt.Errorf("ошибка добавления IP в базу данных: %w", err)
		}
		return true, nil
	}

	return false, nil
}
