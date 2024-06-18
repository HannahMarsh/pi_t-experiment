package bulletin_board

import (
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"net/http"
	"sync"
	"time"
)

// BulletinBoard represents the bulletin board that keeps track of active nodes and coordinates the start signal
type BulletinBoard struct {
	Network sync.Map // Maps node IDs to their queue sizes
	mu      sync.RWMutex
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard() *BulletinBoard {
	return &BulletinBoard{
		Network: sync.Map{},
	}
}

// UpdateNode adds a node to the active nodes list
func (bb *BulletinBoard) UpdateNode(node *api.PrivateNodeApi) error {
	var a any
	if _, present := bb.Network.Load(node.ID); !present {
		a, _ = bb.Network.LoadOrStore(node.ID, NewNodeView(node, time.Second*10))
	} else {
		a, _ = bb.Network.Load(node.ID)
	}
	if nv, ok := a.(*NodeView); !ok {
		return fmt.Errorf("bulletin_board.UpdateNode(): failed to cast to *NodeView")
	} else {
		nv.UpdateNode(node)
		return nil
	}
}

// GetActiveNodes returns the list of active nodes
func (bb *BulletinBoard) GetActiveNodes() []api.PublicNodeApi {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	return bb.ActiveNodes
}

func (bb *BulletinBoard) StartRuns() {
	for {
		time.Sleep(time.Second * 10)
		if bb.allNodesReady() {
			bb.signalNodesToStart()
		}
	}
}

func (bb *BulletinBoard) allNodesReady() bool {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	for _, node := range bb.ActiveNodes {
		queueSize, ok := bb.NodeQueueMap.Load(node.ID)
		if !ok || queueSize.(int) == 0 {
			return false
		}
	}
	return true
}

func (bb *BulletinBoard) signalNodesToStart() {
	for _, node := range bb.ActiveNodes {
		url := fmt.Sprintf("http://%s/start", node.Address)
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			fmt.Printf("Error signaling node %d to start: %v\n", node.ID, err)
			continue
		}
		resp.Body.Close()
	}
}
