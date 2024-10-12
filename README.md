# DDOS Protection API & Agent

Этот проект состоит из API-сервера для защиты от DDoS-атак и агента, который мониторит сетевой трафик на интерфейсах, блокирует IP-адреса при необходимости и отправляет данные на сервер. 

## Оглавление
- [Технологии](#технологии)
- [Установка и настройка](#установка-и-настройка)
  - [Установка API](#установка-api)
  - [Установка агента](#установка-агента)
  - [Настройка конфигурационного файла](#настройка-конфигурационного-файла)
  - [Установка агента как системного сервиса](#установка-агента-как-системного-сервиса)
- [Запуск и использование](#запуск-и-использование)
- [API Документация](#api-документация)
- [Требования](#требования)
- [Лицензия](#лицензия)

## Технологии
Проект использует следующие технологии:
- [Go](https://golang.org/) - язык программирования для API и агента
- [Gin](https://github.com/gin-gonic/gin) - веб-фреймворк для API
- [gopacket](https://github.com/google/gopacket) - библиотека для захвата и анализа пакетов
- [Npcap](https://nmap.org/npcap/) - драйвер для захвата пакетов в Windows
- [SQLite](https://sqlite.org/) - база данных для хранения информации о запросах и блокировках
- [Redis](https://redis.io/) - для учета лимитов запросов

## Установка и настройка

### Установка API
1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/yourusername/ddos-protection-api.git
   cd ddos-protection-api

### Настройка конфигурационного файла

Для настройки сервера создайте файл config.json в корневой директории проекта. Этот файл управляет настройками для базы данных, Redis и сервера API.

```json

{
    "database": {
        "type": "sqlite",                // Тип базы данных: sqlite, mysql, или postgres
        "connection": "ddos_protection.db" // Путь к базе данных для SQLite или строка подключения для MySQL/Postgres
    },
    "server": {
        "port": "8080",                  // Порт для запуска API сервера
        "ssl_cert": "",                  // Путь к SSL-сертификату (оставьте пустым, если SSL не используется)
        "ssl_key": ""                    // Путь к приватному ключу для SSL (оставьте пустым, если SSL не используется)
    },
    "redis": {
        "address": "localhost:6379",     // Адрес сервера Redis (например, localhost:6379)
        "password": "",                  // Пароль для Redis (оставьте пустым, если пароль не требуется)
        "db": 0                          // Номер базы данных Redis (по умолчанию 0)
    }
}

```Параметры конфигурации:

    database.type: Определяет тип используемой базы данных. Поддерживаются значения sqlite, mysql и postgres.
    database.connection: Строка подключения к базе данных. Для SQLite укажите путь к файлу базы данных, для MySQL или PostgreSQL укажите строку подключения.
    server.port: Порт, на котором будет запущен API-сервер.
    server.ssl_cert и server.ssl_key: Пути к файлам SSL-сертификата и ключа. Оставьте пустыми для запуска без SSL.
    redis.address: Адрес подключения к серверу Redis, обычно в формате localhost:6379.
    redis.password: Пароль для Redis. Если пароль не требуется, оставьте это поле пустым.
    redis.db: Номер базы данных в Redis, по умолчанию — 0.

```Пример файла agent.conf

Агент также требует конфигурационный файл agent.conf, который должен находиться в одной директории с исполняемым файлом агента:

```json

{
    "server_url": "http://localhost:8080",
    "token": "your_jwt_token_here",
    "interface": "eth0",
    "agent_name": "Agent1",
    "protocols": ["tcp", "udp"],
    "ports": [22, 80, 443],
    "request_limit": 100,
    "time_window_ms": 100
}

```Параметры конфигурации агента:

    server_url: URL адрес API-сервера, куда агент отправляет запросы.
    token: JWT-токен для аутентификации агента.
    interface: Имя сетевого интерфейса, который агент будет мониторить.
    agent_name: Уникальное имя агента, используемое для идентификации источника запросов.
    protocols: Список протоколов для мониторинга (например, tcp, udp).
    ports: Список портов для мониторинга. Агент будет отслеживать только указанные порты.
    request_limit: Лимит запросов от одного IP за заданное окно времени.
    time_window_ms: Окно времени в миллисекундах для учета лимита запросов.