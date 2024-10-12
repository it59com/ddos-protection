package main

import (
	"ddos-protection-api/config"
	"ddos-protection-api/db"
	"ddos-protection-api/routes"
	"ddos-protection-api/services"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// Загружаем конфигурацию
	if err := config.LoadConfig("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем базу данных
	if err := db.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer db.DB.Close()

	// Инициализируем Redis
	services.InitRedis()

	// Настройка роутов Gin
	r := gin.Default()
	routes.InitRoutes(r)

	// Проверка наличия SSL сертификатов и запуск сервера
	port := config.AppConfig.Server.Port
	if config.AppConfig.Server.SSLCert != "" && config.AppConfig.Server.SSLKey != "" {
		log.Printf("Запуск сервера на порту %s с поддержкой SSL", port)
		if err := r.RunTLS(":"+port, config.AppConfig.Server.SSLCert, config.AppConfig.Server.SSLKey); err != nil {
			log.Fatalf("Ошибка запуска сервера с SSL: %v", err)
		}
	} else {
		log.Printf("Запуск сервера на порту %s без SSL", port)
		if err := r.Run(":" + port); err != nil {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}
}
