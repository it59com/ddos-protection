package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// Структура конфигурации агента
type AgentConfig struct {
	ServerURL    string   `json:"server_url"`
	Token        string   `json:"token"`
	Interface    string   `json:"interface"`
	AgentName    string   `json:"agent_name"`
	Protocols    []string `json:"protocols"`
	Ports        []int    `json:"ports"`
	RequestLimit int      `json:"request_limit"`
	TimeWindow   int      `json:"time_window_ms"`
}

const blockEndpoint = "/block"

// Структура для отслеживания состояния IP-адресов и портов
type IPPortState struct {
	count     int
	lastReset time.Time
}

var ipPortStates = make(map[string]*IPPortState)
var ipPortMutex sync.Mutex

// Чтение конфигурации из файла
func loadConfig(filename string) (*AgentConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл конфигурации: %w", err)
	}
	defer file.Close()

	config := &AgentConfig{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("не удалось декодировать файл конфигурации: %w", err)
	}

	return config, nil
}

// Функция для отправки запроса на блокировку IP с указанием порта
func blockIP(ip string, port int, config *AgentConfig) error {
	url := fmt.Sprintf("%s%s/%s?firewall=%s&port=%d", config.ServerURL, blockEndpoint, ip, config.AgentName, port)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Установка заголовка авторизации
	req.Header.Set("Authorization", "Bearer "+config.Token)

	// Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("не удалось заблокировать IP, статус код: %d", resp.StatusCode)
	}

	fmt.Printf("Запрос на блокировку IP %s на порту %d успешно выполнен\n", ip, port)
	return nil
}

// Функция для обработки пакетов
func handlePackets(packetSource *gopacket.PacketSource, config *AgentConfig) {
	for packet := range packetSource.Packets() {
		// Извлечение IP слоя
		ipLayer := packet.NetworkLayer()
		if ipLayer == nil {
			continue
		}

		// Извлечение информации об IP-адресе
		srcIP := ipLayer.NetworkFlow().Src().String()

		// Проверка протокола и порта
		if !isAllowedProtocol(packet, config.Protocols) {
			continue
		}

		// Извлечение порта из транспортного слоя
		transportLayer := packet.TransportLayer()
		if transportLayer == nil {
			continue
		}

		var srcPort int
		switch layer := transportLayer.(type) {
		case *layers.TCP:
			srcPort = int(layer.SrcPort)
		case *layers.UDP:
			srcPort = int(layer.SrcPort)
		}

		if !isAllowedPort(srcPort, config.Ports) {
			continue
		}

		// Проверка IP и порта, и блокировка при необходимости
		if checkAndBlockIP(srcIP, srcPort, config) {
			if err := blockIP(srcIP, srcPort, config); err != nil {
				log.Printf("Ошибка при блокировке IP %s на порту %d: %v\n", srcIP, srcPort, err)
			}
		}
	}
}

// Проверка IP и порта и блокировка при превышении лимита запросов
func checkAndBlockIP(ip string, port int, config *AgentConfig) bool {
	ipPortMutex.Lock()
	defer ipPortMutex.Unlock()

	key := fmt.Sprintf("%s:%d", ip, port)
	state, exists := ipPortStates[key]
	if !exists || time.Since(state.lastReset) > time.Duration(config.TimeWindow)*time.Millisecond {
		// Сброс счётчика при новом IP или по истечении временного окна
		ipPortStates[key] = &IPPortState{
			count:     1,
			lastReset: time.Now(),
		}
		return false
	}

	// Увеличиваем счетчик и проверяем лимит
	state.count++
	if state.count > config.RequestLimit {
		delete(ipPortStates, key) // Очистка состояния для данного IP и порта
		return true
	}

	return false
}

// Функция для проверки протокола пакета
func isAllowedProtocol(packet gopacket.Packet, protocols []string) bool {
	if transportLayer := packet.TransportLayer(); transportLayer != nil {
		protocol := transportLayer.LayerType()
		for _, p := range protocols {
			switch strings.ToLower(p) {
			case "tcp":
				if protocol == layers.LayerTypeTCP {
					return true
				}
			case "udp":
				if protocol == layers.LayerTypeUDP {
					return true
				}
			}
		}
	}
	return false
}

// Функция для проверки порта
func isAllowedPort(port int, ports []int) bool {
	for _, p := range ports {
		if port == p {
			return true
		}
	}
	return false
}

func main() {
	// Загружаем конфигурацию
	config, err := loadConfig("agent.conf")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Проверка интерфейса
	if err := validateInterface(config.Interface); err != nil {
		log.Fatalf("Ошибка проверки интерфейса: %v", err)
	}

	// Открытие устройства
	handle, err := pcap.OpenLive(config.Interface, 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("Ошибка при открытии устройства %s: %v", config.Interface, err)
	}
	defer handle.Close()

	// Установка фильтра для протоколов
	filter := fmt.Sprintf("tcp or udp")
	if err := handle.SetBPFFilter(filter); err != nil {
		log.Fatalf("Ошибка при установке фильтра: %v", err)
	}

	// Чтение пакетов
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	handlePackets(packetSource, config)
}

// validateInterface проверяет, доступен ли указанный интерфейс
func validateInterface(interfaceName string) error {
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
