package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"sync"
	"time"

	"math/rand"
)

// Node represents a node in the onion routing network
type Node struct {
	ID               int
	Host             string
	Port             int
	PublicKey        []byte
	PrivateKey       []byte
	ActiveNodes      []api.PublicNodeApi
	MessageQueue     []*api.Message
	OnionQueue       []*Onion
	mu               sync.Mutex
	NodeInfo         api.PublicNodeApi
	BulletinBoardUrl string
	wg               sync.WaitGroup
	requests         sync.Map
}

// NewNode creates a new node
func NewNode(id int, host string, port int, bulletinBoardUrl string) (*Node, error) {
	if publicKey, privateKey, err := utils.GenerateKeyPair(); err != nil {
		return nil, fmt.Errorf("node.NewNode(): failed to generate key pair: %w", err)
	} else {
		n := &Node{
			ID:           id,
			Host:         host,
			Port:         port,
			PublicKey:    publicKey,
			PrivateKey:   privateKey,
			ActiveNodes:  make([]api.PublicNodeApi, 0),
			MessageQueue: make([]*api.Message, 0),
			OnionQueue:   make([]*Onion, 0),
			NodeInfo: api.PublicNodeApi{
				ID:        id,
				Address:   fmt.Sprintf("%s:%d", host, port),
				PublicKey: publicKey,
			},
			BulletinBoardUrl: bulletinBoardUrl,
			wg:               sync.WaitGroup{},
		}
		if err2 := n.RegisterWithBulletinBoard(); err2 != nil {
			return nil, fmt.Errorf("node.NewNode(): failed to register with bulletin board: %w", err2)
		}

		go n.StartPeriodicUpdates(time.Second * 1)

		return n, nil
	}
}

func (n *Node) StartPeriodicUpdates(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := n.updateBulletinBoard(); err != nil {
				fmt.Printf("Error updating bulletin board: %v\n", err)
				return
			}
			n.ProcessMessageQueue()
		}
	}()
}

func (n *Node) ProcessMessageQueue() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, msg := range n.MessageQueue {
		// Create an onion from the message
		if onion, err := n.NewOnion(msg, 1); err != nil {
			fmt.Printf("Error creating onion: %v\n", err)
		} else {
			n.OnionQueue = append(n.OnionQueue, onion)
		}
	}
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

func (n *Node) NewOnion(msg *api.Message, pathLength int) (*Onion, error) {
	if msg_string, err := json.Marshal(msg); err != nil {
		return nil, fmt.Errorf("NewOnion(): failed to marshal message: %w", err)
	} else {
		if to := n.getNode(msg.To); to != nil {
			if o, err2 := NewOnion(fmt.Sprintf("%s/receive", to.Address), msg_string, to.PublicKey); err2 != nil {
				return nil, fmt.Errorf("NewOnion(): failed to create onion: %w", err2)
			} else {
				for i := 0; i < pathLength; i++ {
					var intermediary *api.PublicNodeApi
					for intermediary.ID == n.ID {
						intermediary = n.getRandomNode()
					}
					if err3 := o.AddLayer(intermediary.Address, intermediary.PublicKey); err3 != nil {
						return nil, fmt.Errorf("NewOnion(): failed to add layer: %w", err3)
					}
				}
				return o, nil
			}
		} else {
			return nil, fmt.Errorf("NewOnion(): failed to get node with id %d", msg.To)
		}
	}
}

func (n *Node) startRun(activeNodes []api.PublicNodeApi) (didParticipate bool, e error) {
	n.mu.Lock()
	if len(activeNodes) == 0 {
		n.mu.Unlock()
		return false, fmt.Errorf("startRun(): no active nodes")
	}
	n.ActiveNodes = activeNodes
	onionsToSend := n.OnionQueue
	n.OnionQueue = make([]*Onion, 0)
	n.mu.Unlock()
	slog.Info("Starting run with", "num_onions", len(onionsToSend))
	n.wg.Wait()
	n.wg.Add(1)
	defer n.wg.Done()

	var participate bool = false
	for _, node := range activeNodes {
		if node.ID == n.ID {
			participate = true
		}
	}
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

func (n *Node) Receive(o *Onion) error {
	if err := o.RemoveLayer(n.PrivateKey); err != nil {
		return fmt.Errorf("node.Receive(): failed to remove layer: %w", err)
	} else if o.HasNextLayer() {
		if err2 := sendToNode(o); err2 != nil {
			return fmt.Errorf("node.Receive(): failed to send to next node: %w", err2)
		}
	} else if o.HasMessage() {
		// Process the final message here
		slog.Info("Received onion with message", "message", o.Message)
		context.TODO()
	} else {
		slog.Info("Received dummy onion")
		context.TODO()
	}
	return nil
}

func sendToNode(o *Onion) error {
	url := fmt.Sprintf("http://%s/receive", o.Address)
	if data, err := json.Marshal(o); err != nil {
		return fmt.Errorf("sendToNode(): failed to marshal onion: %w", err)
	} else if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
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
