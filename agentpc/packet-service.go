package agentpc

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Функция для обработки пакетов
func HandlePacketsAgent(packetSource *gopacket.PacketSource, config *AgentConfig) {
	for packet := range packetSource.Packets() {
		// Извлечение IP слоя
		ipLayer := packet.NetworkLayer()
		if ipLayer == nil {
			continue
		}

		// Извлечение информации об IP-адресе
		srcIP := ipLayer.NetworkFlow().Src().String()

		// Исключение пакетов от локального адреса
		if srcIP == config.LocalIP {
			log.Printf("Пропуск пакета от локального IP %s\n", srcIP)
			continue
		}

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
			} else {
				log.Printf("IP %s на порту %d успешно заблокирован\n", srcIP, srcPort)
			}
		}
	}
}

// isAllowedProtocol проверяет, соответствует ли протокол указанным в конфигурации
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

// isAllowedPort проверяет, входит ли порт в список разрешенных портов
func isAllowedPort(port int, ports []int) bool {
	for _, p := range ports {
		if port == p {
			return true
		}
	}
	return false
}

// checkAndBlockIP проверяет, превышен ли лимит запросов для данного IP и порта
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
		log.Printf("Начало отслеживания нового IP %s на порту %d", ip, port)
		return false
	}

	// Увеличиваем счетчик и проверяем лимит
	state.count++
	if state.count > config.RequestLimit {
		log.Printf("Превышен лимит для IP %s на порту %d. Блокировка...", ip, port)
		delete(ipPortStates, key) // Очистка состояния для данного IP и порта
		return true
	}

	return false
}
