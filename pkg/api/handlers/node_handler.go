package handlers

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/internal/node"
	"net/http"

	"github.com/HannahMarsh/pi_t-experiment/internal/usecases"
)

type NodeHandler struct {
	Service *usecases.NodeService
}

func (h *NodeHandler) Receive(w http.ResponseWriter, r *http.Request) {
	var o node.Onion
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.Receive(&o); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
