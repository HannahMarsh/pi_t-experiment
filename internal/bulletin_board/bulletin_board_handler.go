package bulletin_board

import (
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"net/http"
)

func (bb *BulletinBoard) HandleRegisterNode(w http.ResponseWriter, r *http.Request) {
	var node api.PrivateNodeApi
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bb.UpdateNode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) HandleUpdateNodeInfo(w http.ResponseWriter, r *http.Request) {
	var nodeInfo api.PrivateNodeApi
	if err := json.NewDecoder(r.Body).Decode(&nodeInfo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bb.UpdateNode(&nodeInfo); err != nil {
		fmt.Printf("Error updating node %d: %v\n", nodeInfo.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (bb *BulletinBoard) signalNodesToStart() error {
	for _, node := range utils.NewMapStream(bb.Network).Filter(func(_ int, node *NodeView) bool {
		return node.IsActive()
	}).GetValues().Array {
		url := fmt.Sprintf("http://%s/start", node.Address)
		if resp, err := http.Post(url, "application/json", nil); err != nil {
			fmt.Printf("Error signaling node %d to start: %v\n", node.ID, err)
			continue
		} else if err = resp.Body.Close(); err != nil {
			return fmt.Errorf("error closing response body: %w", err)
		}
	}
	return nil
}
