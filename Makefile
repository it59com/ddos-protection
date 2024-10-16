# Папка для сборки
BUILD_DIR = build

# Пути к исходным файлам
SERVER_SRC = server/server.go
AGENT_SRC = agent/agent.go
UPDATE_SRC = update/update.go

# Пути к исполняемым файлам
SERVER_BIN = $(BUILD_DIR)/server
AGENT_BIN = $(BUILD_DIR)/agent
UPDATE_BIN = $(BUILD_DIR)/update

# Папка для конфигурации
CONFIG_DIR = /etc/ddos-protection

# Пути к файлам службы systemd
SYSTEMD_DIR = /etc/systemd/system
AGENT_SERVICE = systemd/ddos-agent.service
SERVER_SERVICE = systemd/ddos-server.service

# Пути конфигурации
CONFIG_DIR = /etc/ddos-protection
AGENT_CONFIG = agent.conf
AGENT_CONFIG_ORIGIN = agent.conf.orign
SERVER_CONFIG = config.json
SERVER_CONFIG_ORIGIN = config.json.orign


# Help
.PHONY: help

help:
	@echo "Доступные команды:"
	@echo "  make build           - Сборка сервера и агента"
	@echo "  make build-server    - Сборка только сервера"
	@echo "  make build-agent     - Сборка только агента"
	@echo "  make build-update    - Сборка только обновления"
	@echo "  make run-server      - Запуск сервера"
	@echo "  make run-agent       - Запуск агента"
	@echo "  make install-server  - Установка сервера как службы systemd"
	@echo "  make install-agent   - Установка агента как службы systemd"
	@echo "  make stop-server     - Остановка службы сервера"
	@echo "  make stop-agent      - Остановка службы агента"
	@echo "  make status-server   - Проверка статуса службы сервера"
	@echo "  make status-agent    - Проверка статуса службы агента"
	@echo "  make log-server      - Просмотр логов службы сервера"
	@echo "  make log-agent       - Просмотр логов службы агента"

# Команды сборки
all: build

build: build-server build-agent

build-server:
	@mkdir -p $(BUILD_DIR)
	@echo "Сборка сервера..."
	@go build -o $(SERVER_BIN) $(SERVER_SRC)
	@echo "Сервер собран: $(SERVER_BIN)"

build-agent:
	@mkdir -p $(BUILD_DIR)
	@echo "Сборка агента..."
	@go build -o $(AGENT_BIN) $(AGENT_SRC)
	@echo "Агент собран: $(AGENT_BIN)"

build-update:
	@mkdir -p $(BUILD_DIR)
	@echo "Сборка обновления..."
	@go build -o $(UPDATE_BIN) $(UPDATE_SRC)
	@echo "Update собран: $(UPDATE_BIN)"

# Команды для запуска
run-server:
	@$(SERVER_BIN)

run-agent:
	@$(AGENT_BIN)

# Команды для управления службами
install-server:
	@echo "Установка службы сервера..."
	@sudo cp $(SERVER_BIN) /usr/local/bin/ddos-server
	@sudo cp config/server.conf $(CONFIG_DIR)/server.conf
	@sudo systemctl enable --now ddos-server.service
	@echo "Служба сервера установлена и запущена."

install-agent:
	@echo "Установка службы агента..."
	@sudo cp $(AGENT_BIN) /usr/local/bin/ddos-agent
	@sudo cp config/agent.conf $(CONFIG_DIR)/agent.conf
	@sudo systemctl enable --now ddos-agent.service
	@echo "Служба агента установлена и запущена."

# Команды для остановки служб
stop-server:
	@echo "Остановка службы сервера..."
	@sudo systemctl stop ddos-server.service
	@echo "Служба сервера остановлена."

stop-agent:
	@echo "Остановка службы агента..."
	@sudo systemctl stop ddos-agent.service
	@echo "Служба агента остановлена."

# Команды для проверки статуса служб
status-server:
	@sudo systemctl status ddos-server.service

status-agent:
	@sudo systemctl status ddos-agent.service

# Команды для просмотра логов
log-server:
	@sudo journalctl -u ddos-server.service -f

log-agent:
	@sudo journalctl -u ddos-agent.service -f


# Удаление временных файлов и бинарников
clean:
	@echo "Очистка..."
	rm -rf $(BUILD_DIR)/*.log $(BUILD_DIR)/*.out $(AGENT_BIN) $(SERVER_BIN)
