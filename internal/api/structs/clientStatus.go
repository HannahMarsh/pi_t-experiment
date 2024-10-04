package structs

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"log/slog"
	"sync"
	"time"
)

type ClientStatus struct {
	MessagesSent     []Sent
	MessagesReceived []Received
	Client           PublicNodeApi
	mu               sync.RWMutex
}

type Sent struct {
	ClientReceiver PublicNodeApi
	RoutingPath    []PublicNodeApi
	Message        Message
	TimeSent       time.Time
}

type Received struct {
	Message      Message
	TimeReceived time.Time
}

func NewClientStatus(id, port, promPort int, address, host, publicKey string) *ClientStatus {
	return &ClientStatus{
		MessagesSent:     make([]Sent, 0),
		MessagesReceived: make([]Received, 0),
		Client: PublicNodeApi{
			ID:             id,
			Address:        address,
			PublicKey:      publicKey,
			Host:           host,
			Port:           port,
			PrometheusPort: promPort,
			Time:           time.Now(),
		},
	}
}

func (cs *ClientStatus) AddSent(clientReceiver PublicNodeApi, routingPath []PublicNodeApi, message Message) {
	if config.GetVis() {
		cs.mu.Lock()
		defer cs.mu.Unlock()
		cs.MessagesSent = append(cs.MessagesSent, Sent{
			ClientReceiver: clientReceiver,
			RoutingPath:    routingPath,
			Message:        message,
			TimeSent:       time.Now(),
		})
	}

	//	slog.Info(PrettyLogger.GetFuncName(), "message", message)
}

func (cs *ClientStatus) AddReceived(message Message) {
	if config.GetVis() {
		cs.mu.Lock()
		defer cs.mu.Unlock()
		cs.MessagesReceived = append(cs.MessagesReceived, Received{
			Message:      message,
			TimeReceived: time.Now(),
		})
		//slog.Info("", "from", config.AddressToName(message.From), "to", config.AddressToName(message.To), "message", message.Msg)
	}
}

func (cs *ClientStatus) GetStatus() string {
	if config.GetVis() {
		cs.mu.RLock()
		defer cs.mu.RUnlock()
		if str, err := json.Marshal(cs); err != nil {
			slog.Error("Error marshalling client status", err)
			return ""
		} else {
			return string(str)
		}
	} else {
		return ""
	}
}
