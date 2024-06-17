package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/node"
	"time"
)

// BulletinBoardRepository defines the methods for interacting with the bulletin board
type BulletinBoardRepository interface {
	RegisterNode(node *node.Node) error
	GetActiveNodes() ([]node.Node, error)
	BroadcastStartSignal() error
}

// BulletinBoard represents the bulletin board that keeps track of active nodes and coordinates the start signal
type BulletinBoard struct {
	ID          string
	ActiveNodes []node.Node
	LastUpdated time.Time
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard(id string) *BulletinBoard {
	return &BulletinBoard{
		ID:          id,
		ActiveNodes: []node.Node{},
		LastUpdated: time.Now(),
	}
}

// AddNode adds a node to the active nodes list
func (bb *BulletinBoard) AddNode(node node.Node) {
	bb.ActiveNodes = append(bb.ActiveNodes, node)
	bb.LastUpdated = time.Now()
}

// GetActiveNodes returns the list of active nodes
func (bb *BulletinBoard) GetActiveNodes() []node.Node {
	return bb.ActiveNodes
}
