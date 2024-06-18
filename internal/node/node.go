package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"github.com/enriquebris/goconcurrentqueue"
	"github.com/orcaman/concurrent-map/v2"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"sync"
)

// Node represents a node in the onion routing network
type Node struct {
	ID               int
	Host             string
	Port             int
	PublicKey        []byte
	PrivateKey       []byte
	OtherNodes       cmap.ConcurrentMap[string, *api.PublicNodeApi]
	QueuedOnions     goconcurrentqueue.Queue
	NodeInfo         api.PublicNodeApi
	BulletinBoardUrl string
	wg               sync.WaitGroup
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
			OtherNodes:   cmap.New[*api.PublicNodeApi](),
			QueuedOnions: goconcurrentqueue.NewFIFO(),
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

		return n, nil
	}
}

func (n *Node) updateNode(node *api.PublicNodeApi) {
	n.OtherNodes.Set(node.Address, node)
}

func (n *Node) startRun(activeNodes []api.PublicNodeApi) (didParticipate bool, e error) {
	n.wg.Wait()
	n.wg.Add(1)
	defer n.wg.Done()

	n.OtherNodes.Clear()
	var participate bool = false
	for _, node := range activeNodes {
		n.updateNode(&node)
		if node.ID == n.ID {
			participate = true
		}
	}
	if participate {
		for {
			if o, err := n.QueuedOnions.Dequeue(); o == nil || err != nil {
				if err.Error() == "empty queue" {
					break
				}
				return true, fmt.Errorf("startRun(): failed to dequeue onion: %w", err)
			} else if on, ok := o.(*Onion); !ok {
				return true, fmt.Errorf("startRun(): invalid onion type: %T", o)
			} else if err2 := sendToNode(on); err2 != nil {
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
		if _, present := n.OtherNodes.Get(o.Address); !present {
			return fmt.Errorf("node.Receive(): next node not found: %s", o.Address)
		} else if err2 := sendToNode(o); err2 != nil {
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
