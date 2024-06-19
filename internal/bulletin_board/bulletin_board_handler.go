package bulletin_board

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"net/http"
)

func (bb *BulletinBoard) HandleRegisterNode(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received node registration request")
	var node api.PrivateNodeApi
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		slog.Error("Error decoding node registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Registering node with", "id", node.ID)
	if err := bb.UpdateNode(&node); err != nil {
		slog.Error("Error updating node", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) HandleUpdateNodeInfo(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received node info update request")
	var nodeInfo api.PrivateNodeApi
	if err := json.NewDecoder(r.Body).Decode(&nodeInfo); err != nil {
		slog.Error("Error decoding node info update request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Updating node with", "id", nodeInfo.ID)
	if err := bb.UpdateNode(&nodeInfo); err != nil {
		fmt.Printf("Error updating node %d: %v\n", nodeInfo.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleGetActiveNodes handles GET requests to return all active nodes
func (bb *BulletinBoard) HandleGetActiveNodes(w http.ResponseWriter, r *http.Request) {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	activeNodes := utils.NewMapStream(bb.Network).Filter(func(_ int, node *NodeView) bool {
		return node.IsActive()
	}).GetValues().Array

	activeNodesApis := utils.Map(activeNodes, func(node *NodeView) api.PublicNodeApi {
		return node.Api
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(activeNodesApis); err != nil {
		slog.Error("Error encoding response", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (bb *BulletinBoard) signalNodesToStart() error {
	slog.Info("Signaling nodes to start")
	activeNodes := utils.NewMapStream(bb.Network).Filter(func(_ int, node *NodeView) bool {
		return node.IsActive()
	}).GetValues().Array

	activeNodesApis := utils.Map(activeNodes, func(node *NodeView) api.PublicNodeApi {
		return node.Api
	})

	if data, err := json.Marshal(activeNodesApis); err != nil {
		return fmt.Errorf("failed to marshal activeNodes: %w", err)
	} else {
		for _, node := range activeNodes {
			node := node
			go func() {
				url := fmt.Sprintf("%s/start", node.Address)
				if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
					fmt.Printf("Error signaling node %d to start: %v\n", node.ID, err2)
				} else if err3 := resp.Body.Close(); err3 != nil {
					fmt.Printf("Error closing response body: %v\n", err3)
				}
			}()
		}
		return nil
	}
}
