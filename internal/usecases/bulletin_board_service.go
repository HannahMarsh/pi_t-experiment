package usecases

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/domain/interfaces"
	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
	"time"
)

type BulletinBoardService struct {
	repo     interfaces.BulletinBoardRepository
	interval time.Duration
}

func (s *BulletinBoardService) RegisterNode(node *models.Node) error {
	return s.repo.RegisterNode(node)
}

func (s *BulletinBoardService) GetActiveNodes() ([]models.Node, error) {
	return s.repo.GetActiveNodes()
}

func (s *BulletinBoardService) BroadcastStartSignal() error {
	nodes, err := s.repo.GetActiveNodes()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		// Logic to send start signal to each node
	}
	return nil
}

func (s *BulletinBoardService) StartRuns() {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ticker.C:
			err := s.BroadcastStartSignal()
			if err != nil {
				// Handle error
			}
		}
	}
}
