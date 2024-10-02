package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"time"
)

type ClientView struct {
	ID                       int
	Address                  string
	PublicKey                string
	Host                     string
	Port                     int
	PromPort                 int
	MessageQueue             []structs.PublicNodeApi
	LastHeartbeat            time.Time
	MaxTimeBetweenHeartbeats time.Duration
}

func NewClientView(c structs.PublicNodeApi, maxTimeBetweenHeartbeats time.Duration) *ClientView {
	return &ClientView{
		ID:                       c.ID,
		Address:                  c.Address,
		PublicKey:                c.PublicKey,
		Host:                     c.Host,
		Port:                     c.Port,
		PromPort:                 c.PrometheusPort,
		MessageQueue:             make([]structs.PublicNodeApi, 0),
		LastHeartbeat:            time.Now(),
		MaxTimeBetweenHeartbeats: maxTimeBetweenHeartbeats,
	}
}

func (nv *ClientView) UpdateClient(c structs.IntentToSend) {
	//if nv.LastHeartbeat.After(c.Time) {
	//	slog.Warn("ClientView.UpdateClient(): received heartbeat from client that is older than the last heartbeat")
	//	return
	//} else {
	nv.LastHeartbeat = c.Time
	nv.MessageQueue = c.To
	//}
}

func (nv *ClientView) IsActive() bool {
	return true
	//nv.mu.RLock()
	//defer nv.mu.RUnlock()
	//return time.Since(nv.LastHeartbeat) < nv.MaxTimeBetweenHeartbeats
}
