package structs

import (
	"encoding/json"
	"github.com/HannahMarsh/PrettyLogger"
	"golang.org/x/exp/slog"
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

func NewClientStatus(id int, address, publicKey string) *ClientStatus {
	return &ClientStatus{
		MessagesSent:     make([]Sent, 0),
		MessagesReceived: make([]Received, 0),
		Client: PublicNodeApi{
			ID:        id,
			Address:   address,
			PublicKey: publicKey,
			Time:      time.Now(),
			IsMixer:   false,
		},
	}
}

func (cs *ClientStatus) AddSent(clientReceiver PublicNodeApi, routingPath []PublicNodeApi, message Message) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.MessagesSent = append(cs.MessagesSent, Sent{
		ClientReceiver: clientReceiver,
		RoutingPath:    routingPath,
		Message:        message,
		TimeSent:       time.Now(),
	})

	//	slog.Info(PrettyLogger.GetFuncName(), "message", message)
}

func (cs *ClientStatus) AddReceived(message Message) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.MessagesReceived = append(cs.MessagesReceived, Received{
		Message:      message,
		TimeReceived: time.Now(),
	})
	slog.Info(PrettyLogger.GetFuncName(), "message", message)
}

func (cs *ClientStatus) GetStatus() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if str, err := json.Marshal(cs); err != nil {
		slog.Error("Error marshalling client status", err)
		return ""
	} else {
		return string(str)
	}
}
