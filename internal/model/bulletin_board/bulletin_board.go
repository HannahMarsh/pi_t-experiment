package bulletin_board

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"net/http"
	"sync"
	"time"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
)

// BulletinBoard represents the bulletin board that keeps track of active nodes and coordinates the start signal
type BulletinBoard struct {
	Network         map[int]*NodeView   // Maps node IDs
	Clients         map[int]*ClientView // Maps client IDs
	mu              sync.RWMutex
	config          *config.Config
	lastStartRun    time.Time
	timeBetweenRuns time.Duration
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard(config *config.Config) *BulletinBoard {
	return &BulletinBoard{
		Network:         make(map[int]*NodeView),
		Clients:         make(map[int]*ClientView),
		config:          config,
		lastStartRun:    time.Now(),
		timeBetweenRuns: time.Second * 10,
	}
}

// UpdateNode adds a node to the active nodes list
func (bb *BulletinBoard) UpdateNode(node structs.PublicNodeApi) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if _, present := bb.Network[node.ID]; !present {
		bb.Network[node.ID] = NewNodeView(node, time.Second*10)
	}
	bb.Network[node.ID].UpdateNode(node)
	return nil
}

func (bb *BulletinBoard) RegisterClient(client structs.PublicNodeApi) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	if _, present := bb.Clients[client.ID]; !present {
		bb.Clients[client.ID] = NewClientView(client, time.Second*10)
	}
	return nil
}

func (bb *BulletinBoard) RegisterIntentToSend(its structs.IntentToSend) error {
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

func (bb *BulletinBoard) signalNodesToStart() error {
	slog.Info("Signaling nodes to start")
	activeNodes := utils.MapEntries(utils.FilterMap(bb.Network, func(_ int, node *NodeView) bool {
		return node.IsActive() && node.Address != ""
	}), func(_ int, nv *NodeView) structs.PublicNodeApi {
		return structs.PublicNodeApi{
			ID:        nv.ID,
			Address:   nv.Address,
			PublicKey: nv.PublicKey,
			Time:      nv.LastHeartbeat,
			IsMixer:   nv.IsMixer,
		}
	})

	activeClients := utils.MapEntries(utils.FilterMap(bb.Clients, func(_ int, cl *ClientView) bool {
		return cl.IsActive() && cl.Address != ""
	}), func(_ int, cv *ClientView) structs.PublicNodeApi {
		return structs.PublicNodeApi{
			ID:        cv.ID,
			Address:   cv.Address,
			PublicKey: cv.PublicKey,
		}
	})

	numMessages := utils.MaxValue(utils.MapEntries(bb.Clients, func(_ int, client *ClientView) int {
		return len(client.MessageQueue)
	})) + 2

	mixers := utils.Filter(activeNodes, func(n structs.PublicNodeApi) bool {
		return n.Address != "" && n.IsMixer
	})

	gatekeepers := utils.Filter(activeNodes, func(n structs.PublicNodeApi) bool {
		return n.Address != "" && !n.IsMixer
	})

	checkpoints := GetCheckpoints(activeNodes, activeClients, config.GlobalConfig.L, numMessages)

	clientStartSignals := make(map[structs.PublicNodeApi]structs.ClientStartRunApi)

	for _, client := range activeClients {
		csr := structs.ClientStartRunApi{
			Clients:              activeClients,
			Mixers:               mixers,
			Gatekeepers:          gatekeepers,
			NumMessagesPerClient: numMessages,
			Checkpoints:          checkpoints[client.ID],
		}
		clientStartSignals[client] = csr
	}

	nodeStartSignals := make(map[structs.PublicNodeApi]structs.NodeStartRunApi)
	for _, node := range activeNodes {
		nodeCheckpoints := utils.Filter(utils.MapFlatMap(checkpoints, func(_ int, cps []structs.Checkpoint) []structs.Checkpoint {
			return cps
		}), func(checkpoint structs.Checkpoint) bool {
			return checkpoint.Receiver.ID == node.ID
		})
		nsr := structs.NodeStartRunApi{
			Clients:     activeClients,
			Mixers:      mixers,
			Gatekeepers: gatekeepers,
			Checkpoints: nodeCheckpoints,
		}
		nodeStartSignals[node] = nsr
	}

	var wg sync.WaitGroup
	wg.Add(len(activeNodes) + len(activeClients))

	var err error
	for client, csr := range clientStartSignals {
		go func(client structs.PublicNodeApi, csr structs.ClientStartRunApi) {
			defer wg.Done()
			if data, err2 := json.Marshal(csr); err2 != nil {
				slog.Error("Error signaling client to start\n", err2)
				err = PrettyLogger.WrapError(err2, "failed to marshal start signal")
			} else {
				url := fmt.Sprintf("%s/start", client.Address)
				if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
					slog.Error("Error signaling client to start\n", err2)
					err = PrettyLogger.WrapError(err2, "failed to signal client to start")
				} else if err3 := resp.Body.Close(); err3 != nil {
					slog.Error("Error closing response body", err3)
					err = PrettyLogger.WrapError(err3, "failed to close response body")
				}
			}
		}(client, csr)
	}

	for node, nsr := range nodeStartSignals {
		defer wg.Done()
		node := node
		nsr := nsr
		go func() {
			if data, err2 := json.Marshal(nsr); err2 != nil {
				slog.Error("Error signaling node to start\n", err2)
				err = PrettyLogger.WrapError(err2, "failed to marshal start signal")
			} else {
				url := fmt.Sprintf("%s/start", node.Address)
				if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
					slog.Error("Error signaling node to start\n", err2)
					err = PrettyLogger.WrapError(err2, "failed to signal node to start")
				} else if err3 := resp.Body.Close(); err3 != nil {
					slog.Error("Error closing response body", err3)
					err = PrettyLogger.WrapError(err3, "failed to close response body")
				}
			}
		}()
	}

	wg.Wait()
	if err != nil {
		return PrettyLogger.WrapError(err, "error signaling nodes to start")
	}
	return nil
}

func (bb *BulletinBoard) StartRuns() error {
	for {
		if time.Since(bb.lastStartRun) >= bb.timeBetweenRuns {
			bb.lastStartRun = time.Now()
			if bb.allNodesReady() {
				if err := bb.signalNodesToStart(); err != nil {
					return PrettyLogger.WrapError(err, "error signaling nodes to start")
				} else {
					return nil
				}
			}
		}

		time.Sleep(time.Second * 5)
	}
}

func (bb *BulletinBoard) allNodesReady() bool {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	activeNodes := utils.Filter(utils.GetValues(bb.Network), func(node *NodeView) bool {
		return node.IsActive()
	})

	if len(activeNodes) < bb.config.MinNodes {
		slog.Info("Not enough active nodes")
		return false
	}

	totalMessages := utils.Sum(utils.MapEntries(bb.Clients, func(_ int, client *ClientView) int {
		return len(client.MessageQueue)
	}))

	if totalMessages < bb.config.MinTotalMessages {
		slog.Info("Not enough messages", "totalMessages", totalMessages, "Min", bb.config.MinTotalMessages)
		return false
	}

	slog.Info("All nodes ready")
	return true
}
