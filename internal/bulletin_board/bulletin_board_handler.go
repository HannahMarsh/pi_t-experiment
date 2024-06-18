package bulletin_board

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"net/http"
)

func (bb *BulletinBoard) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var node api.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bb.UpdateNode(node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) UpdateQueueInfo(w http.ResponseWriter, r *http.Request) {
	var nodeQueueInfo api.NodeQueueInfo
	if err := json.NewDecoder(r.Body).Decode(&nodeQueueInfo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	bb.NodeQueueMap.Store(nodeQueueInfo.ID, nodeQueueInfo.QueueSize)
	w.WriteHeader(http.StatusOK)
}
