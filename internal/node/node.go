package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"

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
	if privateKey, publicKey, err := pi_t.KeyGen(); err != nil {
		return nil, PrettyLogger.WrapError(err, "node.NewNode(): failed to generate key pair")
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
			return nil, PrettyLogger.WrapError(err2, "node.NewNode(): failed to register with bulletin board")
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
	n.mu.Lock()
	defer n.mu.Unlock()
	if msgString, err := json.Marshal(msg); err != nil {
		return PrettyLogger.WrapError(err, "failed to marshal message")
	} else if to := n.getNode(msg.To); to == nil {
		return PrettyLogger.NewError("QueueOnion(): failed to get node with id %d", msg.To)
	} else if routingPath, err2 := n.DetermineRoutingPath(pathLength); err2 != nil {
		return PrettyLogger.WrapError(err2, "failed to determine routing path")
	} else {
		publicKeys := utils.Map(routingPath, func(node api.PublicNodeApi) string {
			return node.PublicKey
		})
		addresses := utils.Map(routingPath, func(node api.PublicNodeApi) string {
			return node.Address
		})
		if addr, onion, err3 := pi_t.FormOnion(msgString, publicKeys, addresses); err3 != nil {
			return PrettyLogger.WrapError(err3, "failed to create onion")
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
	//n.wg.Wait()
	//n.wg.Add(1)
	//defer n.wg.Done()

	n.mu.Lock()
	if len(activeNodes) == 0 {
		n.mu.Unlock()
		return false, PrettyLogger.NewError("no active nodes")
	}
	n.ActiveNodes = utils.Copy(activeNodes)
	onionsToSend := n.OnionQueue.Values()
	n.OnionQueue.Clear()
	n.mu.Unlock()

	slog.Info("Starting run with", "num_onions", len(onionsToSend))

	participate := utils.Contains(activeNodes, n.IDsMatch)

	if participate {
		for _, onion := range onionsToSend {
			if err2 := sendToNode(onion); err2 != nil {
				return true, PrettyLogger.WrapError(err2, "failed to send onion to next node")
			}
		}
		return true, nil
	}
	return false, nil
}

func (n *Node) Receive(o string) error {
	if destination, payload, err := pi_t.PeelOnion(o, n.PrivateKey); err != nil {
		return PrettyLogger.WrapError(err, "node.Receive(): failed to remove layer")
	} else {
		if destination == "" {
			var msg api.Message
			if err2 := json.Unmarshal([]byte(payload), &msg); err2 != nil {
				return PrettyLogger.WrapError(err2, "node.Receive(): failed to unmarshal message")
			}
			slog.Info("Received message", "from", msg.From, "to", msg.To, "msg", msg.Msg)

		} else {
			slog.Info("Received onion", "destination", destination)
			//bruised, err2 := pi_t.BruiseOnion(payload)
			//if err2 != nil {
			//	return PrettyLogger.WrapError(err2, "node.Receive(): failed to bruise onion")
			//}
			if err3 := sendToNode(QueuedOnion{
				ConstructedOnion:   payload,
				DestinationAddress: destination,
			}); err != nil {
				return PrettyLogger.WrapError(err3, "node.Receive(): failed to send to next node")
			}
		}
	}
	return nil
}

func sendToNode(onion QueuedOnion) error {
	url := fmt.Sprintf("%s/receive", onion.DestinationAddress)
	o := api.OnionApi{
		Onion: onion.ConstructedOnion,
	}
	if data, err := json.Marshal(o); err != nil {
		slog.Error("failed to marshal msgs", err)
	} else if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
		return PrettyLogger.WrapError(err2, "failed to send POST request with onion to next node")
	} else {
		defer func(Body io.ReadCloser) {
			if err3 := Body.Close(); err3 != nil {
				slog.Error("sendToNode(): Error closing response body", err3)
			}
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return PrettyLogger.NewError("sendToNode(): Failed to send to next node, status code: %d, status: %s", resp.StatusCode, resp.Status)
		}
	}
	return nil
}
