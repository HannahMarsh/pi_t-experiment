package relay

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"log/slog"
	"net/http"
)

// HandleReceiveOnion handles incoming onion requests sent to the relay.
func (n *Relay) HandleReceiveOnion(w http.ResponseWriter, r *http.Request) {
	api_functions.HandleReceiveOnion(w, r, n.Receive)
}

// HandleGetStatus returns the current status of the relay in response to an HTTP request.
func (n *Relay) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(n.GetStatus())); err != nil {
		slog.Error("Error writing response", err)
	}
}

// HandleStartRun handles the initiation of a relay run based on a start signal received via an HTTP request.
func (n *Relay) HandleStartRun(w http.ResponseWriter, r *http.Request) {
	slog.Info("Starting run")
	var start structs.RelayStartRunApi
	// Decode the JSON request body into the start signal struct.
	if err := json.NewDecoder(r.Body).Decode(&start); err != nil {
		slog.Error("Error decoding active relays", err)   // Log any errors that occur during decoding.
		http.Error(w, err.Error(), http.StatusBadRequest) // Respond with a Bad Request status if decoding fails.
		return
	}

	go func() {
		if didParticipate, err := n.startRun(start); err != nil {
			slog.Error("Error starting run", err)
		} else {
			slog.Info("Run complete", "did_participate", didParticipate)
		}
	}()
	w.WriteHeader(http.StatusOK)
}
