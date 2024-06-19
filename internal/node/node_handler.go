package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
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
		n.mu.Lock()
		n.MessageQueue = append(n.MessageQueue, &msg)
		n.mu.Unlock()
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

func (n *Node) GetActiveNodes() ([]api.PublicNodeApi, error) {
	url := fmt.Sprintf("%s/nodes", n.BulletinBoardUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making GET request to %s: %v", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
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

func (n *Node) updateBulletinBoard() error {
	n.mu.Lock()
	a, _ := n.GetActiveNodes()
	if a != nil && len(a) > 0 {
		n.ActiveNodes = a
	}
	m := utils.NewStream(n.MessageQueue).MapToInt(func(msg *api.Message) int {
		return msg.To
	}).Array
	n.mu.Unlock()
	nodeInfo := api.PrivateNodeApi{
		ID:           n.ID,
		Address:      n.NodeInfo.Address,
		PublicKey:    n.PublicKey,
		MessageQueue: m,
	}
	if data, err := json.Marshal(nodeInfo); err != nil {
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
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to register node, status code: %d, %s", resp.StatusCode, resp.Status)
			}
			return nil
		}
	}
}
