package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

//func (n *Node) HandleAddOnions(w http.ResponseWriter, r *http.Request) {
//	var onions []Onion
//	if err := json.NewDecoder(r.Body).Decode(&onions); err != nil {
//		http.Error(w, err.Error(), http.StatusBadRequest)
//		return
//	}
//	n.wg.Wait()
//	if part, err := n.startRun(&onions); err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	} else if part {
//
//	}
//	w.WriteHeader(http.StatusOK)
//}

func (n *Node) RegisterWithBulletinBoard() error {
	data, err := json.Marshal(n.NodeInfo)
	if err != nil {
		return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to marshal node info: %w", err)
	}
	resp, err := http.Post(n.BulletinBoardUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err2 := Body.Close()
		if err2 != nil {
			fmt.Printf("node.RegisterWithBulletinBoard(): error closing response body: %v\n", err2)
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("node.RegisterWithBulletinBoard(): failed to register node, status code: %d", resp.StatusCode)
	}
	return nil
}
