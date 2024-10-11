package main

import (
	"ddos-protection-api/config"
	"ddos-protection-api/db"
	migrations "ddos-protection-api/migrations/sql"
	"log"
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

	// Запускаем миграции
	if err := migrations.RunMigrations(); err != nil {
		log.Fatalf("Ошибка запуска обновления базы данных: %v", err)
	}
	defer db.DB.Close()

	log.Println("Все миграции успешно применены")
}
