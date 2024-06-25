package api

import (
	"encoding/json"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

type NodeStatus struct {
	Received []OnionStatus
	Node     PublicNodeApi
	mu       sync.RWMutex
}

type OnionStatus struct {
	LastHop           string
	ThisAddress       string
	NextHop           string
	Layer             int
	IsCheckPointOnion bool
	TimeReceived      time.Time
	Bruises           int
	Dropped           bool
}

func (ns *NodeStatus) AddOnion(lastHop, thisAddress, nextHop string, layer int, isCheckPointOnion bool, bruises int, dropped bool) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.Received = append(ns.Received, OnionStatus{
		LastHop:           lastHop,
		ThisAddress:       thisAddress,
		NextHop:           nextHop,
		Layer:             layer,
		IsCheckPointOnion: isCheckPointOnion,
		TimeReceived:      time.Now(),
		Bruises:           bruises,
		Dropped:           dropped,
	})
}

func (ns *NodeStatus) GetStatus() string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	if str, err := json.Marshal(ns); err != nil {
		slog.Error("Error marshalling client status", err)
		return ""
	} else {
		return string(str)
	}
}
