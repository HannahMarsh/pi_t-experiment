package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"golang.org/x/exp/slog"
)

func (n *Node) HandleReceive(w http.ResponseWriter, r *http.Request) {
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
	var activeNodes []api.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&activeNodes); err != nil {
		slog.Error("Error decoding active nodes", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//slog.Info("Active nodes", "activeNodes", activeNodes)
	go func() {
		if didParticipate, err := n.startRun(activeNodes); err != nil {
			slog.Error("Error starting run", err)
		} else {
			slog.Info("Run complete", "did_participate", didParticipate)
		}
	}()
	w.WriteHeader(http.StatusOK)
}

func (n *Node) HandleClientRequest(w http.ResponseWriter, r *http.Request) {

	var msgs []api.Message
	if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//slog.Info("Received client request", "num_messages", len(msgs), "destinations", utils.Map(msgs, func(m api.Message) int { return m.To }))
	//slog.Info("Enqueuing messages", "num_messages", len(msgs))
	for _, msg := range msgs {
		if err := n.QueueOnion(msg, 2); err != nil {
			slog.Error("Error queueing message", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (n *Node) RegisterWithBulletinBoard() error {
	slog.Info("Sending node registration request.", "id", n.ID)
	return n.updateBulletinBoard("/register", http.StatusCreated)
}

func (n *Node) GetActiveNodes() ([]api.PublicNodeApi, error) {
	url := fmt.Sprintf("%s/nodes", n.BulletinBoardUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making GET request to %s: %v", url, err)
	}
	defer func(Body io.ReadCloser) {
		if err2 := Body.Close(); err2 != nil {
			fmt.Printf("error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var activeNodes []api.PublicNodeApi
	if err = json.NewDecoder(resp.Body).Decode(&activeNodes); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return activeNodes, nil
}

func (n *Node) updateBulletinBoard(endpoint string, expectedStatusCode int) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	t := time.Now()
	if data, err := json.Marshal(n.getPrivateNodeInfo(t)); err != nil {
		return PrettyLogger.WrapError(err, "node.UpdateBulletinBoard(): failed to marshal node info")
	} else {
		url := n.BulletinBoardUrl + endpoint
		//slog.Info("Sending node registration request.", "url", url, "id", n.ID)
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return PrettyLogger.WrapError(err2, "node.UpdateBulletinBoard(): failed to send POST request to bulletin board")
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					fmt.Printf("node.UpdateBulletinBoard(): error closing response body: %v\n", err2)
				}
			}(resp.Body)
			if resp.StatusCode != expectedStatusCode {
				return PrettyLogger.NewError("failed to %s node, status code: %d, %s", endpoint, resp.StatusCode, resp.Status)
			} else {
				n.lastUpdate = t
			}
			return nil
		}
	}
}
