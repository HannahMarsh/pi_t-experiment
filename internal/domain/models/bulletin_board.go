package models

import "time"

// BulletinBoardRepository defines the methods for interacting with the bulletin board
type BulletinBoardRepository interface {
	RegisterNode(node *Node) error
	GetActiveNodes() ([]Node, error)
	BroadcastStartSignal() error
}

// BulletinBoard represents the bulletin board that keeps track of active nodes and coordinates the start signal
type BulletinBoard struct {
	ID          string
	ActiveNodes []Node
	LastUpdated time.Time
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard(id string) *BulletinBoard {
	return &BulletinBoard{
		ID:          id,
		ActiveNodes: []Node{},
		LastUpdated: time.Now(),
	}
}

// AddNode adds a node to the active nodes list
func (bb *BulletinBoard) AddNode(node Node) {
	bb.ActiveNodes = append(bb.ActiveNodes, node)
	bb.LastUpdated = time.Now()
}

// GetActiveNodes returns the list of active nodes
func (bb *BulletinBoard) GetActiveNodes() []Node {
	return bb.ActiveNodes
}
