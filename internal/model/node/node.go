package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"io"
	"net/http"
	"sync"
	"time"

	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"golang.org/x/exp/slog"
)

// Node represents a node in the onion routing network
type Node struct {
	ID                  int
	Host                string
	Port                int
	PrivateKey          string
	PublicKey           string
	isMixer             bool
	mu                  sync.RWMutex
	BulletinBoardUrl    string
	lastUpdate          time.Time
	status              *structs.NodeStatus
	checkpoints         map[int]int
	expectedCheckpoints [][]int
}

// NewNode creates a new node
func NewNode(id int, host string, port int, bulletinBoardUrl string, isMixer bool) (*Node, error) {
	if privateKey, publicKey, err := keys.KeyGen(); err != nil {
		return nil, pl.WrapError(err, "node.NewClient(): failed to generate key pair")
	} else {
		n := &Node{
			ID:                  id,
			Host:                host,
			Port:                port,
			PublicKey:           publicKey,
			PrivateKey:          privateKey,
			BulletinBoardUrl:    bulletinBoardUrl,
			isMixer:             isMixer,
			status:              structs.NewNodeStatus(id, fmt.Sprintf("http://%s:%d", host, port), publicKey, isMixer),
			checkpoints:         make(map[int]int),
			expectedCheckpoints: make([][]int, 0),
		}
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
		Address:   fmt.Sprintf("http://%s:%d", n.Host, n.Port),
		PublicKey: n.PublicKey,
		Time:      time.Now(),
		IsMixer:   n.isMixer,
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

	n.expectedCheckpoints = start.Checkpoints

	for _, c := range n.checkpoints {
		n.status.AddExpectedCheckpoint(c)
	}

	//n.wg.Wait()
	//n.wg.Add(1)
	//defer n.wg.Done()
	return true, nil
}

func (n *Node) Receive(from string, o string, sharedKey [32]byte) error {
	layer, metadata, peeled, nextHop, err := pi_t.PeelOnion(o, sharedKey)
	if err != nil {
		return pl.WrapError(err, "node.Receive(): failed to remove layer")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if metadata.Nonce != -1 {
		if utils.Contains(n.expectedCheckpoints[layer], func(i int) bool {
			return i == metadata.Nonce
		}) { // nonce is verified
			n.checkpoints[layer]++
			if n.isMixer {
				peeled.Sepal.RemoveBlock()
			}
		} else {
			if n.isMixer {
				peeled.Sepal.AddBruise()
			}
		}

		n.status.AddCheckpointOnion(layer)
	}

	slog.Info("Received onion", "ischeckpoint?", metadata.Nonce != -1, "layer", layer, "nextHop", config.AddressToName(nextHop))

	n.status.AddOnion(from, fmt.Sprintf("http://%s:%d", n.Host, n.Port), nextHop, layer, metadata.Nonce != -1)

	if err3 := n.sendToNode(nextHop, peeled); err != nil {
		return pl.WrapError(err3, "node.Receive(): failed to send to next node")
	}

	return nil
}

func (n *Node) sendToNode(addr string, constructedOnion onion_model.Onion) error {
	go func() {
		err := api_functions.SendOnion(addr, fmt.Sprintf("http://%s:%d", n.Host, n.Port), constructedOnion)
		if err != nil {
			slog.Error("Error sending onion", err)
		}
	}()
	return nil
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
		Address:   fmt.Sprintf("http://%s:%d", n.Host, n.Port),
		PublicKey: n.PublicKey,
		IsMixer:   n.isMixer,
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
