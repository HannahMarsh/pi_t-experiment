package usecases

import (
	"time"

	"github.com/HannahMarsh/pi_t-experiment/internal/domain/interfaces"
	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
)

type NodeService struct {
	Repo     interfaces.NodeRepository
	Interval time.Duration
}

func (s *NodeService) Receive(o *models.Onion) error {
	panic("not implemented")
}

func (s *NodeService) Heartbeat(nodeID string) error {
	return s.Repo.Heartbeat(nodeID)
}

func (s *NodeService) StartActions() {
	ticker := time.NewTicker(s.Interval)
	for {
		select {
		case <-ticker.C:
			// Logic to start node actions for each run
		}
	}
}
