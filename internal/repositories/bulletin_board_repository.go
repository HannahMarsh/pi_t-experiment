package repositories

import (
	"errors"
	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
)

// BulletinBoardRepositoryImpl is the implementation of the BulletinBoardRepository interface
type BulletinBoardRepositoryImpl struct {
	// In a real-world scenario, this could be a connection to a database
	activeNodes map[string]models.Node
}

// NewBulletinBoardRepository creates a new instance of BulletinBoardRepositoryImpl
func NewBulletinBoardRepository() *BulletinBoardRepositoryImpl {
	return &BulletinBoardRepositoryImpl{
		activeNodes: make(map[string]models.Node),
	}
}

// RegisterNode adds a node to the active nodes list
func (repo *BulletinBoardRepositoryImpl) RegisterNode(node *models.Node) error {
	if _, exists := repo.activeNodes[node.ID]; exists {
		return errors.New("node already registered")
	}
	repo.activeNodes[node.ID] = *node
	return nil
}

// GetActiveNodes returns the list of active nodes
func (repo *BulletinBoardRepositoryImpl) GetActiveNodes() ([]models.Node, error) {
	nodes := make([]models.Node, 0, len(repo.activeNodes))
	for _, node := range repo.activeNodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// BroadcastStartSignal is a placeholder for broadcasting start signals to nodes
func (repo *BulletinBoardRepositoryImpl) BroadcastStartSignal() error {
	// Logic to broadcast start signal to all active nodes
	return nil
}
