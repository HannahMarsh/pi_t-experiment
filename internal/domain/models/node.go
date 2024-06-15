package models

import "time"

// Node represents a node in the onion routing network
type Node struct {
	ID            string
	Host          string
	Port          int
	LastHeartbeat time.Time
}

// NewNode creates a new node
func NewNode(id, host string, port int) *Node {
	return &Node{
		ID:            id,
		Host:          host,
		Port:          port,
		LastHeartbeat: time.Now(),
	}
}

// UpdateHeartbeat updates the last heartbeat time of the node
func (n *Node) UpdateHeartbeat() {
	n.LastHeartbeat = time.Now()
}
