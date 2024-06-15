package repositories

import (
	"errors"
	"time"

	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
)

// NodeRepositoryImpl is the implementation of the NodeRepository interface
type NodeRepositoryImpl struct {
	nodes map[string]models.Node
}

// NewNodeRepository creates a new instance of NodeRepositoryImpl
func NewNodeRepository() *NodeRepositoryImpl {
	return &NodeRepositoryImpl{
		nodes: make(map[string]models.Node),
	}
}

// RegisterNode registers a node in the repository
func (repo *NodeRepositoryImpl) RegisterNode(node *models.Node) error {
	if _, exists := repo.nodes[node.ID]; exists {
		return errors.New("node already exists")
	}
	repo.nodes[node.ID] = *node
	return nil
}

// Heartbeat updates the heartbeat status of a node
func (repo *NodeRepositoryImpl) Heartbeat(nodeID string) error {
	node, exists := repo.nodes[nodeID]
	if !exists {
		return errors.New("node not found")
	}
	// Logic to update node's heartbeat status
	node.LastHeartbeat = time.Now() // Assume Node struct has LastHeartbeat field
	repo.nodes[nodeID] = node
	return nil
}

// GetActiveNodes returns a list of all active nodes
func (repo *NodeRepositoryImpl) GetActiveNodes() ([]models.Node, error) {
	nodes := make([]models.Node, 0, len(repo.nodes))
	for _, node := range repo.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}
