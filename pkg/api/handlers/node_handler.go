package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
	"github.com/HannahMarsh/pi_t-experiment/internal/usecases"
)

type NodeHandler struct {
	service *usecases.NodeService
}

func (h *NodeHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.RegisterNode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *NodeHandler) StartActions() {
	go h.service.StartActions()
}
