package usecases

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/domain/interfaces"
	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
	"log"
	"time"
)

// BulletinBoardService handles the logic for managing the bulletin board
type BulletinBoardService struct {
	Repo     interfaces.BulletinBoardRepository
	Interval time.Duration
}

// RegisterNode registers a new node with the bulletin board
func (s *BulletinBoardService) RegisterNode(node *models.Node) error {
	return s.Repo.RegisterNode(node)
}

// GetActiveNodes retrieves the list of active nodes from the bulletin board
func (s *BulletinBoardService) GetActiveNodes() ([]models.Node, error) {
	return s.Repo.GetActiveNodes()
}

// BroadcastStartSignal broadcasts a start signal to all active nodes
func (s *BulletinBoardService) BroadcastStartSignal() error {
	nodes, err := s.Repo.GetActiveNodes()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		// Placeholder for logic to send start signal to each node
		// This might involve sending an HTTP request or a message via a queue system
		log.Printf("Sending start signal to node %s at %s:%d", node.ID, node.Host, node.Port)
	}
	return nil
}

// StartRuns begins the periodic broadcasting of start signals to nodes
func (s *BulletinBoardService) StartRuns() {
	ticker := time.NewTicker(s.Interval)
	for {
		select {
		case <-ticker.C:
			err := s.BroadcastStartSignal()
			if err != nil {
				log.Printf("Error broadcasting start signal: %v", err)
			}
		}
	}
}
