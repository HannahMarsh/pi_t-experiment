package bulletin_board

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
)

func (bb *BulletinBoard) HandleRegisterNode(w http.ResponseWriter, r *http.Request) {
	//	slog.Info("Received node registration request")
	var node api.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		slog.Error("Error decoding node registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Registering node with", "id", node.ID)
	if err := bb.UpdateNode(node); err != nil {
		slog.Error("Error updating node", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) HandleRegisterClient(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received client registration request")
	var client api.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		slog.Error("Error decoding client registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("Registering client with", "id", client.ID)
	if err := bb.RegisterClient(client); err != nil {
		slog.Error("Error registering client", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (bb *BulletinBoard) HandleRegisterIntentToSend(w http.ResponseWriter, r *http.Request) {
	//	slog.Info("Received intent-to-send request")
	var its api.IntentToSend
	if err := json.NewDecoder(r.Body).Decode(&its); err != nil {
		slog.Error("Error decoding intent-to-send registration request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bb.RegisterIntentToSend(its); err != nil {
		slog.Error("Error registering intent-to-send request", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (bb *BulletinBoard) HandleUpdateNodeInfo(w http.ResponseWriter, r *http.Request) {
	//slog.Info("Received node info update request")
	var nodeInfo api.PublicNodeApi
	if err := json.NewDecoder(r.Body).Decode(&nodeInfo); err != nil {
		slog.Error("Error decoding node info update request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//	slog.Info("Updating node with", "id", nodeInfo.ID)
	if err := bb.UpdateNode(nodeInfo); err != nil {
		fmt.Printf("Error updating node %d: %v\n", nodeInfo.ID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

//
//// HandleGetActiveNodes handles GET requests to return all active nodes
//func (bb *BulletinBoard) HandleGetActiveNodes(w http.ResponseWriter, r *http.Request) {
//	bb.mu.Lock()
//	defer bb.mu.Unlock()
//	activeNodes := utils.NewMapStream(bb.Network).Filter(func(_ int, node *ClientView) bool {
//		return node.IsActive()
//	}).GetValues().Array
//
//	activeNodesApis := utils.Map(activeNodes, func(node *ClientView) api.PublicNodeApi {
//		return node.Api
//	})
//
//	w.Header().Set("Content-Type", "application/json")
//	if err := json.NewEncoder(w).Encode(activeNodesApis); err != nil {
//		slog.Error("Error encoding response", err)
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//}

func (bb *BulletinBoard) signalNodesToStart() error {
	slog.Info("Signaling nodes to start")
	activeNodes := utils.MapEntries(utils.FilterMap(bb.Network, func(_ int, node *NodeView) bool {
		return node.IsActive()
	}), func(_ int, nv *NodeView) api.PublicNodeApi {
		return api.PublicNodeApi{
			ID:        nv.ID,
			Address:   nv.Address,
			PublicKey: nv.PublicKey,
			Time:      nv.LastHeartbeat,
			IsMixer:   nv.IsMixer,
		}
	})

	activeClients := utils.MapEntries(utils.FilterMap(bb.Clients, func(_ int, cl *ClientView) bool {
		return cl.IsActive()
	}), func(_ int, cv *ClientView) api.PublicNodeApi {
		return api.PublicNodeApi{
			ID:        cv.ID,
			Address:   cv.Address,
			PublicKey: cv.PublicKey,
		}
	})

	numMessages := utils.Max(utils.MapEntries(bb.Clients, func(_ int, client *ClientView) int {
		return len(client.MessageQueue)
	})) + 2

	mixers := utils.Filter(activeNodes, func(n api.PublicNodeApi) bool {
		return n.Address != "" && n.IsMixer
	})

	gatekeepers := utils.Filter(activeNodes, func(n api.PublicNodeApi) bool {
		return n.Address != "" && !n.IsMixer
	})

	vs := api.StartRunApi{
		ParticipatingClients: activeClients,
		Mixers:               mixers,
		Gatekeepers:          gatekeepers,
		NumMessagesPerClient: numMessages,
	}

	if data, err := json.Marshal(vs); err != nil {
		return PrettyLogger.WrapError(err, "failed to marshal start signal")
	} else {
		var wg sync.WaitGroup
		all := utils.Copy(activeNodes)
		all = append(all, activeClients...)
		all = utils.Filter(all, func(n api.PublicNodeApi) bool {
			return n.Address != ""
		})
		for _, n := range all {
			n := n
			wg.Add(1)
			go func() {
				defer wg.Done()
				url := fmt.Sprintf("%s/start", n.Address)
				if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
					slog.Error("Error signaling node to start \n"+url, err2)
				} else if err3 := resp.Body.Close(); err3 != nil {
					fmt.Printf("Error closing response body: %v\n", err3)
				}
			}()
		}
		//for _, c := range activeClients {
		//	c := c
		//	wg.Add(1)
		//	go func() {
		//		defer wg.Done()
		//		url := fmt.Sprintf("%s/start", c.Address)
		//		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
		//			slog.Error("Error signaling client to start\n", err2)
		//		} else if err3 := resp.Body.Close(); err3 != nil {
		//			fmt.Printf("Error closing response body: %v\n", err3)
		//		}
		//	}()
		//}
		wg.Wait()
		return nil
	}
}
