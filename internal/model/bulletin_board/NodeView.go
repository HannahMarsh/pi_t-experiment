package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"sync"
	"time"
)

type NodeView struct {
	ID                       int
	Address                  string
	PublicKey                string
	mu                       sync.RWMutex
	LastHeartbeat            time.Time
	MaxTimeBetweenHeartbeats time.Duration
	IsMixer                  bool
}

func NewNodeView(n structs.PublicNodeApi, maxTimeBetweenHeartbeats time.Duration) *NodeView {
	return &NodeView{
		ID:                       n.ID,
		Address:                  n.Address,
		PublicKey:                n.PublicKey,
		LastHeartbeat:            n.Time,
		MaxTimeBetweenHeartbeats: maxTimeBetweenHeartbeats,
		IsMixer:                  n.IsMixer,
	}
}

func (nv *NodeView) UpdateNode(c structs.PublicNodeApi) {
	nv.mu.Lock()
	defer nv.mu.Unlock()
	if nv.LastHeartbeat.After(c.Time) {
		return
	} else {
		nv.LastHeartbeat = c.Time
	}
}

func (nv *NodeView) IsActive() bool {
	nv.mu.RLock()
	defer nv.mu.RUnlock()
	return time.Since(nv.LastHeartbeat) < nv.MaxTimeBetweenHeartbeats
}
