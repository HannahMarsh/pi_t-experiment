package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"time"
)

func (n *Node) HandleReceive(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received onion")
	var o Onion
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		slog.Error("Error decoding onion", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := n.Receive(&o); err != nil {
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
	slog.Info("Received client request")
	var msgs []api.Message
	if err := json.NewDecoder(r.Body).Decode(&msgs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Enqueuing messages", "num_messages", len(msgs))
	for _, msg := range msgs {
		if err := n.QueuedRequests.Enqueue(msg); err != nil {
			slog.Error("Error enqueuing message", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (n *Node) RegisterWithBulletinBoard() error {
	if data, err := json.Marshal(n.NodeInfo); err != nil {
		return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to marshal node info: %w", err)
	} else {
		url := n.BulletinBoardUrl + "/register"
		slog.Info("Sending node registration request.", "url", url, "id", n.NodeInfo.ID)
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to send POST request to bulletin board: %w", err2)
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					fmt.Printf("node.RegisterWithBulletinBoard(): error closing response body: %v\n", err2)
				}
			}(resp.Body)
			if resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to register node, status code: %d, %s", resp.StatusCode, resp.Status)
			}
			return nil
		}
	}
}

func (n *Node) updateBulletinBoard() error {
	// getsnapshot of requested messages
	pr := api.PrivateNodeApi{
		TimeOfRequest: time.Now(),
		ID: n.ID,
		Address: n.NodeInfo.Address,
		PublicKey: n.PublicKey,
		MessageQueue:
	}
	if data, err := json.Marshal(n.NodeInfo); err != nil {
		return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to marshal node info: %w", err)
	} else {
		url := n.BulletinBoardUrl + "/update"
		slog.Info("Sending node registration request.", "url", url, "id", n.NodeInfo.ID)
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to send POST request to bulletin board: %w", err2)
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					fmt.Printf("node.RegisterWithBulletinBoard(): error closing response body: %v\n", err2)
				}
			}(resp.Body)
			if resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to register node, status code: %d, %s", resp.StatusCode, resp.Status)
			}
			return nil
		}
	}
}
