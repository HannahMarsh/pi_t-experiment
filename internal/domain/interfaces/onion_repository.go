package interfaces

import "github.com/HannahMarsh/pi_t-experiment/internal/domain/models"

// OnionRepository defines the methods for interacting with onions
type OnionRepository interface {
	SaveOnion(onion *models.Onion) error
	GetOnion(id string) (*models.Onion, error)
}
