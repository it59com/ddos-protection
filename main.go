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
	serverFile  = "server/server_api"
	agentFile   = "agent/agent.go"
	updateFile  = "update/update.go"
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
		buildAll()
	default:
		fmt.Println("Неизвестная команда:", command)
		fmt.Println("Использование: main.go {start|stop|status|update|build}")
	}
}

// Функция для запуска службы
func startService() {
	cmd := exec.Command("./server/server")
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
	// Остановка всех процессов с именем server_api
	out, err := exec.Command("pkill", "-f", "server_api").CombinedOutput()
	if err != nil {
		log.Printf("Ошибка при остановке сервера API: %v", err)
		fmt.Println(string(out))
		return
	}

	fmt.Println("Сервер API остановлен.")
}

// Функция для проверки статуса службы
func statusService() {
	out, err := exec.Command("pgrep", "-fl", "server_api").CombinedOutput()
	if err != nil || len(out) == 0 {
		fmt.Println("Сервер API не запущен.")
	} else {
		fmt.Printf("Сервер API запущен:\n%s", out)
	}
}

// Функция для обновления агента
func updateAgent() {
	fmt.Println("Запуск обновления агента...")
	cmd := exec.Command("./update/update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Ошибка при обновлении агента: %v", err)
	}

	fmt.Println("Обновление агента завершено.")
}

// Функция для сборки всех исполняемых файлов
func buildAll() {
	// Приведите сборку к тому виду, который отражает фактические имена и пути к файлам
	filesToBuild := map[string]string{
		"./server/server": "server/server.go", // путь к исходному коду и место, где вы хотите собрать исполняемый файл
		"./agent/agent":   "agent/agent.go",   // аналогично для агента
		//"./update/update":     "update/update.go", // аналогично для update
	}

	for outputFile, sourceFile := range filesToBuild {
		fmt.Printf("Сборка %s из %s...\n", outputFile, sourceFile)

		cmd := exec.Command("go", "build", "-o", outputFile, sourceFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Fatalf("Ошибка при сборке %s: %v", outputFile, err)
		}

		fmt.Printf("Сборка завершена: %s\n", outputFile)
	}

	fmt.Println("Все файлы успешно собраны.")
}
