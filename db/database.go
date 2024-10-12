package db

import (
	"ddos-protection-api/config"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql" // Импорт для MySQL
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"  // Импорт для PostgreSQL
	_ "modernc.org/sqlite" // Импорт для SQLite
)

type Config struct {
	Database struct {
		Type       string `json:"type"`
		Connection string `json:"connection"`
	} `json:"database"`
	Server struct {
		Port    string `json:"port"`
		SSLCert string `json:"ssl_cert"`
		SSLKey  string `json:"ssl_key"`
	} `json:"server"`
}

var DB *sqlx.DB

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

func InitDB(config *config.Config) error {
	var err error

	switch config.Database.Type {
	case "sqlite":
		DB, err = sqlx.Open("sqlite", config.Database.Connection)
	case "postgres":
		DB, err = sqlx.Open("postgres", config.Database.Connection)
	case "mysql":
		DB, err = sqlx.Open("mysql", config.Database.Connection)
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
	          VALUES (?, ?, CURRENT_TIMESTAMP, ?, 1, ?, ?);`

	_, err := DB.Exec(query, userID, ip, requestCount, firewall, port)
	if err != nil {
		return fmt.Errorf("не удалось добавить IP в базу данных: %w", err)
	}
	return nil
}

// Пример функции для создания записи в таблице sessions при аутентификации
func CreateSession(email, token, host string) error {
	query := `INSERT INTO sessions (email, token, created_at, host) VALUES (?, ?, CURRENT_TIMESTAMP, ?)`
	_, err := DB.Exec(query, email, token, host)
	if err != nil {
		return fmt.Errorf("не удалось создать сессию: %v", err)
	}
	return nil
}

// GetUserIDByToken получает userID на основе токена из таблицы sessions
func GetUserIDByToken(token string) (int, error) {
	// Удаление префикса "Bearer " из токена, если он присутствует
	token = strings.TrimPrefix(token, "Bearer ")

	// Получение email из таблицы sessions на основе token
	var email string
	query := `SELECT email FROM sessions WHERE token = ?`
	err := DB.QueryRow(query, token).Scan(&email)
	if err != nil {
		return 0, fmt.Errorf("токен не найден или неактивен: %v", err)
	}

	// Получение userID из таблицы users на основе email
	var userID int
	query = `SELECT id FROM users WHERE email = ?`
	err = DB.QueryRow(query, email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("пользователь не найден: %v", err)
	}

	return userID, nil
}
