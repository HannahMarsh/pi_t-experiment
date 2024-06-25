package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"golang.org/x/exp/slog"
)

// Node represents a node in the onion routing network
type Node struct {
	ID               int
	Host             string
	Port             int
	PrivateKey       string
	PublicKey        string
	isMixer          bool
	mu               sync.RWMutex
	BulletinBoardUrl string
	lastUpdate       time.Time
	status           *api.NodeStatus
}

// NewNode creates a new node
func NewNode(id int, host string, port int, bulletinBoardUrl string, isMixer bool) (*Node, error) {
	if privateKey, publicKey, err := pi_t.KeyGen(); err != nil {
		return nil, pl.WrapError(err, "node.NewClient(): failed to generate key pair")
	} else {
		n := &Node{
			ID:               id,
			Host:             host,
			Port:             port,
			PublicKey:        publicKey,
			PrivateKey:       privateKey,
			BulletinBoardUrl: bulletinBoardUrl,
			isMixer:          isMixer,
			status: &api.NodeStatus{
				Received: make([]api.OnionStatus, 0),
				Node: api.PublicNodeApi{
					ID:        id,
					Address:   fmt.Sprintf("http://%s:%d", host, port),
					PublicKey: publicKey,
					Time:      time.Now(),
					IsMixer:   isMixer,
				},
			},
		}
		if err2 := n.RegisterWithBulletinBoard(); err2 != nil {
			return nil, pl.WrapError(err2, "node.NewNode(): failed to register with bulletin board")
		}

		go n.StartPeriodicUpdates(time.Second * 3)

		return n, nil
	}
}

func (n *Node) GetStatus() string {
	return n.status.GetStatus()
}

func (n *Node) getPublicNodeInfo() api.PublicNodeApi {
	return api.PublicNodeApi{
		ID:        n.ID,
		Address:   fmt.Sprintf("http://%s:%d", n.Host, n.Port),
		PublicKey: n.PublicKey,
		Time:      time.Now(),
		IsMixer:   n.isMixer,
	}
}

func (n *Node) StartPeriodicUpdates(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			//slog.Info("Updating bulletin board")
			if err := n.updateBulletinBoard("/updateNode", http.StatusOK); err != nil {
				fmt.Printf("Error updating bulletin board: %v\n", err)
				return
			}
		}
	}()
}

func (n *Node) startRun(start api.StartRunApi) (didParticipate bool, e error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	//n.wg.Wait()
	//n.wg.Add(1)
	//defer n.wg.Done()
	return true, nil
}

func (n *Node) Receive(o string) error {
	if peeled, err := pi_t.PeelOnion(o, n.PrivateKey); err != nil {
		return pl.WrapError(err, "node.Receive(): failed to remove layer")
	} else {
		n.status.AddOnion(peeled.LastHop, fmt.Sprintf("http://%s:%d", n.Host, n.Port), peeled.NextHop, peeled.Layer, peeled.IsCheckpointOnion)

		if peeled.NextHop == "" {
			var msg api.Message
			if err2 := json.Unmarshal([]byte(peeled.Payload), &msg); err2 != nil {
				return pl.WrapError(err2, "node.Receive(): failed to unmarshal message")
			}
			slog.Info("Received message", "from", msg.From, "to", msg.To, "msg", msg.Msg)

		} else {
			if peeled.IsCheckpointOnion {
				slog.Info("Received checkpoint onion", "layer", peeled.Layer, "destination", peeled.NextHop)
			} else {
				slog.Info("Received onion", "layer", peeled.Layer, "destination", peeled.NextHop)
			}
			//bruised, err2 := pi_t.BruiseOnion(payload)
			//if err2 != nil {
			//	return pl.WrapError(err2, "node.Receive(): failed to bruise onion")
			//}
			if err3 := sendToNode(peeled.NextHop, peeled.Payload); err != nil {
				return pl.WrapError(err3, "node.Receive(): failed to send to next node")
			}
		}
	}
	return nil
}

func sendToNode(addr string, constructedOnion string) error {
	url := fmt.Sprintf("%s/receive", addr)
	o := api.OnionApi{
		Onion: constructedOnion,
	}
	if data, err := json.Marshal(o); err != nil {
		slog.Error("failed to marshal msgs", err)
	} else if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
		return pl.WrapError(err2, "failed to send POST request with onion to next node")
	} else {
		defer func(Body io.ReadCloser) {
			if err3 := Body.Close(); err3 != nil {
				slog.Error("sendToNode(): Error closing response body", err3)
			}
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return pl.NewError("sendToNode(): Failed to send to next node, status code: %d, status: %s", resp.StatusCode, resp.Status)
		}
	}
	return nil
}

func (n *Node) RegisterWithBulletinBoard() error {
	slog.Info("Sending node registration request.", "id", n.ID)
	return n.updateBulletinBoard("/registerNode", http.StatusCreated)
}

func (n *Node) GetActiveNodes() ([]api.PublicNodeApi, error) {
	url := fmt.Sprintf("%s/nodes", n.BulletinBoardUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, pl.WrapError(err, fmt.Sprintf("error making GET request to %s", url))
	}
	defer func(Body io.ReadCloser) {
		if err2 := Body.Close(); err2 != nil {
			fmt.Printf("error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, pl.NewError("unexpected status code: %d", resp.StatusCode)
	}

	var activeNodes []api.PublicNodeApi
	if err = json.NewDecoder(resp.Body).Decode(&activeNodes); err != nil {
		return nil, pl.WrapError(err, "error decoding response body")
	}

	return activeNodes, nil
}

func (n *Node) updateBulletinBoard(endpoint string, expectedStatusCode int) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	t := time.Now()
	if data, err := json.Marshal(api.PublicNodeApi{
		ID:        n.ID,
		Address:   fmt.Sprintf("http://%s:%d", n.Host, n.Port),
		PublicKey: n.PublicKey,
		IsMixer:   n.isMixer,
		Time:      t,
	}); err != nil {
		return pl.WrapError(err, "node.UpdateBulletinBoard(): failed to marshal node info")
	} else {
		url := n.BulletinBoardUrl + endpoint
		//slog.Info("Sending node registration request.", "url", url, "id", n.ID)
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return pl.WrapError(err2, "node.UpdateBulletinBoard(): failed to send POST request to bulletin board")
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					fmt.Printf("node.UpdateBulletinBoard(): error closing response body: %v\n", err2)
				}
			}(resp.Body)
			if resp.StatusCode != expectedStatusCode {
				return pl.NewError("failed to %s node, status code: %d, %s", endpoint, resp.StatusCode, resp.Status)
			} else {
				n.lastUpdate = t
			}
			return nil
		}
	}
}
