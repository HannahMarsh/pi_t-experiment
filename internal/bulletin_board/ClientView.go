package bulletin_board

import (
	"crypto/ecdh"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"sync"
	"time"
)

type ClientView struct {
	ID                       int
	Address                  string
	PublicKey                *ecdh.PublicKey
	MessageQueue             []api.PublicClientApi
	mu                       sync.RWMutex
	LastHeartbeat            time.Time
	MaxTimeBetweenHeartbeats time.Duration
}

func NewClientView(c api.PublicClientApi, maxTimeBetweenHeartbeats time.Duration) *ClientView {
	return &ClientView{
		ID:                       c.ID,
		Address:                  c.Address,
		PublicKey:                c.PublicKey,
		MessageQueue:             make([]api.PublicClientApi, 0),
		LastHeartbeat:            time.Now(),
		MaxTimeBetweenHeartbeats: maxTimeBetweenHeartbeats,
	}
}

func (nv *ClientView) UpdateClient(c api.IntentToSend) {
	nv.mu.Lock()
	defer nv.mu.Unlock()
	if nv.LastHeartbeat.After(c.Time) {
		return
	} else {
		nv.LastHeartbeat = c.Time
		nv.MessageQueue = c.To
	}
}

func (nv *ClientView) IsActive() bool {
	nv.mu.RLock()
	defer nv.mu.RUnlock()
	return time.Since(nv.LastHeartbeat) < nv.MaxTimeBetweenHeartbeats
}
