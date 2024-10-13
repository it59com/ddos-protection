package db

import (
	"database/sql"
	"fmt"
)

// CheckIfRepeatAttack проверяет, был ли IP ранее заблокирован для данного пользователя
func CheckIfRepeatAttack(userID int, ip string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM ip_addresses WHERE user_id = $1 AND ip = $2`
	err := DB.QueryRow(query, userID, ip).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке повторной атаки: %v", err)
	}
	return count > 0, nil
}

// GetRequestCount возвращает количество запросов от определенного IP и порта для указанного пользователя
func GetRequestCount(userID int, ip, host string, port int) (int, error) {
	var requestCount int
	query := `SELECT request_count FROM requests WHERE user_id = $1 AND ip = $2 AND host = $3 AND port = $4`
	err := DB.QueryRow(query, userID, ip, host, port).Scan(&requestCount)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если запись не найдена, возвращаем 0 и без ошибки
			return 0, nil
		}
		return 0, fmt.Errorf("ошибка получения количества запросов: %v", err)
	}
	return requestCount, nil
}
