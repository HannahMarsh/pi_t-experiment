package bulletin_board

import (
	"encoding/json"
	"github.com/HannahMarsh/pi_t-experiment/internal/node"
	"net/http"
)

func (bb *BulletinBoard) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var node node.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := bb.AddNode(node); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
