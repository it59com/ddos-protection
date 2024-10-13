package db

import (
	"ddos-protection-api/config"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Импорт для PostgreSQL
)

type Config struct {
	Database struct {
		Type     string `json:"type"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		DB       string `json:"db"`
		User     string `json:"user"`
		Password string `json:"password"`
		SslMode  string `json:"sslmode"`
	} `json:"database"`
	Server struct {
		Port    string `json:"port"`
		SSLCert string `json:"ssl_cert"`
		SSLKey  string `json:"ssl_key"`
	} `json:"server"`
}

var DB *sqlx.DB

// LoadConfig загружает конфигурацию из JSON файла
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл конфигурации: %w", err)
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("не удалось декодировать конфигурацию: %w", err)
	}

	return config, nil
}

// InitDB инициализирует подключение к базе данных на основе конфигурации
func InitDB(config *config.Config) error {
	var err error
	connStr := ""

	switch config.Database.Type {
	case "postgres":
		connStr = fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
			config.Database.Host,
			config.Database.Port,
			config.Database.DB,
			config.Database.User,
			config.Database.Password,
			config.Database.SslMode,
		)
		DB, err = sqlx.Open("postgres", connStr)
	default:
		return fmt.Errorf("неподдерживаемый тип базы данных: %s", config.Database.Type)
	}

	if err != nil {
		return fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("не удалось пинговать базу данных: %w", err)
	}

	log.Printf("Подключение к базе данных (%s) установлено успешно", config.Database.Type)
	return nil
}

// AddToDatabase добавляет информацию о заблокированном IP-адресе в базу данных
func AddToDatabase(ip, firewall string, requestCount, userID, port int) error {
	query := `INSERT INTO ip_addresses (user_id, ip, blocked_at, request_count, weight, firewall_source, port) 
	          VALUES ($1, $2, CURRENT_TIMESTAMP, $3, 1, $4, $5);`

	_, err := DB.Exec(query, userID, ip, requestCount, firewall, port)
	if err != nil {
		return fmt.Errorf("не удалось добавить IP в базу данных: %w", err)
	}
	return nil
}

// Пример функции для создания записи в таблице sessions при аутентификации
func CreateSession(email, token, host string) error {
	query := `INSERT INTO sessions (email, token, created_at, host) VALUES ($1, $2, CURRENT_TIMESTAMP, $3)`
	_, err := DB.Exec(query, email, token, host)
	if err != nil {
		return fmt.Errorf("не удалось создать сессию: %v", err)
	}
	return nil
}

// GetEmailByToken получает email на основе токена из таблицы sessions
func GetEmailByToken(token string) (string, error) {
	token = strings.TrimPrefix(token, "Bearer ")
	var email string
	query := `SELECT email FROM sessions WHERE token = $1`
	err := DB.QueryRow(query, token).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("токен не найден или неактивен: %v", err)
	}
	return email, nil
}

// GetUserIDByToken получает userID на основе токена из таблицы sessions
func GetUserIDByToken(token string) (int, error) {
	token = strings.TrimPrefix(token, "Bearer ")
	var email string
	query := `SELECT email FROM sessions WHERE token = $1`
	err := DB.QueryRow(query, token).Scan(&email)
	if err != nil {
		return 0, fmt.Errorf("токен не найден или неактивен: %v", err)
	}

	var userID int
	query = `SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`
	err = DB.QueryRow(query, email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден: %v", err)
	}

	return userID, nil
}

// RemoveFromDatabase удаляет запись IP-адреса из таблицы ip_addresses
func RemoveFromDatabase(ip, firewall string, userID, port int) error {
	query := `DELETE FROM ip_addresses WHERE ip = $1 AND firewall_source = $2 AND user_id = $3 AND port = $4`
	_, err := DB.Exec(query, ip, firewall, userID, port)
	if err != nil {
		return fmt.Errorf("ошибка удаления IP из базы данных: %w", err)
	}
	return nil
}

// GetTotalWeightForIP возвращает общий вес для данного IP из таблицы ip_weights
func GetTotalWeightForIP(ip string) (int, error) {
	var totalWeight int
	query := `SELECT AVG(weight) FROM ip_weights WHERE ip = $1`
	err := DB.QueryRow(query, ip).Scan(&totalWeight)
	if err != nil {
		return 0, fmt.Errorf("ошибка при получении общего веса для IP: %w", err)
	}
	return totalWeight, nil
}
