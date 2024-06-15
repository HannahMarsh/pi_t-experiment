package interfaces

import "github.com/HannahMarsh/pi_t-experiment/internal/domain/models"

// BulletinBoardRepository defines the methods for interacting with the bulletin board
type BulletinBoardRepository interface {
	RegisterNode(node *models.Node) error
	GetActiveNodes() ([]models.Node, error)
	BroadcastStartSignal() error
}
