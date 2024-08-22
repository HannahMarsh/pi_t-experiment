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

	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"log/slog"
)

// BulletinBoard keeps track of active relays and coordinates the start signal
type BulletinBoard struct {
	Network         map[int]*RelayView  // Maps relay IDs to their respective RelayView structs.
	Clients         map[int]*ClientView // Maps client IDs to their respective ClientView structs.
	mu              sync.RWMutex        // Mutex for read/write locking
	lastStartRun    time.Time           // Timestamp of the last start signal sent.
	timeBetweenRuns time.Duration       // Minimum duration between consecutive start signals.
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard() *BulletinBoard {
	return &BulletinBoard{
		Network:         make(map[int]*RelayView),
		Clients:         make(map[int]*ClientView),
		lastStartRun:    time.Now(),
		timeBetweenRuns: time.Second * 10,
	}
}

// UpdateNode adds or updates a relay in the active nodes list based on the provided PublicNodeApi data.
func (bb *BulletinBoard) UpdateNode(node structs.PublicNodeApi) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// If the node is not already present in the Network, create a new RelayView for it.
	if _, present := bb.Network[node.ID]; !present {
		bb.Network[node.ID] = NewNodeView(node, time.Second*10)
	}

	// Update the existing or newly created RelayView with the latest node information.
	bb.Network[node.ID].UpdateNode(node)
}

// RegisterClient adds a client to the active clients list based on the provided PublicNodeApi data.
func (bb *BulletinBoard) RegisterClient(client structs.PublicNodeApi) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// If the client is not already present in the Clients map, create a new ClientView for it.
	if _, present := bb.Clients[client.ID]; !present {
		bb.Clients[client.ID] = NewClientView(client, time.Second*10)
	}
	return
}

// RegisterIntentToSend records a client's intent to send a message, updating the active clients list accordingly.
func (bb *BulletinBoard) RegisterIntentToSend(its structs.IntentToSend) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Ensure the sender is registered in the Clients map.
	if _, present := bb.Clients[its.From.ID]; !present {
		bb.Clients[its.From.ID] = NewClientView(its.From, time.Second*10)
	} else {
		// Register any additional clients in the 'To' field of the IntentToSend.
		for _, client := range its.To {
			if _, present = bb.Clients[client.ID]; !present {
				bb.Clients[client.ID] = NewClientView(client, time.Second*10)
			}
		}
	}
	// Update the sender's ClientView with the intent to send data.
	bb.Clients[its.From.ID].UpdateClient(its)
	return nil
}

// signalNodesToStart sends the start signal to all active nodes (clients and relays) in the network.
func (bb *BulletinBoard) signalNodesToStart() error {
	slog.Info("Signaling nodes to start...")

	// Filter and map active relays to their PublicNodeApi representations.
	activeNodes := utils.MapEntries(utils.FilterMap(bb.Network, func(_ int, node *RelayView) bool {
		return node.IsActive() && node.Address != ""
	}), func(_ int, nv *RelayView) structs.PublicNodeApi {
		return structs.PublicNodeApi{
			ID:        nv.ID,
			Address:   nv.Address,
			PublicKey: nv.PublicKey,
			Time:      nv.LastHeartbeat,
		}
	})

	// Filter and map active clients to their PublicNodeApi representations.
	activeClients := utils.MapEntries(utils.FilterMap(bb.Clients, func(_ int, cl *ClientView) bool {
		return cl.IsActive() && cl.Address != ""
	}), func(_ int, cv *ClientView) structs.PublicNodeApi {
		return structs.PublicNodeApi{
			ID:        cv.ID,
			Address:   cv.Address,
			PublicKey: cv.PublicKey,
		}
	})

	// Generate checkpoint onions for the run based on the desired server load from the global configuration
	checkpoints := GetCheckpoints(activeNodes, activeClients)

	// Prepare start signals for each client, including checkpoints.
	clientStartSignals := make(map[structs.PublicNodeApi]structs.ClientStartRunApi)
	for _, client := range activeClients {
		csr := structs.ClientStartRunApi{
			Clients:          activeClients,
			Relays:           activeNodes,
			CheckpointOnions: checkpoints[client.ID],
		}
		clientStartSignals[client] = csr
	}

	// Aggregate all checkpoints by the receiving relay ID.
	allCheckpoints := utils.GroupBy(utils.Flatten(utils.MapMap(checkpoints, func(_ int, cos []structs.CheckpointOnion) []structs.Checkpoint {
		return utils.FlatMap(cos, func(co structs.CheckpointOnion) []structs.Checkpoint {
			return co.Path
		})
	})), func(checkpoint structs.Checkpoint) int {
		return checkpoint.Receiver.ID
	})

	// Prepare start signals for each relay, including all relevant checkpoints.
	nodeStartSignals := make(map[structs.PublicNodeApi]structs.RelayStartRunApi)
	for _, node := range activeNodes {
		nodeStartSignals[node] = structs.RelayStartRunApi{
			Checkpoints: allCheckpoints[node.ID],
		}
	}

	// Synchronize the completion of signaling all nodes.
	var wg sync.WaitGroup
	wg.Add(len(activeNodes) + len(activeClients))

	// Signal all active clients to start the run.
	for client, csr := range clientStartSignals {
		go func(client structs.PublicNodeApi, csr structs.ClientStartRunApi) {
			defer wg.Done()

			data, err := json.Marshal(csr)
			if err != nil {
				slog.Error("Error signaling client to start\n", err)
				return
			}

			// Send the start signal to the client's /start endpoint.
			url := fmt.Sprintf("%s/start", client.Address)
			if resp, err := http.Post(url, "application/json", bytes.NewBuffer(data)); err != nil {
				slog.Error("Error signaling client to start\n", err)
			} else if err := resp.Body.Close(); err != nil {
				slog.Error("Error closing response body", err)
			}

		}(client, csr)
	}

	// Signal all active relays to start the run.
	for relay, nsr := range nodeStartSignals {
		defer wg.Done()
		nsr := nsr
		go func(relay structs.PublicNodeApi, nsr structs.RelayStartRunApi) {
			if data, err := json.Marshal(nsr); err != nil {
				slog.Error("Error signaling relay to start\n", err)
			} else {
				url := fmt.Sprintf("%s/start", relay.Address)
				if resp, err := http.Post(url, "application/json", bytes.NewBuffer(data)); err != nil {
					slog.Error("Error signaling relay to start\n", err)
				} else if err := resp.Body.Close(); err != nil {
					slog.Error("Error closing response body", err)
				}
			}
		}(relay, nsr)
	}

	// Wait for all signaling operations to complete.
	wg.Wait()
	return nil
}

// StartRuns periodically checks if all nodes are ready and, if so, signals them to start a new run.
func (bb *BulletinBoard) StartRuns() error {
	for {
		// Check if the time since the last start run is greater than the required interval.
		if time.Since(bb.lastStartRun) >= bb.timeBetweenRuns {
			bb.lastStartRun = time.Now() // Update the timestamp for the last start run.
			if bb.allNodesReady() {      // Check if all nodes are ready to start.
				if err := bb.signalNodesToStart(); err != nil {
					return pl.WrapError(err, "error signaling nodes to start")
				} else {
					return nil // If successful, exit the loop.
				}
			}
		}

		time.Sleep(time.Second * 5) // Wait 5 seconds before the next check.
	}
}

// allNodesReady checks if all required nodes and clients are active and ready to start a run.
func (bb *BulletinBoard) allNodesReady() bool {
	bb.mu.RLock()
	defer bb.mu.RUnlock()

	// Count the number of active relay nodes.
	activeNodes := utils.CountAny(utils.GetValues(bb.Network), func(node *RelayView) bool {
		return node.IsActive()
	})

	// If the number of active relays is less than required, log and return false.
	if activeNodes < len(config.GlobalConfig.Relays) {
		slog.Info("Not all nodes are registered")
		return false
	}

	// Count the number of clients that have registered intent to send messages.
	registeredClients := utils.CountAny(utils.GetValues(bb.Clients), func(client *ClientView) bool {
		return client.MessageQueue != nil && len(client.MessageQueue) > 0
	})

	// If the number of registered clients is less than required, log and return false.
	if registeredClients < len(config.GlobalConfig.Clients) {
		slog.Info("Not all clients are registered")
		return false
	}

	// If all nodes and clients are ready, log and return true.
	slog.Info("All nodes ready")
	return true
}
