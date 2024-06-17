package node

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/orcaman/concurrent-map/v2"
)

// Node represents a node in the onion routing network
type Node struct {
	ID         int
	Host       string
	Port       int
	PublicKey  []byte
	PrivateKey []byte
	OtherNodes cmap.ConcurrentMap[string, api.Node]
}

// NewNode creates a new node
func NewNode(id int, host string, port int, publicKey []byte, privateKey []byte) *Node {
	return &Node{
		ID:         id,
		Host:       host,
		Port:       port,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		OtherNodes: cmap.New[api.Node](),
	}
}

func (n *Node) updateNode(node api.Node) {
	n.OtherNodes.Set(node.Address, node)
}

func (n *Node) startRun(activeNodes []api.Node) {
	n.OtherNodes.Clear()
	var participate bool = false
	for _, node := range activeNodes {
		n.updateNode(node)
		if node.ID == n.ID {
			participate = true
		}
	}
	if participate {

	}
}

func (n *Node) Receive(o *Onion) error {
	if err := o.RemoveLayer(n.PrivateKey); err != nil {
		return err
	} else if o.HasNextLayer() {

	}
	panic("not implemented")
}
