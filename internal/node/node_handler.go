package node

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"golang.org/x/exp/slog"
	"net/http"
)

func (n *Node) HandleReceiveOnion(w http.ResponseWriter, r *http.Request) {
	var o api.OnionApi
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		slog.Error("Error decoding onion", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := n.Receive(o.Onion); err != nil {
		slog.Error("Error receiving onion", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (n *Node) HandleStartRun(w http.ResponseWriter, r *http.Request) {
	slog.Info("Starting run")
	var start api.StartRunApi
	if err := json.NewDecoder(r.Body).Decode(&start); err != nil {
		slog.Error("Error decoding active nodes", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//slog.Info("Active nodes", "start", start)
	go func() {
		if didParticipate, err := n.startRun(start); err != nil {
			slog.Error("Error starting run", err)
		} else {
			slog.Info("Run complete", "did_participate", didParticipate)
		}
	}()
	w.WriteHeader(http.StatusOK)
}

//
//func (n *Node) HandleClientRequest(w http.ResponseWriter, r *http.Request) {
//
//	var msgs []api.Message
//	if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
//		http.Error(w, err.Error(), http.StatusBadRequest)
//		return
//	}
//	//slog.Info("Received client request", "num_messages", len(msgs), "destinations", utils.Map(msgs, func(m api.Message) int { return m.To }))
//	//slog.Info("Enqueuing messages", "num_messages", len(msgs))
//	for _, msg := range msgs {
//		if err := n.QueueOnion(msg, 2); err != nil {
//			slog.Error("Error queueing message", err)
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//	}
//	w.WriteHeader(http.StatusOK)
//}
