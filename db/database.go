package db

import (
	"ddos-protection-api/config"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

// AddToDatabase добавляет информацию о заблокированном IP в базу данных
func AddToDatabase(ip string, firewall string, requestCount int) error {
	query := `INSERT INTO ip_addresses (ip, blocked_at, request_count, weight, firewall_source) 
              VALUES (?, CURRENT_TIMESTAMP, ?, 1, ?);`
	_, err := DB.Exec(query, ip, requestCount, firewall)
	if err != nil {
		return fmt.Errorf("не удалось добавить IP в базу данных: %w", err)
	}
	return nil
}
