package main

import (
	"ddos-protection-api/config"
	"ddos-protection-api/db"
	"fmt"
	"log"
)

func main_clean() {
	// Загружаем конфигурацию
	// Загружаем конфигурацию
	if err := config.LoadConfig("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем базу данных
	if err := db.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer db.DB.Close()

	// Удаление всех таблиц из базы данных
	if err := dropAllTables(); err != nil {
		log.Fatalf("Ошибка при удалении таблиц: %v", err)
	}

	log.Println("Все таблицы успешно удалены из базы данных")
}

// dropAllTables удаляет все таблицы из базы данных
func dropAllTables() error {
	// Запрашиваем список всех таблиц
	rows, err := db.DB.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return fmt.Errorf("ошибка при получении списка таблиц: %w", err)
	}
	defer rows.Close()

	// Удаляем каждую таблицу
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("ошибка при чтении имени таблицы: %w", err)
		}

		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)
		_, err := db.DB.Exec(dropQuery)
		if err != nil {
			return fmt.Errorf("ошибка при удалении таблицы %s: %w", tableName, err)
		}

		log.Printf("Таблица %s успешно удалена", tableName)
	}

	return nil
}
