package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"sync"
	"time"
)

type NodeView struct {
	ID                       int
	Address                  string
	PublicKey                string
	MessageQueue             []int
	mu                       sync.RWMutex
	LastHeartbeat            time.Time
	MaxTimeBetweenHeartbeats time.Duration
	Api                      api.PublicNodeApi
}

func NewNodeView(n *api.PrivateNodeApi, maxTimeBetweenHeartbeats time.Duration) *NodeView {
	return &NodeView{
		ID:                       n.ID,
		Address:                  n.Address,
		PublicKey:                n.PublicKey,
		MessageQueue:             n.MessageQueue,
		LastHeartbeat:            n.TimeOfRequest,
		MaxTimeBetweenHeartbeats: maxTimeBetweenHeartbeats,
		Api: api.PublicNodeApi{
			ID:        n.ID,
			Address:   n.Address,
			PublicKey: n.PublicKey,
		},
	}
}

func (nv *NodeView) UpdateNode(n *api.PrivateNodeApi) {
	nv.mu.Lock()
	defer nv.mu.Unlock()
	if nv.LastHeartbeat.After(n.TimeOfRequest) {
		return
	} else {
		nv.LastHeartbeat = n.TimeOfRequest
		nv.MessageQueue = n.MessageQueue
	}
}

func (nv *NodeView) IsActive() bool {
	nv.mu.RLock()
	defer nv.mu.RUnlock()
	return time.Since(nv.LastHeartbeat) < nv.MaxTimeBetweenHeartbeats
}
