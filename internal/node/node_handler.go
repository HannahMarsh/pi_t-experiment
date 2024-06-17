package node

import (
	"encoding/json"
	"net/http"
)

func (n *Node) HandleReceive(w http.ResponseWriter, r *http.Request) {
	var o Onion
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := n.Receive(&o); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (n *Node) RegisterWithBulletinBoard() *Onion {

}
