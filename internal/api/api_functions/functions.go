package api_functions

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"time"
)

// sendOnion sends an onion to the specified address with compression and timeout
func SendOnion(to, from, onionStr string) error {
	slog.Info(pl.GetFuncName()+": Sending onion...", "from", config.AddressToName(from), "to", config.AddressToName(to))
	url := fmt.Sprintf("%s/receive", to)

	data, err := base64.StdEncoding.DecodeString(onionStr)
	if err != nil {
		return pl.WrapError(err, "%s: failed to decode onion string", pl.GetFuncName())
	}

	//beforeSize := len(data)

	compressedData, err := utils.Compress(data)
	if err != nil {
		return pl.WrapError(err, "%s: failed to compress onion", pl.GetFuncName())
	}

	//afterSize := len(compressedData)
	//slog.Info(pl.GetFuncName(), "before", beforeSize, "after", afterSize, "Saved", fmt.Sprintf("%.2f%%", 100-float64(afterSize)/float64(beforeSize)*100))

	encodeToString := base64.StdEncoding.EncodeToString(compressedData)
	onion := structs.OnionApi{
		To:    to,
		From:  from,
		Onion: encodeToString,
	}

	payload, err := json.Marshal(onion)
	if err != nil {
		return pl.WrapError(err, "%s: failed to marshal onion", pl.GetFuncName())
	}

	client := &http.Client{
		Timeout: 10 * time.Second, // Set timeout
	}

	//slog.Info(pl.GetFuncName() + ": payload size: " + fmt.Sprintf("%d", len(payload)))

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return pl.WrapError(err, "%s: failed to send POST request with onion to first mixer", pl.GetFuncName())
	}

	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			slog.Error("Error closing response body", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return pl.NewError("%s: failed to send to first node(url=%s), status code: %d, status: %s", pl.GetFuncName(), url, resp.StatusCode, resp.Status)
	}

	slog.Info("✅ Successfully sent onion. ", "from", config.AddressToName(from), "to", config.AddressToName(to))
	return nil
}

func HandleReceiveOnion(w http.ResponseWriter, r *http.Request, receiveFunction func(string) error) {
	var o structs.OnionApi
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to read body", http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal(body, &o); err != nil {
		slog.Error("Error decoding onion", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := base64.StdEncoding.DecodeString(o.Onion)
	if err != nil {
		slog.Error("Error decoding onion", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	decompressedData, err := utils.Decompress(data)
	if err != nil {
		slog.Error("Error decompressing data", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	onionString := base64.StdEncoding.EncodeToString(decompressedData)
	if err = receiveFunction(onionString); err != nil {
		slog.Error("Error receiving onion", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
