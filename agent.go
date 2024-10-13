package main

import (
	"ddos-protection-api/agentpc" // Убедитесь, что здесь указан правильный импорт пакета
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Чтение конфигурации из файла
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

// validateInterface проверяет, доступен ли указанный интерфейс
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

func main() {
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

	// Запуск мониторинга пакетов в отдельной горутине
	go func() {
		filter := "tcp or udp"
		if err := handle.SetBPFFilter(filter); err != nil {
			log.Fatalf("Ошибка при установке фильтра: %v", err)
		}
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		agentpc.HandlePacketsAgent(packetSource, config)
	}()
	agentName := config.AgentName
	agentpc.NewWebSocketAgent(config.ServerURL, config.Token, agentName)

	// Запуск WebSocket подключения в отдельной горутине

	// Блокировка основного потока, чтобы горутины продолжали выполняться
	select {}
}
