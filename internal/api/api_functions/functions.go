package api_functions

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// sendOnion sends an onion to the specified address with compression and timeout
func SendOnion(to, from string, o onion_model.Onion) error {
	slog.Debug("Sending onion...", "from", config.AddressToName(from), "to", config.AddressToName(to))
	url := fmt.Sprintf("%s/receive", to)

	//data, err := base64.StdEncoding.DecodeString(onionStr)
	//if err != nil {
	//	return pl.WrapError(err, "%s: failed to decode onion string", pl.GetFuncName())
	//}
	//
	////beforeSize := len(data)
	//
	//compressedData, err := utils.Compress(data)
	//if err != nil {
	//	return pl.WrapError(err, "%s: failed to compress onion", pl.GetFuncName())
	//}
	//
	////afterSize := len(compressedData)
	////slog.Info(pl.GetFuncName(), "before", beforeSize, "after", afterSize, "Saved", fmt.Sprintf("%.2f%%", 100-float64(afterSize)/float64(beforeSize)*100))
	//
	//encodeToString := base64.StdEncoding.EncodeToString(compressedData)

	data, err := json.Marshal(o)
	if err != nil {
		return pl.WrapError(err, "%s: failed to marshal onion", pl.GetFuncName())
	}

	oStr := base64.StdEncoding.EncodeToString(data)

	onion := structs.OnionApi{
		To:    to,
		From:  from,
		Onion: oStr,
	}

	payload, err := json.Marshal(onion)
	if err != nil {
		return pl.WrapError(err, "%s: failed to marshal onion", pl.GetFuncName())
	}

	compressedBuffer, err := utils.Compress(payload)
	if err != nil {
		return pl.WrapError(err, "%s: failed to compress onion", pl.GetFuncName())
	}

	client := &http.Client{
		Timeout: 30 * time.Second, // Set timeout
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, &compressedBuffer)
	if err != nil {
		return pl.WrapError(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		return pl.WrapError(err, "%s: failed to send POST request with onion to %s", pl.GetFuncName(), config.AddressToName(to))
	}

	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			slog.Error("Error closing response body", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return pl.NewError("%s: failed to send to first node(url=%s), status code: %d, status: %s", pl.GetFuncName(), url, resp.StatusCode, resp.Status)
	}

	slog.Debug("✅ Successfully sent onion. ", "from", config.AddressToName(from), "to", config.AddressToName(to))
	return nil
}

func HandleReceiveOnion(w http.ResponseWriter, r *http.Request, receiveFunction func(api structs.OnionApi) error) {

	var body []byte
	var err error

	// Check if the request is gzipped
	if r.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			slog.Error("Error creating gzip reader", err)
			http.Error(w, "Failed to read gzip content", http.StatusBadRequest)
			return
		}
		defer func(gzipReader *gzip.Reader) {
			if err := gzipReader.Close(); err != nil {
				slog.Error("Error closing gzip reader", err)
			}
		}(gzipReader)

		body, err = io.ReadAll(gzipReader)
		if err != nil {
			slog.Error("Error reading gzip content", err)
			http.Error(w, "Failed to read gzip content", http.StatusBadRequest)
			return
		}
	} else {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unable to read body", http.StatusInternalServerError)
			return
		}
	}

	var o structs.OnionApi
	if err := json.Unmarshal(body, &o); err != nil {
		slog.Error("Error decoding onion", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = receiveFunction(o); err != nil {
		slog.Error("Error receiving onion", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
