package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/HannahMarsh/pi_t-experiment/internal/domain/models"
	"github.com/HannahMarsh/pi_t-experiment/internal/usecases"
)

type BulletinBoardHandler struct {
	Service *usecases.BulletinBoardService
}

func (h *BulletinBoardHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.RegisterNode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *BulletinBoardHandler) StartRuns() {
	go h.Service.StartRuns()
}
