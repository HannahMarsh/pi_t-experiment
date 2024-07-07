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

	vs := structs.StartRunApi{
		ParticipatingClients: activeClients,
		Mixers:               mixers,
		Gatekeepers:          gatekeepers,
		NumMessagesPerClient: numMessages,
	}

	if data, err := json.Marshal(vs); err != nil {
		return PrettyLogger.WrapError(err, "failed to marshal start signal")
	} else {
		var wg sync.WaitGroup
		all := utils.Copy(activeNodes)
		all = append(all, activeClients...)
		all = utils.Filter(all, func(n structs.PublicNodeApi) bool {
			return n.Address != ""
		})
		for _, n := range all {
			n := n
			wg.Add(1)
			go func() {
				defer wg.Done()
				url := fmt.Sprintf("%s/start", n.Address)
				if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
					slog.Error("Error signaling node to start \n"+url, err2)
				} else if err3 := resp.Body.Close(); err3 != nil {
					fmt.Printf("Error closing response body: %v\n", err3)
				}
			}()
		}
		//for _, c := range activeClients {
		//	c := c
		//	wg.Add(1)
		//	go func() {
		//		defer wg.Done()
		//		url := fmt.Sprintf("%s/start", c.Address)
		//		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
		//			slog.Error("Error signaling client to start\n", err2)
		//		} else if err3 := resp.Body.Close(); err3 != nil {
		//			fmt.Printf("Error closing response body: %v\n", err3)
		//		}
		//	}()
		//}
		wg.Wait()
		return nil
	}
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
	activeNodes := utils.NewMapStream(bb.Network).Filter(func(_ int, node *NodeView) bool {
		return node.IsActive()
	}).GetValues()

	if len(activeNodes.Array) < bb.config.MinNodes {
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
