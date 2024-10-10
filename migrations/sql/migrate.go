package migrations

import (
	"ddos-protection-api/db"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

// Получение всех примененных миграций
func getAppliedMigrations() (map[string]bool, error) {
	migrations := make(map[string]bool)

	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.DB.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании таблицы schema_migrations: %w", err)
	}

	rows, err := db.DB.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении версий миграций: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("ошибка при чтении версии миграции: %w", err)
		}
		migrations[version] = true
	}

	return migrations, nil
}

// Применение миграции
func applyMigration(version, content string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %w", err)
	}

	if _, err := tx.Exec(content); err != nil {
		tx.Rollback()
		return fmt.Errorf("ошибка при применении миграции %s: %w", version, err)
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("ошибка при записи версии миграции %s: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка при коммите миграции %s: %w", version, err)
	}

	log.Printf("Миграция %s успешно применена", version)
	return nil
}

// Запуск миграций
func RunMigrations() error {
	appliedMigrations, err := getAppliedMigrations()
	if err != nil {
		return err
	}

	files, err := filepath.Glob("migrations/sql/*.sql")
	if err != nil {
		return fmt.Errorf("ошибка при получении списка файлов миграций: %w", err)
	}

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".sql")
		if appliedMigrations[version] {
			log.Printf("Миграция %s уже применена, пропуск", version)
			continue
		}

		content, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("ошибка при чтении файла миграции %s: %w", version, err)
		}

		if err := applyMigration(version, string(content)); err != nil {
			return err
		}
	}

	return nil
}
