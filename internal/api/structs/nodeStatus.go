package structs

import (
	"encoding/json"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

type NodeStatus struct {
	Received                 []OnionStatus
	Node                     PublicNodeApi
	CheckpointOnionsReceived map[int]int
	ExpectedCheckpoints      map[int]int
	TotalOnionsReceived      map[int]int
	mu                       sync.RWMutex
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
	NonceVerification bool
	ExpectCheckPoint  bool
}

func NewNodeStatus(id int, address, publicKey string, isMixer bool) *NodeStatus {
	return &NodeStatus{
		Received: make([]OnionStatus, 0),
		Node: PublicNodeApi{
			ID:        id,
			Address:   address,
			PublicKey: publicKey,
			Time:      time.Now(),
			IsMixer:   isMixer,
		},
		CheckpointOnionsReceived: make(map[int]int),
		ExpectedCheckpoints:      make(map[int]int),
		TotalOnionsReceived:      make(map[int]int),
	}
}

func (ns *NodeStatus) AddCheckpointOnion(layer int) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.CheckpointOnionsReceived[layer]++
}

func (ns *NodeStatus) AddExpectedCheckpoint(layer int) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.ExpectedCheckpoints[layer]++
}

func (ns *NodeStatus) AddOnion(lastHop, thisAddress, nextHop string, layer int, isCheckPointOnion bool) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.Received = append(ns.Received, OnionStatus{
		LastHop:           lastHop,
		ThisAddress:       thisAddress,
		NextHop:           nextHop,
		Layer:             layer,
		IsCheckPointOnion: isCheckPointOnion,
		TimeReceived:      time.Now(),
	})
	ns.TotalOnionsReceived[layer]++
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
