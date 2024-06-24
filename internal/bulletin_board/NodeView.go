package bulletin_board

import (
	"crypto/ecdh"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"sync"
	"time"
)

type NodeView struct {
	ID                       int
	Address                  string
	PublicKey                *ecdh.PublicKey
	mu                       sync.RWMutex
	LastHeartbeat            time.Time
	MaxTimeBetweenHeartbeats time.Duration
}

func NewNodeView(n api.PrivateNodeApi, maxTimeBetweenHeartbeats time.Duration) *NodeView {
	return &NodeView{
		ID:                       n.ID,
		Address:                  n.Address,
		PublicKey:                n.PublicKey,
		LastHeartbeat:            n.TimeOfRequest,
		MaxTimeBetweenHeartbeats: maxTimeBetweenHeartbeats,
	}
}

func (nv *NodeView) UpdateNode(c api.PrivateNodeApi) {
	nv.mu.Lock()
	defer nv.mu.Unlock()
	if nv.LastHeartbeat.After(c.TimeOfRequest) {
		return
	} else {
		nv.LastHeartbeat = c.TimeOfRequest
	}
}

func (nv *NodeView) IsActive() bool {
	nv.mu.RLock()
	defer nv.mu.RUnlock()
	return time.Since(nv.LastHeartbeat) < nv.MaxTimeBetweenHeartbeats
}
