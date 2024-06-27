package bulletin_board

import (
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"golang.org/x/exp/slog"
	"net/http"
)

func (bb *BulletinBoard) HandleRegisterNode(w http.ResponseWriter, r *http.Request) {
	//	slog.Info("Received node registration request")
	var node structs.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		slog.Error("Error decoding node registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Registering node with", "id", node.ID)
	if err := bb.UpdateNode(node); err != nil {
		slog.Error("Error updating node", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) HandleRegisterClient(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received client registration request")
	var client structs.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		slog.Error("Error decoding client registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Registering client with", "id", client.ID)
	if err := bb.RegisterClient(client); err != nil {
		slog.Error("Error registering client", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) HandleRegisterIntentToSend(w http.ResponseWriter, r *http.Request) {
	//	slog.Info("Received intent-to-send request")
	var its structs.IntentToSend
	if err := json.NewDecoder(r.Body).Decode(&its); err != nil {
		slog.Error("Error decoding intent-to-send registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bb.RegisterIntentToSend(its); err != nil {
		slog.Error("Error registering intent-to-send request", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (bb *BulletinBoard) HandleUpdateNodeInfo(w http.ResponseWriter, r *http.Request) {
	//slog.Info("Received node info update request")
	var nodeInfo structs.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&nodeInfo); err != nil {
		slog.Error("Error decoding node info update request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//	slog.Info("Updating node with", "id", nodeInfo.ID)
	if err := bb.UpdateNode(nodeInfo); err != nil {
		fmt.Printf("Error updating node %d: %v\n", nodeInfo.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
