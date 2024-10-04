package bulletin_board

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"log/slog"
	"net/http"
)

// HandleRegisterRelay processes HTTP requests for registering a relay node.
func (bb *BulletinBoard) HandleRegisterRelay(w http.ResponseWriter, r *http.Request) {
	var relay structs.PublicNodeApi

	// Decode the JSON request body into the relay struct.
	if err := json.NewDecoder(r.Body).Decode(&relay); err != nil {
		// If decoding fails, log the error and respond with a Bad Request status.
		slog.Error("Error decoding relay registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log the relay registration event with the relay ID.
	slog.Info("Registering relay with", "id", relay.ID)

	// Update the bulletin board with the new relay information.
	bb.UpdateRelay(relay)

	w.WriteHeader(http.StatusCreated)
}

// HandleRegisterClient processes HTTP requests for registering a client node.
func (bb *BulletinBoard) HandleRegisterClient(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received client registration request") // Log that a client registration request has been received.

	var client structs.PublicNodeApi

	// Decode the JSON request body into the client struct.
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		slog.Error("Error decoding client registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log the client registration event with the client ID.
	slog.Info("Registering client with", "id", client.ID)

	// Register the client with the bulletin board.
	go bb.RegisterClient(client)

	w.WriteHeader(http.StatusCreated)
}

// HandleRegisterIntentToSend processes HTTP requests for registering a client's intent to send a message.
//func (bb *BulletinBoard) HandleRegisterIntentToSend(w http.ResponseWriter, r *http.Request) {
//	var its structs.IntentToSend
//
//	// Decode the JSON request body into the intent-to-send struct.
//	if err := json.NewDecoder(r.Body).Decode(&its); err != nil {
//		slog.Error("Error decoding intent-to-send registration request", err)
//		http.Error(w, err.Error(), http.StatusBadRequest)
//		return
//	}
//
//	// Register the intent to send with the bulletin board.
//	if err := bb.RegisterIntentToSend(its); err != nil {
//		slog.Error("Error registering intent-to-send request", err)
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	w.WriteHeader(http.StatusOK)
//}

// HandleUpdateRelayInfo processes HTTP requests for updating relay node information.
func (bb *BulletinBoard) HandleUpdateRelayInfo(w http.ResponseWriter, r *http.Request) {
	var nodeInfo structs.PublicNodeApi

	// Decode the JSON request body into the nodeInfo struct.
	if err := json.NewDecoder(r.Body).Decode(&nodeInfo); err != nil {
		slog.Error("Error decoding relay info update request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update the node information in the bulletin board.
	bb.UpdateRelay(nodeInfo)

	w.WriteHeader(http.StatusOK)
}

// HandleUpdateConfig processes HTTP requests for updating the config
func (bb *BulletinBoard) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config

	// Decode the JSON request body into the nodeInfo struct.
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		slog.Error("Error decoding config update request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go config.UpdateConfig(cfg)

	w.WriteHeader(http.StatusOK)
}
