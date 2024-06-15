package repositories

import (
	"errors"

	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
)

// OnionRepositoryImpl is the implementation of the OnionRepository interface
type OnionRepositoryImpl struct {
	onions map[string]models.Onion
}

// NewOnionRepository creates a new instance of OnionRepositoryImpl
func NewOnionRepository() *OnionRepositoryImpl {
	return &OnionRepositoryImpl{
		onions: make(map[string]models.Onion),
	}
}

// SaveOnion saves an onion in the repository
func (repo *OnionRepositoryImpl) SaveOnion(onion *models.Onion) error {
	if _, exists := repo.onions[onion.ID]; exists {
		return errors.New("onion already exists")
	}
	repo.onions[onion.ID] = *onion
	return nil
}

// GetOnion retrieves an onion by its ID
func (repo *OnionRepositoryImpl) GetOnion(id string) (*models.Onion, error) {
	onion, exists := repo.onions[id]
	if !exists {
		return nil, errors.New("onion not found")
	}
	return &onion, nil
}
