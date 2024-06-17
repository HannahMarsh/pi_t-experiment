package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
	"github.com/HannahMarsh/pi_t-experiment/internal/usecases"
)

type NodeHandler struct {
	Service *usecases.NodeService
}

func (h *NodeHandler) Receive(w http.ResponseWriter, r *http.Request) {
	var msg models.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.Receive(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *NodeHandler) StartActions() {
	go h.Service.StartActions()
}
