package node

import "time"

// Node represents a node in the onion routing network
type Node struct {
	ID            int
	Host          string
	Port          int
	LastHeartbeat time.Time
	PublicKey     []byte
	PrivateKey    []byte
}

// NewNode creates a new node
func NewNode(id int, host string, port int, publicKey []byte) *Node {
	return &Node{
		ID:            id,
		Host:          host,
		Port:          port,
		LastHeartbeat: time.Now(),
		PublicKey:     publicKey,
	}
}

// UpdateHeartbeat updates the last heartbeat time of the node
func (n *Node) UpdateHeartbeat() {
	n.LastHeartbeat = time.Now()
}

func (n *Node) Receive(o *Onion) error {
	if err := o.RemoveLayer(n.PrivateKey); err != nil {
		return err
	} else if o.HasNextLayer() {

	}
	panic("not implemented")
}
