package handlers

import (
	"encoding/json"
	"net/http"
	"pi_t-experiment/internal/domain/models"
	"pi_t-experiment/internal/usecases"
)

type BulletinBoardHandler struct {
	service *usecases.BulletinBoardService
}

func (h *BulletinBoardHandler) RegisterNode(w http.ResponseWriter, r *http.Request) {
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

func (h *BulletinBoardHandler) StartRuns() {
	go h.service.StartRuns()
}
