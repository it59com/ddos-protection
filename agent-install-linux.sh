#!/bin/bash

# Параметры
AGENT_NAME="agent"
SERVICE_NAME="ddos-agent"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"
LOG_FILE="/var/log/ddos-agent.log"
BUILD_DIR=$(pwd)

# Функция для компиляции
compile_agent() {
    echo "Компиляция $AGENT_NAME..."
    go build -o "$BUILD_DIR/$AGENT_NAME" "$BUILD_DIR/$AGENT_NAME.go"
    if [[ $? -ne 0 ]]; then
        echo "Ошибка при компиляции!"
        exit 1
    fi
    echo "Компиляция завершена. Исполняемый файл: $BUILD_DIR/$AGENT_NAME"
}

# Функция для запуска
run_agent() {
    echo "Запуск $AGENT_NAME..."
    "$BUILD_DIR/$AGENT_NAME" &
    echo "$AGENT_NAME запущен как фоновый процесс."
}

# Функция для установки службы systemd
install_service() {
    echo "Создание службы systemd для $AGENT_NAME..."
    if [[ ! -f $BUILD_DIR/$AGENT_NAME ]]; then
        echo "Исполняемый файл $AGENT_NAME не найден! Сначала выполните компиляцию."
        exit 1
    fi

    sudo bash -c "cat > $SERVICE_FILE" <<EOL
[Unit]
Description=DDOS Protection Agent
After=network.target

[Service]
ExecStart=$BUILD_DIR/$AGENT_NAME
Restart=always
StandardOutput=file:$LOG_FILE
StandardError=file:$LOG_FILE

[Install]
WantedBy=multi-user.target
EOL

    # Перезагрузка systemd для загрузки новой службы
    sudo systemctl daemon-reload
    # Включение службы
    sudo systemctl enable $SERVICE_NAME
    # Запуск службы
    sudo systemctl start $SERVICE_NAME

    echo "Служба $SERVICE_NAME установлена и запущена."
}

# Функция для проверки статуса службы
check_status() {
    echo "Проверка статуса службы $SERVICE_NAME..."
    sudo systemctl status $SERVICE_NAME
}

# Функция для просмотра лога службы
view_log() {
    echo "Просмотр лога $SERVICE_NAME..."
    sudo tail -f $LOG_FILE
}

# Функция для удаления запущенных процессов agent
kill_agent_processes() {
    echo "Поиск и завершение всех запущенных процессов $AGENT_NAME..."
    pkill -f "$AGENT_NAME"
    echo "Все процессы $AGENT_NAME завершены."
}

# Функция для удаления службы
uninstall_service() {
    echo "Удаление службы $SERVICE_NAME..."
    sudo systemctl stop $SERVICE_NAME
    sudo systemctl disable $SERVICE_NAME
    sudo rm -f $SERVICE_FILE
    sudo systemctl daemon-reload
    echo "Служба $SERVICE_NAME удалена."
}

# Меню действий
case "$1" in
    compile)
        compile_agent
        ;;
    run)
        run_agent
        ;;
    install)
        compile_agent
        install_service
        ;;
    status)
        check_status
        ;;
    log)
        view_log
        ;;
    kill)
        kill_agent_processes
        ;;
    uninstall)
        uninstall_service
        kill_agent_processes
        ;;
    *)
        echo "Использование: $0 {compile|run|install|status|log|kill|uninstall}"
        echo "  compile     - Компиляция агентского исполняемого файла"
        echo "  run         - Запуск агентского исполняемого файла как фоновый процесс"
        echo "  install     - Компиляция и установка службы systemd"
        echo "  status      - Проверка статуса службы systemd"
        echo "  log         - Просмотр лога службы в режиме реального времени"
        echo "  kill        - Завершение всех запущенных процессов агентского исполняемого файла"
        echo "  uninstall   - Удаление службы и завершение всех процессов агентского исполняемого файла"
        ;;
esac
