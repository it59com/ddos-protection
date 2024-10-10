package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Структура конфигурации агента
type AgentConfig struct {
	ServerURL string   `json:"server_url"`
	Token     string   `json:"token"`
	Interface string   `json:"interface"`
	AgentName string   `json:"agent_name"`
	Protocols []string `json:"protocols"`
	Ports     []int    `json:"ports"`
}

const blockEndpoint = "/block"

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

// Проверка, доступен ли интерфейс
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

// Функция для отправки запроса на блокировку IP
func blockIP(ip string, config *AgentConfig) error {
	url := fmt.Sprintf("%s%s/%s?firewall=%s", config.ServerURL, blockEndpoint, ip, config.AgentName)
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

	fmt.Printf("Запрос на блокировку IP %s успешно выполнен\n", ip)
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
		if !isAllowedProtocol(packet, config.Protocols) || !isAllowedPort(packet, config.Ports) {
			continue
		}

		// Блокировка IP
		if err := blockIP(srcIP, config); err != nil {
			log.Printf("Ошибка при блокировке IP %s: %v\n", srcIP, err)
		}
	}
}

// Функция для проверки протокола пакета
func isAllowedProtocol(packet gopacket.Packet, protocols []string) bool {
	proto := packet.TransportLayer().LayerType().String()
	for _, p := range protocols {
		if strings.EqualFold(p, proto) {
			return true
		}
	}
	return false
}

// Функция для проверки порта пакета
func isAllowedPort(packet gopacket.Packet, ports []int) bool {
	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return false
	}
	srcPortStr, dstPortStr := transportLayer.TransportFlow().Endpoints()
	srcPort, err1 := strconv.Atoi(srcPortStr.String())
	dstPort, err2 := strconv.Atoi(dstPortStr.String())

	if err1 != nil || err2 != nil {
		return false
	}

	for _, port := range ports {
		if port == srcPort || port == dstPort {
			return true
		}
	}
	return false
}

// Установка агента как системного сервиса
func installService() error {
	serviceContent := `[Unit]
Description=DDOS Protection Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/ddos-agent
Restart=always
User=root

[Install]
WantedBy=multi-user.target`

	servicePath := "/etc/systemd/system/ddos-agent.service"

	// Запись файла сервиса
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("ошибка записи файла сервиса: %w", err)
	}

	// Активируем сервис
	cmds := []string{"systemctl daemon-reload", "systemctl enable ddos-agent", "systemctl start ddos-agent"}
	for _, cmd := range cmds {
		parts := strings.Fields(cmd)
		if err := exec.Command(parts[0], parts[1:]...).Run(); err != nil {
			return fmt.Errorf("ошибка при выполнении команды %s: %w", cmd, err)
		}
	}

	fmt.Println("Сервис ddos-agent установлен и запущен.")
	return nil
}

func listInterfaces() {

	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Ошибка при получении списка интерфейсов: %v", err)
	}
	fmt.Println("Доступные интерфейсы:")
	for _, i := range interfaces {
		fmt.Printf("- %s\n", i.Name)
	}
}

func main() {
	listInterfaces()
	// Загрузка конфигурации
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

	// Установка фильтра для протоколов и портов
	filter := fmt.Sprintf("tcp or udp")
	if err := handle.SetBPFFilter(filter); err != nil {
		log.Fatalf("Ошибка при установке фильтра: %v", err)
	}
	fmt.Printf("Запуск агента на интерфейсе %s с именем агента %s\n", config.Interface, config.AgentName)
	fmt.Printf("Протоколы: %v, Порты: %v\n", config.Protocols, config.Ports)

	// Чтение пакетов
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	handlePackets(packetSource, config)
}
