package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"sync"
	"time"
)

type RelayView struct {
	ID                       int
	Address                  string
	PublicKey                string
	Host                     string
	Port                     int
	PromPort                 int
	mu                       sync.RWMutex
	LastHeartbeat            time.Time
	MaxTimeBetweenHeartbeats time.Duration
}

func NewNodeView(n structs.PublicNodeApi, maxTimeBetweenHeartbeats time.Duration) *RelayView {
	return &RelayView{
		ID:                       n.ID,
		Address:                  n.Address,
		PublicKey:                n.PublicKey,
		Host:                     n.Host,
		Port:                     n.Port,
		PromPort:                 n.PrometheusPort,
		LastHeartbeat:            n.Time,
		MaxTimeBetweenHeartbeats: maxTimeBetweenHeartbeats,
	}
}

func (nv *RelayView) UpdateNode(c structs.PublicNodeApi) {
	nv.mu.Lock()
	defer nv.mu.Unlock()
	if nv.LastHeartbeat.After(c.Time) {
		return
	} else {
		nv.LastHeartbeat = c.Time
	}
}

func (nv *RelayView) IsActive() bool {
	nv.mu.RLock()
	defer nv.mu.RUnlock()
	return time.Since(nv.LastHeartbeat) < nv.MaxTimeBetweenHeartbeats
}
