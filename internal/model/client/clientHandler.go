package client

import (
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// HandleReceive handles incoming onions sent to the client by other nodes.
func (c *Client) HandleReceive(w http.ResponseWriter, r *http.Request) {
	api_functions.HandleReceiveOnion(w, r, c.Receive)
}

// HandleStartRun handles the initiation of a run based on a start signal received via an HTTP request.
func (c *Client) HandleStartRun(w http.ResponseWriter, r *http.Request) {
	slog.Info("Starting run")
	var start structs.ClientStartRunApi
	if err := json.NewDecoder(r.Body).Decode(&start); err != nil {
		slog.Error("Error decoding active nodes", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//slog.Info("Active nodes", "activeNodes", activeNodes)
	go func() {
		if err := c.startRun(start); err != nil {
			slog.Error("Error starting run", err)
		} else {
			slog.Info("Done sending onions")
		}
	}()
	w.WriteHeader(http.StatusOK)
}

func (c *Client) HandleRegisterWithBulletinBoard(w http.ResponseWriter, r *http.Request) {
	slog.Info("Registering with bulletin board")

	go func(c *Client) {
		for {
			if err := c.RegisterWithBulletinBoard(); err != nil {
				slog.Error("failed to register with bulletin board: " + err.Error())
			} else {
				slog.Info("Registered with bulletin board")
				break
			}
			time.Sleep(5 * time.Second)
		}
	}(c)
	w.WriteHeader(http.StatusOK)
}

// GetActiveNodes retrieves the list of active nodes from the bulletin board.
func (c *Client) GetActiveNodes() ([]structs.PublicNodeApi, error) {
	url := fmt.Sprintf("%s/Clients", c.BulletinBoardUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, PrettyLogger.WrapError(err, fmt.Sprintf("error making GET request to %s", url))
	}
	defer func(Body io.ReadCloser) {
		if err2 := Body.Close(); err2 != nil {
			fmt.Printf("error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, PrettyLogger.NewError("unexpected status code: %d", resp.StatusCode)
	}

	var activeClients []structs.PublicNodeApi
	if err = json.NewDecoder(resp.Body).Decode(&activeClients); err != nil {
		return nil, PrettyLogger.WrapError(err, "error decoding response body")
	}

	return activeClients, nil
}
