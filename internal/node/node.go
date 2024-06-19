package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"sync"
	"time"

	rng "math/rand"
)

var rand = rng.New(rng.NewSource(time.Now().UnixNano()))

// Node represents a node in the onion routing network
type Node struct {
	ID               int
	Host             string
	Port             int
	PublicKey        string
	PrivateKey       string
	ActiveNodes      []api.PublicNodeApi
	OnionQueue       *utils.SafeHeap[QueuedOnion]
	mu               sync.RWMutex
	BulletinBoardUrl string
	wg               sync.WaitGroup
	requests         sync.Map
	lastUpdate       time.Time
}

type QueuedOnion struct {
	ConstructedOnion   string
	DestinationAddress string
	OriginalMessage    api.Message
	TimeReceived       time.Time
}

func qoLess(a, b QueuedOnion) bool {
	return a.TimeReceived.Before(b.TimeReceived)
}

// NewNode creates a new node
func NewNode(id int, host string, port int, bulletinBoardUrl string) (*Node, error) {
	if publicKey, privateKey, err := pi_t.KeyGen(); err != nil {
		return nil, fmt.Errorf("node.NewNode(): failed to generate key pair: %w", err)
	} else {
		n := &Node{
			ID:               id,
			Host:             host,
			Port:             port,
			PublicKey:        publicKey,
			PrivateKey:       privateKey,
			ActiveNodes:      make([]api.PublicNodeApi, 0),
			OnionQueue:       utils.NewSafeHeap(qoLess),
			BulletinBoardUrl: bulletinBoardUrl,
			wg:               sync.WaitGroup{},
		}
		if err2 := n.RegisterWithBulletinBoard(); err2 != nil {
			return nil, fmt.Errorf("node.NewNode(): failed to register with bulletin board: %w", err2)
		}

		go n.StartPeriodicUpdates(time.Second * 3)

		return n, nil
	}
}

func (n *Node) getPublicNodeInfo() api.PublicNodeApi {
	return api.PublicNodeApi{
		ID:        n.ID,
		Address:   fmt.Sprintf("http://%s:%d", n.Host, n.Port),
		PublicKey: n.PublicKey,
	}
}

func (n *Node) getPrivateNodeInfo(timeOfRequest time.Time) api.PrivateNodeApi {
	mq := n.OnionQueue.MapToInt(func(qo QueuedOnion) int {
		return qo.OriginalMessage.To
	})
	return api.PrivateNodeApi{
		TimeOfRequest: timeOfRequest,
		ID:            n.ID,
		Address:       fmt.Sprintf("http://%s:%d", n.Host, n.Port),
		PublicKey:     n.PublicKey,
		MessageQueue:  mq,
	}
}

func (n *Node) StartPeriodicUpdates(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := n.updateBulletinBoard("/update", http.StatusOK); err != nil {
				fmt.Printf("Error updating bulletin board: %v\n", err)
				return
			} else if activeNodes, err2 := n.GetActiveNodes(); err2 != nil {
				fmt.Printf("Error getting active nodes: %v\n", err2)
				return
			} else {
				n.mu.Lock()
				n.ActiveNodes = utils.Copy(activeNodes)
				n.mu.Unlock()
			}
		}
	}()
}

func (n *Node) getNode(id int) *api.PublicNodeApi {
	for _, node := range n.ActiveNodes {
		if node.ID == id {
			return &node
		}
	}
	return nil
}

func (n *Node) getRandomNode() *api.PublicNodeApi {
	r := rand.Intn(len(n.ActiveNodes))
	return &n.ActiveNodes[r]
}

func (n *Node) QueueOnion(msg api.Message, pathLength int) error {
	timeReceived := time.Now()
	if msgString, err := json.Marshal(msg); err != nil {
		return fmt.Errorf("NewOnion(): failed to marshal message: %w", err)
	} else if to := n.getNode(msg.To); to == nil {
		return fmt.Errorf("NewOnion(): failed to get node with id %d", msg.To)
	} else if routingPath, err2 := n.DetermineRoutingPath(pathLength); err2 != nil {
		return fmt.Errorf("NewOnion(): failed to determine routing path: %w", err2)
	} else {
		publicKeys := utils.Map(routingPath, func(node api.PublicNodeApi) string {
			return node.PublicKey
		})
		addresses := utils.Map(routingPath, func(node api.PublicNodeApi) string {
			return node.Address
		})
		if addr, onion, err3 := pi_t.FormOnion(msgString, publicKeys, addresses); err3 != nil {
			return fmt.Errorf("NewOnion(): failed to create onion: %w", err3)
		} else {
			qo := QueuedOnion{
				ConstructedOnion:   onion,
				DestinationAddress: addr,
				OriginalMessage:    msg,
				TimeReceived:       timeReceived,
			}
			n.OnionQueue.Push(qo)
			return nil
		}
	}
}

// DetermineRoutingPath determines a random routing path of a given length
func (n *Node) DetermineRoutingPath(pathLength int) ([]api.PublicNodeApi, error) {
	if len(n.ActiveNodes) < pathLength {
		return nil, errors.New("not enough nodes to form a path")
	}

	selectedNodes := make([]api.PublicNodeApi, pathLength)
	perm := rand.Perm(len(n.ActiveNodes))

	for i := 0; i < pathLength; i++ {
		selectedNodes[i] = n.ActiveNodes[perm[i]]
	}

	return selectedNodes, nil
}

func (n *Node) IDsMatch(nodeApi api.PublicNodeApi) bool {
	return n.ID == nodeApi.ID
}

func (n *Node) startRun(activeNodes []api.PublicNodeApi) (didParticipate bool, e error) {
	n.wg.Wait()
	n.wg.Add(1)
	defer n.wg.Done()

	n.mu.Lock()
	if len(activeNodes) == 0 {
		n.mu.Unlock()
		return false, fmt.Errorf("startRun(): no active nodes")
	}
	n.ActiveNodes = utils.Copy(activeNodes)
	onionsToSend := n.OnionQueue.Drain()
	n.mu.Unlock()

	slog.Info("Starting run with", "num_onions", len(onionsToSend))

	participate := utils.Contains(activeNodes, n.IDsMatch)

	if participate {
		for _, onion := range onionsToSend {
			if err2 := sendToNode(onion); err2 != nil {
				return true, fmt.Errorf("startRun(): failed to send onion to next node: %w", err2)
			}
		}
		return true, nil
	}
	return false, nil
}

func (n *Node) Receive(o string) error {
	if destination, payload, err := pi_t.PeelOnion(o, n.PrivateKey); err != nil {
		return fmt.Errorf("node.Receive(): failed to remove layer: %w", err)
	} else {
		bruised, err2 := pi_t.BruiseOnion(payload)
		if err2 != nil {
			return fmt.Errorf("node.Receive(): failed to bruise onion: %w", err2)
		}
		if err3 := sendToNode(QueuedOnion{
			ConstructedOnion:   bruised,
			DestinationAddress: destination,
		}); err != nil {
			return fmt.Errorf("node.Receive(): failed to send to next node: %w", err3)
		}
	}
	return nil
}

func sendToNode(onion QueuedOnion) error {
	url := fmt.Sprintf("http://%s/receive", onion.DestinationAddress)
	if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer([]byte(onion.ConstructedOnion))); err2 != nil {
		return fmt.Errorf("sendToNode(): failed to send POST request with onion to next node: %w", err2)
	} else {
		defer func(Body io.ReadCloser) {
			if err3 := Body.Close(); err3 != nil {
				slog.Error("sendToNode(): Error closing response body", err3)
			}
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("sendToNode(): Failed to send to next node, status code: %d, status: %s", resp.StatusCode, resp.Status)
		}
		return nil
	}
}
