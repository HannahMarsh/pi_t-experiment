package metrics

import (
	"net/http"
)

func HandleUpdateMessageQueue(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func HandleStartRun(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func HandleClientSentOnion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func HandleClientReceivedOnion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func HandleNodeSentOnion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func HandleNodeReceivedOnion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
