package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Структура Config для загрузки конфигурации
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
	Redis struct {
		Address  string `json:"address"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	} `json:"redis"`
	LogFile string `json:"log_file"`
}

var AppConfig *Config

// LoadConfig загружает конфигурацию из файла и сохраняет в глобальной переменной AppConfig
func LoadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл конфигурации: %w", err)
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("не удалось декодировать конфигурацию: %w", err)
	}

	AppConfig = config
	return nil
}
