package agentpc

import (
	"sync"
	"time"

	"github.com/google/gopacket"
)

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

type AgentService interface {
	LoadConfig(filename string) (*AgentConfig, error)
	ValidateInterface(interfaceName string) error
	HandlePackets(packetSource *gopacket.PacketSource, config *AgentConfig)
	WebSocketAgentConnect(url, token string)
}

var ipPortMutex sync.Mutex
var ipPortStates = make(map[string]*IPPortState)

type IPPortState struct {
	count     int
	lastReset time.Time
}
