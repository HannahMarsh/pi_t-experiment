package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/metrics"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/cm"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"io"
	"net/http"
	"sync"
	"time"

	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"log/slog"
)

// Node represents a node in the onion routing network
type Node struct {
	ID                  int
	Host                string
	Port                int
	Address             string
	PrivateKey          string
	PublicKey           string
	mu                  sync.RWMutex
	BulletinBoardUrl    string
	lastUpdate          time.Time
	status              *structs.NodeStatus
	checkpointsReceived *cm.ConcurrentMap[int, int]
	expectedNonces      []map[string]bool
	isCorrupted         bool
	wg                  sync.WaitGroup
}

// NewNode creates a new node
func NewNode(id int, host string, port int, bulletinBoardUrl string) (*Node, error) {
	if privateKey, publicKey, err := keys.KeyGen(); err != nil {
		return nil, pl.WrapError(err, "node.NewClient(): failed to generate key pair")
	} else {
		expectedCheckpoints := make([]map[string]bool, config.GlobalConfig.L1+config.GlobalConfig.L2+1)
		for i := range expectedCheckpoints {
			expectedCheckpoints[i] = make(map[string]bool)
		}

		// determine if node is corrupted
		numCorrupted := int(config.GlobalConfig.Chi * float64(len(config.GlobalConfig.Nodes)))
		corruptedNodes := utils.PseudoRandomSubset(config.GlobalConfig.Nodes, numCorrupted, 42)
		isCorrupted := utils.Contains(corruptedNodes, func(node config.Node) bool {
			return node.ID == id
		})

		n := &Node{
			ID:                  id,
			Host:                host,
			Address:             fmt.Sprintf("http://%s:%d", host, port),
			Port:                port,
			PublicKey:           publicKey,
			PrivateKey:          privateKey,
			BulletinBoardUrl:    bulletinBoardUrl,
			status:              structs.NewNodeStatus(id, fmt.Sprintf("http://%s:%d", host, port), publicKey),
			checkpointsReceived: &cm.ConcurrentMap[int, int]{},
			expectedNonces:      expectedCheckpoints,
			isCorrupted:         isCorrupted,
		}
		n.wg.Add(1)
		if err2 := n.RegisterWithBulletinBoard(); err2 != nil {
			return nil, pl.WrapError(err2, "node.NewNode(): failed to register with bulletin board")
		}

		go n.StartPeriodicUpdates(time.Second * 3)

		return n, nil
	}
}

func (n *Node) GetStatus() string {
	return n.status.GetStatus()
}

func (n *Node) getPublicNodeInfo() structs.PublicNodeApi {
	return structs.PublicNodeApi{
		ID:        n.ID,
		Address:   n.Address,
		PublicKey: n.PublicKey,
		Time:      time.Now(),
	}
}

func (n *Node) StartPeriodicUpdates(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			//slog.Info("Updating bulletin board")
			if err := n.updateBulletinBoard("/updateNode", http.StatusOK); err != nil {
				fmt.Printf("Error updating bulletin board: %v\n", err)
				return
			}
		}
	}()
}

func (n *Node) startRun(start structs.NodeStartRunApi) (didParticipate bool, e error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	defer n.wg.Done()

	for _, c := range start.Checkpoints {
		n.expectedNonces[c.Layer][c.Nonce] = true
		n.status.AddExpectedCheckpoint(c.Layer)
	}

	return true, nil
}

func (n *Node) Receive(oApi structs.OnionApi) error {
	n.wg.Wait()

	timeReceived := time.Now()

	role, layer, metadata, peeled, nextHop, err := pi_t.PeelOnion(oApi.Onion, n.PrivateKey)
	if err != nil {
		return pl.WrapError(err, "node.Receive(): failed to remove layer")
	}

	wasBruised := false
	isCheckpoint := false

	if metadata.Nonce != "" {
		isCheckpoint = true
		if _, present := n.expectedNonces[layer][metadata.Nonce]; present { // nonce is verified
			n.checkpointsReceived.GetAndSet(layer, func(i int) int {
				return i + 1
			})
			if role == onion_model.MIXER {
				slog.Debug("Mixer: Nonce was verified, dropping null block.")
				peeled.Sepal = peeled.Sepal.RemoveBlock()
			}
		} else { // nonce is not verified
			if role == onion_model.MIXER {
				slog.Debug("Mixer: Nonce was not verified, dropping master key.")
				peeled.Sepal = peeled.Sepal.AddBruise()
				wasBruised = true
			}
		}

		n.status.AddCheckpointOnion(layer)
	} else if role == onion_model.MIXER {
		peeled.Sepal = peeled.Sepal.RemoveBlock()
	}

	slog.Info("Received onion", "ischeckpoint?", metadata.Nonce != "", "layer", layer, "nextHop", config.AddressToName(nextHop))

	n.status.AddOnion(oApi.From, n.Address, nextHop, layer, isCheckpoint, !wasBruised)

	metrics.Observe(metrics.PROCESSING_TIME, time.Since(timeReceived).Seconds())
	metrics.Inc(metrics.ONION_COUNT, layer)

	n.sendToNode(nextHop, peeled)

	return nil
}

func (n *Node) sendToNode(addr string, constructedOnion onion_model.Onion) {
	go func(addr string, constructedOnion onion_model.Onion) {
		err := api_functions.SendOnion(addr, n.Address, constructedOnion)
		if err != nil {
			slog.Error("Error sending onion", err)
		}
	}(addr, constructedOnion)
}

func (n *Node) RegisterWithBulletinBoard() error {
	slog.Info("Sending node registration request.", "id", n.ID)
	return n.updateBulletinBoard("/registerNode", http.StatusCreated)
}

func (n *Node) GetActiveNodes() ([]structs.PublicNodeApi, error) {
	url := fmt.Sprintf("%s/nodes", n.BulletinBoardUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, pl.WrapError(err, fmt.Sprintf("error making GET request to %s", url))
	}
	defer func(Body io.ReadCloser) {
		if err2 := Body.Close(); err2 != nil {
			fmt.Printf("error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, pl.NewError("unexpected status code: %d", resp.StatusCode)
	}

	var activeNodes []structs.PublicNodeApi
	if err = json.NewDecoder(resp.Body).Decode(&activeNodes); err != nil {
		return nil, pl.WrapError(err, "error decoding response body")
	}

	return activeNodes, nil
}

func (n *Node) updateBulletinBoard(endpoint string, expectedStatusCode int) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	t := time.Now()
	if data, err := json.Marshal(structs.PublicNodeApi{
		ID:        n.ID,
		Address:   n.Address,
		PublicKey: n.PublicKey,
		Time:      t,
	}); err != nil {
		return pl.WrapError(err, "node.UpdateBulletinBoard(): failed to marshal node info")
	} else {
		url := n.BulletinBoardUrl + endpoint
		//slog.Info("Sending node registration request.", "url", url, "id", n.ID)
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return pl.WrapError(err2, "node.UpdateBulletinBoard(): failed to send POST request to bulletin board")
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					fmt.Printf("node.UpdateBulletinBoard(): error closing response body: %v\n", err2)
				}
			}(resp.Body)
			if resp.StatusCode != expectedStatusCode {
				return pl.NewError("failed to %s node, status code: %d, %s", endpoint, resp.StatusCode, resp.Status)
			} else {
				n.lastUpdate = t
			}
			return nil
		}
	}
}
