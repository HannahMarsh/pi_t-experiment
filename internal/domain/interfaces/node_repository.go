package interfaces

import "github.com/HannahMarsh/pi_t-experiment/internal/domain/models"

// NodeRepository defines the methods for interacting with nodes
type NodeRepository interface {
	RegisterNode(node *models.Node) error
	Heartbeat(nodeID string) error
	GetActiveNodes() ([]models.Node, error)
}
