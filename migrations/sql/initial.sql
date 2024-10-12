-- initial.sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы сессий
CREATE TABLE IF NOT EXISTS sessions (
   email TEXT NOT NULL,
    token TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    host TEXT,
    agent_name TEXT,
    is_active BOOLEAN DEFAULT TRUE,
     FOREIGN KEY (email) REFERENCES users(email) ON DELETE CASCADE
);

-- Создание таблицы запросов
CREATE TABLE IF NOT EXISTS requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    ip TEXT NOT NULL,
    host TEXT NOT NULL,
    request_count INTEGER DEFAULT 1,
    last_request DATETIME DEFAULT CURRENT_TIMESTAMP,
    firewall_source TEXT,
    port INTEGER
);

-- Создание таблицы заблокированных IP-адресов
CREATE TABLE IF NOT EXISTS ip_addresses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    ip TEXT NOT NULL,
    blocked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    request_count INTEGER DEFAULT 1,
    weight INTEGER DEFAULT 1,
    port INTEGER,  -- Добавлен столбец для порта
    firewall_source TEXT
);
