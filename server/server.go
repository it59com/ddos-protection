package main

import (
	"ddos-protection-api/config"
	"ddos-protection-api/db"
	"ddos-protection-api/routes"
	"ddos-protection-api/services"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func setupLogging(logFile string) {
	// Настройка вывода логов в файл, если указан путь к файлу логов
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Не удалось открыть файл для логирования: %v", err)
		}
		log.SetOutput(file)
	} else {
		// Если файл не указан, вывод в стандартный вывод
		log.SetOutput(os.Stdout)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	// Загружаем конфигурацию
	if err := config.LoadConfig("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Настройка логирования
	setupLogging(config.AppConfig.LogFile)

	// Инициализируем базу данных
	if err := db.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer func() {
		log.Println("Закрытие базы данных")
		db.DB.Close()
	}()

	// Инициализируем Redis
	log.Println("Инициализация Redis")
	services.InitRedis()

	// Запуск фоновой службы для снижения веса IP-адресов
	log.Println("Запуск фоновой службы для снижения веса IP-адресов")
	services.StartBackgroundService()

	// Настройка роутов Gin
	log.Println("Настройка маршрутов")
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
