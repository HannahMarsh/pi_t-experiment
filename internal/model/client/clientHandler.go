package client

import (
	"encoding/json"
	"fmt"
	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
)

func (c *Client) HandleReceive(w http.ResponseWriter, r *http.Request) {
	api_functions.HandleReceiveOnion(w, r, c.Receive)
	//var o structs.OnionApi
	//if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
	//	slog.Error("Error decoding onion", err)
	//	http.Error(w, err.Error(), http.StatusBadRequest)
	//	return
	//}
	//decompressed, err := api.Receive(o.Onion)
	//if err != nil {
	//	slog.Error("Error decompressing onion", err)
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//if err = c.Receive(decompressed); err != nil {
	//	slog.Error("Error receiving onion", err)
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//w.WriteHeader(http.StatusOK)
}

func (c *Client) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(c.GetStatus())); err != nil {
		slog.Error("Error writing response", err)
	}
}

func (c *Client) HandleStartRun(w http.ResponseWriter, r *http.Request) {
	slog.Info("Starting run")
	var start structs.StartRunApi
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
