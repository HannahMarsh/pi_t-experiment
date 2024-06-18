package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/enriquebris/goconcurrentqueue"
	"github.com/orcaman/concurrent-map/v2"
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
	OtherNodes       cmap.ConcurrentMap[string, *api.Node]
	QueuedOnions     goconcurrentqueue.Queue
	NodeInfo         api.Node
	BulletinBoardUrl string
	wg               sync.WaitGroup
}

// NewNode creates a new node
func NewNode(id int, host string, port int, publicKey []byte, privateKey []byte, bulletinBoardUrl string) *Node {
	return &Node{
		ID:           id,
		Host:         host,
		Port:         port,
		PublicKey:    publicKey,
		PrivateKey:   privateKey,
		OtherNodes:   cmap.New[*api.Node](),
		QueuedOnions: goconcurrentqueue.NewFIFO(),
		NodeInfo: api.Node{
			ID:        id,
			Address:   fmt.Sprintf("%s:%d", host, port),
			PublicKey: publicKey,
		},
		BulletinBoardUrl: bulletinBoardUrl,
		wg:               sync.WaitGroup{},
	}
}

func (n *Node) updateNode(node *api.Node) {
	n.OtherNodes.Set(node.Address, node)
}

func (n *Node) startRun(activeNodes []api.Node) (bool, error) {
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
				return true, err
			} else if on, ok := o.(*Onion); !ok {
				return true, fmt.Errorf("invalid onion type: %T", o)
			} else if err2 := sendToNode(on); err2 != nil {
				return true, err2
			}
		}
		return true, nil
	}
	return false, nil
}

func (n *Node) Receive(o *Onion) error {
	if err := o.RemoveLayer(n.PrivateKey); err != nil {
		return err
	} else if o.HasNextLayer() {
		if _, present := n.OtherNodes.Get(o.Address); !present {
			return fmt.Errorf("next node not found: %s", o.Address)
		} else if err2 := sendToNode(o); err2 != nil {
			return fmt.Errorf("failed to send to next node: %v", err2)
		}
	} else if o.HasMessage() {
		// Process the final message here
	} else {
		return fmt.Errorf("invalid onion: no data or message")
	}
	return nil
}

func sendToNode(o *Onion) error {
	url := fmt.Sprintf("http://%s/receive", o.Address)
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err2 := Body.Close()
		if err2 != nil {
			fmt.Printf("Error closing response body: %v\n", err2)
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send to next node, status code: %d", resp.StatusCode)
	}
	return nil
}
