package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// Параметры для службы
const (
	serviceName = "ddos-api-server"
	execFile    = "./server_api.go"
	agentFile   = "agent.go"
	updateFile  = "update.go"
)

// Основная функция
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: main.go {start|stop|status|update|build}")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "start":
		startService()
	case "stop":
		stopService()
	case "status":
		statusService()
	case "update":
		updateAgent()
	case "build":
		buildAgent()
	default:
		fmt.Println("Неизвестная команда:", command)
		fmt.Println("Использование: main.go {start|stop|status|update|build}")
	}
}

// Функция для запуска службы
func startService() {
	// Компилируем файл server_api.go в исполняемый файл server_api
	buildCmd := exec.Command("go", "build", "-o", "server_api", execFile)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Ошибка при компиляции сервера API: %v", err)
	}

	// Запускаем полученный исполняемый файл server_api
	cmd := exec.Command("./server_api")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Ошибка при запуске сервера API: %v", err)
	}

	fmt.Println("Сервер API запущен.")
}

// Функция для остановки службы
func stopService() {
	// Остановка всех процессов с именем execFile
	out, err := exec.Command("pkill", "-f", execFile).CombinedOutput()
	if err != nil {
		log.Printf("Ошибка при остановке сервера API: %v", err)
		fmt.Println(string(out))
		return
	}

	fmt.Println("Сервер API остановлен.")
}

// Функция для проверки статуса службы
func statusService() {
	// Поиск процесса сервера API
	out, err := exec.Command("pgrep", "-fl", execFile).CombinedOutput()
	if err != nil || len(out) == 0 {
		fmt.Println("Сервер API не запущен.")
	} else {
		fmt.Printf("Сервер API запущен:\n%s", out)
	}
}

// Функция для обновления агента
func updateAgent() {
	fmt.Println("Запуск обновления агента...")
	cmd := exec.Command("go", "run", updateFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Ошибка при обновлении агента: %v", err)
	}

	fmt.Println("Обновление агента завершено.")
}

// Функция для сборки агента в релиз
func buildAgent() {
	fmt.Println("Сборка агента...")

	// Компиляция агентского файла
	cmd := exec.Command("go", "build", "-o", agentFile, "agent.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Ошибка при сборке агента: %v", err)
	}

	fmt.Printf("Сборка завершена. Исполняемый файл: %s\n", agentFile)
}
