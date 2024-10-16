package main

import (
	"ddos-protection-api/agentpc"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func loadConfigAgent(filename string) (*agentpc.AgentConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл конфигурации: %w", err)
	}
	defer file.Close()

	config := &agentpc.AgentConfig{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("не удалось декодировать файл конфигурации: %w", err)
	}

	return config, nil
}

func validateInterfaceAgent(interfaceName string) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("ошибка при получении списка интерфейсов: %w", err)
	}

	for _, i := range interfaces {
		if i.Name == interfaceName {
			return nil
		}
	}
	return fmt.Errorf("интерфейс %s не найден", interfaceName)
}

// Обновленная функция для логирования
func setupLogging(logFilePath string) error {
	var logOutput io.Writer = os.Stdout // По умолчанию вывод в консоль

	if runtime.GOOS == "linux" {
		// Открываем файл для логов
		logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("не удалось открыть файл для логирования: %w", err)
		}
		// Вывод логов одновременно в файл и в консоль
		logOutput = io.MultiWriter(os.Stdout, logFile)
	}

	log.SetOutput(logOutput)
	return nil
}

// demonize делает агент фоновым процессом на Linux
func demonize() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("не удалось определить путь к исполняемому файлу: %w", err)
	}

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // Только на Linux

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("не удалось сделать форк: %w", err)
	}

	os.Exit(0)
	return nil
}

func runAgent() {
	// Настраиваем логирование в файл и консоль
	if err := setupLogging("/var/log/ddos-agent.log"); err != nil {
		log.Fatalf("Ошибка при настройке логирования: %v", err)
	}

	// Загружаем конфигурацию
	config, err := loadConfigAgent("agent.conf")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверка интерфейса
	if err := validateInterfaceAgent(config.Interface); err != nil {
		log.Fatalf("Ошибка проверки интерфейса: %v", err)
	}

	// Открытие устройства
	handle, err := pcap.OpenLive(config.Interface, 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("Ошибка при открытии устройства %s: %v", config.Interface, err)
	}
	defer handle.Close()

	go func() {
		filter := "tcp or udp"
		if err := handle.SetBPFFilter(filter); err != nil {
			log.Fatalf("Ошибка при установке фильтра: %v", err)
		}
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		agentpc.HandlePacketsAgent(packetSource, config)
	}()

	agentpc.NewWebSocketAgent(config.ServerURL, config.Token, config.AgentName)
	select {}
}

func main() {
	mode := flag.String("mode", "run", "режим запуска: run или service")
	flag.Parse()

	if *mode == "service" {
		if runtime.GOOS == "linux" {
			if err := demonize(); err != nil {
				log.Fatalf("Ошибка при демонизации процесса: %v", err)
			}
		}
		runAgent()
	} else {
		runAgent()
	}
}
