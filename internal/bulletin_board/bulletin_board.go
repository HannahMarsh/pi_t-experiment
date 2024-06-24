package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/config"
	"sync"
	"time"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
)

// BulletinBoard represents the bulletin board that keeps track of active nodes and coordinates the start signal
type BulletinBoard struct {
	Network map[int]*NodeView   // Maps node IDs
	Clients map[int]*ClientView // Maps client IDs
	mu      sync.RWMutex
	config  *config.Config
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard(config *config.Config) *BulletinBoard {
	return &BulletinBoard{
		Network: make(map[int]*NodeView),
		Clients: make(map[int]*ClientView),
		config:  config,
	}
}

// UpdateNode adds a node to the active nodes list
func (bb *BulletinBoard) UpdateNode(node api.PrivateNodeApi) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if _, present := bb.Network[node.ID]; !present {
		bb.Network[node.ID] = NewNodeView(node, time.Second*10)
	}
	bb.Network[node.ID].UpdateNode(node)
	return nil
}

func (bb *BulletinBoard) RegisterClient(client api.PublicClientApi) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if _, present := bb.Clients[client.ID]; !present {
		bb.Clients[client.ID] = NewClientView(client, time.Second*10)
	}
	return nil
}

func (bb *BulletinBoard) RegisterIntentToSend(its api.IntentToSend) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if _, present := bb.Clients[its.From.ID]; !present {
		bb.Clients[its.From.ID] = NewClientView(its.From, time.Second*10)
	} else {
		for _, client := range its.To {
			if _, present = bb.Clients[client.ID]; !present {
				bb.Clients[client.ID] = NewClientView(client, time.Second*10)
			}
		}
	}
	bb.Clients[its.From.ID].UpdateClient(its)
	return nil
}

// GetActiveNodes returns the list of active nodes
func (bb *BulletinBoard) GetActiveNodes() []api.PublicNodeApi {
	bb.mu.RLock()
	defer bb.mu.RUnlock()

	return utils.Map(utils.NewMapStream(bb.Network).Filter(func(_ int, node *NodeView) bool {
		return node.IsActive()
	}).GetValues().Array, func(node *NodeView) api.PublicNodeApi {
		return api.PublicNodeApi{
			ID:        node.ID,
			Address:   node.Address,
			PublicKey: node.PublicKey,
		}
	})
}

// GetActiveNodes returns the list of active nodes
func (bb *BulletinBoard) GetActiveClients() []api.PublicClientApi {
	bb.mu.RLock()
	defer bb.mu.RUnlock()

	return utils.Map(utils.NewMapStream(bb.Clients).Filter(func(_ int, client *ClientView) bool {
		return client.IsActive()
	}).GetValues().Array, func(client *ClientView) api.PublicClientApi {
		return api.PublicClientApi{
			ID:        client.ID,
			Address:   client.Address,
			PublicKey: client.PublicKey,
		}
	})
}

func (bb *BulletinBoard) StartRuns() error {
	for {
		time.Sleep(time.Second * 10)
		if bb.allNodesReady() {
			if err := bb.signalNodesToStart(); err != nil {
				return PrettyLogger.WrapError(err, "error signaling nodes to start")
			}
		}
	}
}

func (bb *BulletinBoard) allNodesReady() bool {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	activeNodes := utils.NewMapStream(bb.Network).Filter(func(_ int, node *ClientView) bool {
		return node.IsActive()
	}).GetValues()

	if len(activeNodes.Array) < bb.config.MinNodes {
		slog.Info("Not enough active nodes")
		return false
	}

	return activeNodes.All(func(node *ClientView) bool {
		length := len(node.MessageQueue) >= bb.config.MinQueueLength
		if !length {
			slog.Info("Node not ready", "id", node.ID, "queue_length", len(node.MessageQueue), "min_queue_length", bb.config.MinQueueLength)
		} else {
			slog.Info("Node ready", "id", node.ID, "queue_length", len(node.MessageQueue), "min_queue_length", bb.config.MinQueueLength)
		}
		return length
	})
}
