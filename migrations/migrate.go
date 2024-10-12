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
	// Проверяем начальную миграцию
	if err := ensureInitialMigration(); err != nil {
		return err
	}

	// Получаем список уже примененных миграций
	appliedMigrations, err := getAppliedMigrations()
	if err != nil {
		return err
	}

	// Выполняем миграции из папки migrations/sql
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

		if _, err := db.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("ошибка при применении миграции %s: %w", version, err)
		}

		// Записываем информацию о примененной миграции
		_, err = db.DB.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version)
		if err != nil {
			return fmt.Errorf("ошибка при записи версии миграции %s: %w", version, err)
		}

		log.Printf("Миграция %s успешно применена", version)
	}

	return nil
}

func addColumnIfNotExists(tableName, columnName, columnDefinition string) error {
	// Проверяем, существует ли столбец
	query := fmt.Sprintf(`PRAGMA table_info(%s);`, tableName)
	rows, err := db.DB.Query(query)
	if err != nil {
		return fmt.Errorf("ошибка при получении информации о таблице %s: %w", tableName, err)
	}
	defer rows.Close()

	columnExists := false
	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, dflt_value, pk int
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt_value, &pk); err != nil {
			return fmt.Errorf("ошибка при сканировании информации о столбцах: %w", err)
		}
		if name == columnName {
			columnExists = true
			break
		}
	}

	if !columnExists {
		// Добавляем столбец, если его нет
		alterQuery := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s;`, tableName, columnName, columnDefinition)
		if _, err := db.DB.Exec(alterQuery); err != nil {
			return fmt.Errorf("ошибка при добавлении столбца %s в таблицу %s: %w", columnName, tableName, err)
		}
		log.Printf("Столбец %s успешно добавлен в таблицу %s", columnName, tableName)
	} else {
		log.Printf("Столбец %s уже существует в таблице %s, пропуск", columnName, tableName)
	}

	return nil
}

func ensureInitialMigration() error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("ошибка при создании таблицы schema_migrations: %w", err)
	}

	// Проверяем, применена ли начальная миграция
	var exists bool
	err = db.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 'initial')`).Scan(&exists)
	if err != nil || !exists {
		log.Println("Применение начальной миграции...")

		// Выполняем initial.sql
		content, err := ioutil.ReadFile("migrations/sql/initial.sql")
		if err != nil {
			return fmt.Errorf("ошибка при чтении initial.sql: %w", err)
		}

		if _, err := db.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("ошибка при применении initial.sql: %w", err)
		}

		// Записываем в schema_migrations
		_, err = db.DB.Exec(`INSERT INTO schema_migrations (version) VALUES ('initial')`)
		if err != nil {
			return fmt.Errorf("ошибка при записи initial в schema_migrations: %w", err)
		}

		log.Println("Начальная миграция успешно применена")
	}

	return nil
}
