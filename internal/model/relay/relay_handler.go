package relay

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"log/slog"
	"net/http"
	"time"
)

// HandleReceiveOnion handles incoming onion requests sent to the relay.
func (n *Relay) HandleReceiveOnion(w http.ResponseWriter, r *http.Request) {
	api_functions.HandleReceiveOnion(w, r, n.Receive)
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
		n.rCounterMu.Lock()
		if n.runCounter > 0 {
			n.rCounterMu.Unlock()
			n.wg.Wait()
			n.wg.Add(1)
		} else {
			n.rCounterMu.Unlock()
		}

		if didParticipate, err := n.startRun(start); err != nil {
			slog.Error("Error starting run", err)
		} else {
			slog.Info("Run complete", "did_participate", didParticipate)
		}
	}()
	w.WriteHeader(http.StatusOK)
}

func (n *Relay) HandleRegisterWithBulletinBoard(w http.ResponseWriter, r *http.Request) {
	slog.Info("Registering with bulletin board")

	go func(n *Relay) {
		for {
			if err := n.RegisterWithBulletinBoard(); err != nil {
				slog.Error("failed to register with bulletin board: " + err.Error())
			} else {
				slog.Info("Registered with bulletin board")
				break
			}
			time.Sleep(5 * time.Second)
		}
	}(n)
	w.WriteHeader(http.StatusOK)
}
