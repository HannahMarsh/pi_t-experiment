package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"time"
)

// BulletinBoard represents the bulletin board that keeps track of active nodes and coordinates the start signal
type BulletinBoard struct {
	ActiveNodes []api.Node
	LastUpdated time.Time
}

// NewBulletinBoard creates a new bulletin board
func NewBulletinBoard() *BulletinBoard {
	return &BulletinBoard{
		ActiveNodes: []api.Node{},
		LastUpdated: time.Now(),
	}
}

// AddNode adds a node to the active nodes list
func (bb *BulletinBoard) AddNode(node api.Node) error {
	bb.ActiveNodes = append(bb.ActiveNodes, node)
	bb.LastUpdated = time.Now()
	return nil
}

// GetActiveNodes returns the list of active nodes
func (bb *BulletinBoard) GetActiveNodes() []api.Node {
	return bb.ActiveNodes
}

func (bb *BulletinBoard) StartRuns() {

}
