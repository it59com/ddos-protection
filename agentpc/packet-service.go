package agentpc

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func HandlePacketsAgent(packetSource *gopacket.PacketSource, config *AgentConfig) {
	for packet := range packetSource.Packets() {
		// Извлечение IP слоя
		ipLayer := packet.NetworkLayer()
		if ipLayer == nil {
			continue
		}

		// Извлечение информации об IP-адресе
		srcIP := ipLayer.NetworkFlow().Src().String()
		dstIP := ipLayer.NetworkFlow().Dst().String()

		// Пропускаем пакеты, отправленные от локального IP
		if dstIP == config.LocalIP {
			continue
		}

		// Проверка протокола
		if !isAllowedProtocol(packet, config.Protocols) {
			continue
		}

		// Извлечение порта из транспортного слоя
		transportLayer := packet.TransportLayer()
		if transportLayer == nil {
			continue
		}

		//var srcPort int
		var dstPort int

		switch layer := transportLayer.(type) {
		case *layers.TCP:
			//srcPort = int(layer.SrcPort)
			dstPort = int(layer.DstPort)
		case *layers.UDP:
			//srcPort = int(layer.SrcPort)
			dstPort = int(layer.DstPort)
		}

		// Если задан список портов, проверяем только входящие пакеты на эти порты
		if len(config.Ports) > 0 && !isAllowedPort(dstPort, config.Ports) {
			continue
		}

		// Проверка количества запросов от `srcIP` к `dstPort`, и при необходимости отправка на блокировку
		if checkAndBlockIP(srcIP, dstPort, config) {
			if err := blockIP(srcIP, dstPort, config); err != nil {
				log.Printf("Ошибка при блокировке IP %s на порту %d: %v\n", srcIP, dstPort, err)
			} else {
				log.Printf("IP %s на порту %d успешно заблокирован\n", srcIP, dstPort)
			}
		}
	}
}

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

func isAllowedPort(port int, ports []int) bool {
	for _, p := range ports {
		if port == p {
			return true
		}
	}
	return false
}

func checkAndBlockIP(ip string, port int, config *AgentConfig) bool {
	// Check if the IP is in the excluded list
	for _, excludedIP := range config.ExcludeIPs {
		if strings.HasPrefix(ip, excludedIP) {
			//log.Printf("IP %s исключен из подсчета", ip)
			return false
		}
	}

	ipPortMutex.Lock()
	defer ipPortMutex.Unlock()

	key := fmt.Sprintf("%s:%d", ip, port)
	state, exists := ipPortStates[key]

	if !exists || time.Since(state.lastReset) > time.Duration(config.TimeWindow)*time.Millisecond {
		ipPortStates[key] = &IPPortState{
			count:     1,
			lastReset: time.Now(),
		}
		log.Printf("Начало отслеживания нового IP %s на порту %d", ip, port)
		return false
	}

	state.count++

	if state.count > config.RequestLimit {
		log.Printf("Превышен лимит для IP %s на порту %d. Блокировка...", ip, port)
		delete(ipPortStates, key) // Удаление записи после блокировки
		return true
	} else {
		log.Printf("Отслеживание IP %s на порту %d: текущее количество запросов %d", ip, port, state.count)
	}

	return false
}
