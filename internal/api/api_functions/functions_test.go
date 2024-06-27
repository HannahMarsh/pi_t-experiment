package api_functions

import (
	"encoding/json"
	"errors"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/keys"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

//
//func TestSendOnion(t *testing.T) {
//
//	pl.SetUpLogrusAndSlog("debug")
//
//	if err := config.InitGlobal(); err != nil {
//		slog.Error("failed to init config", err)
//		os.Exit(1)
//	}
//
//	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
//	if err != nil {
//		t.Fatalf("KeyGen() error: %v", err)
//	}
//
//	payload := []byte("secret message")
//	publicKeys := []string{publicKeyPEM, publicKeyPEM}
//	routingPath := []string{"node1", "node2"}
//
//	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1)
//
//	if err != nil {
//		slog.Error("FormOnion() error", err)
//		t.Fatalf("FormOnion() error = %v", err)
//	}
//
//	// Mock server to receive the onion
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		body, err := ioutil.ReadAll(r.Body)
//		if err != nil {
//			slog.Error("Failed to read request body", err)
//			t.Fatalf("Failed to read request body: %v", err)
//		}
//
//		var onion structs.OnionApi
//		if err := json.Unmarshal(body, &onion); err != nil {
//			slog.Error("Failed to unmarshal request body", err)
//			t.Fatalf("Failed to unmarshal request body: %v", err)
//		}
//
//		if onion.From != "node1" {
//			pl.LogNewError("Expected onion.From to be 'node1', got %s", onion.From)
//			t.Fatalf("Expected onion.From to be 'test_from', got %s", onion.From)
//		}
//
//		decompressedData, err := utils.Decompress(onion.Onion)
//		if err != nil {
//			slog.Error("Error decompressing data", err)
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		str := base64.StdEncoding.EncodeToString(decompressedData)
//
//		peelOnion, _, _, _, err2 := pi_t.PeelOnion(str, privateKeyPEM)
//		if err2 != nil {
//			slog.Error("PeelOnion() error", err2)
//			t.Fatalf("PeelOnion() error = %v", err2)
//		}
//
//		headerAdded, err := pi_t.AddHeader(peelOnion, 1, privateKeyPEM, publicKeyPEM)
//
//		peelOnion, _, _, _, err = pi_t.PeelOnion(headerAdded, privateKeyPEM)
//		if err != nil {
//			slog.Error("PeelOnion() error", err)
//			t.Fatalf("PeelOnion() error = %v", err)
//		}
//
//		if peelOnion.Payload != "secret message" {
//			t.Fatalf("Expected onion.Onion to be 'test onion data', got %s", peelOnion.Payload)
//		}
//
//		w.WriteHeader(http.StatusOK)
//	}))
//	defer server.Close()
//
//	err = SendOnion(server.URL, addr, onion)
//	if err != nil {
//		slog.Error("SendOnion() error", err)
//		t.Fatalf("SendOnion() error = %v", err)
//	}
//}

func TestReceiveOnion(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	receiverPort := 8500

	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM, publicKeyPEM}
	routingPath := []string{fmt.Sprintf("http://localhost:%d", receiverPort), "node2"}

	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1)

	if err != nil {
		slog.Error("FormOnion() error", err)
		t.Fatalf("FormOnion() error = %v", err)
	}

	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
		HandleReceiveOnion(w, r, func(onionStr string) error {
			peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM)
			if err2 != nil {
				slog.Error("PeelOnion() error", err2)
				t.Fatalf("PeelOnion() error = %v", err2)
			}

			headerAdded, err := pi_t.AddHeader(peelOnion, 1, privateKeyPEM, publicKeyPEM)

			peelOnion, _, _, _, err = pi_t.PeelOnion(headerAdded, privateKeyPEM)
			if err != nil {
				slog.Error("PeelOnion() error", err)
				t.Fatalf("PeelOnion() error = %v", err)
			}

			if peelOnion.Payload != "secret message" {
				t.Fatalf("Expected onion.Onion to be 'test onion data', got %s", peelOnion.Payload)
			}

			return nil
		})
	})

	go func() {
		address := fmt.Sprintf(":%d", receiverPort)
		if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	err = SendOnion(addr, "sender", onion)
	if err != nil {
		slog.Error("SendOnion() error", err)
		t.Fatalf("SendOnion() error = %v", err)
	}
}

func TestReceiveOnionMultipleLayers(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	receiverPort1 := 8500
	receiverPort2 := 8501

	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
	routingPath := []string{fmt.Sprintf("http://localhost:%d", receiverPort1), fmt.Sprintf("http://localhost:%d", receiverPort2)}

	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1)

	if err != nil {
		slog.Error("FormOnion() error", err)
		t.Fatalf("FormOnion() error = %v", err)
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
			HandleReceiveOnion(w, r, func(onionStr string) error {
				peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM1)
				if err2 != nil {
					slog.Error("PeelOnion() error", err2)
					t.Fatalf("PeelOnion() error = %v", err2)
				}

				headerAdded, err3 := pi_t.AddHeader(peelOnion, 1, privateKeyPEM1, publicKeyPEM1)
				if err3 != nil {
					slog.Error("AddHeader() error", err3)
					t.Fatalf("AddHeader() error = %v", err3)
				}

				err4 := SendOnion(peelOnion.NextHop, fmt.Sprintf("http://localhost:%d", receiverPort1), headerAdded)
				if err4 != nil {
					slog.Error("SendOnion() error", err4)
					t.Fatalf("SendOnion() error = %v", err4)
				}

				return nil
			})
		})
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", receiverPort1),
			Handler: mux,
		}
		if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
			HandleReceiveOnion(w, r, func(onionStr string) error {
				peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM2)
				if err2 != nil {
					slog.Error("PeelOnion() error", err2)
					t.Fatalf("PeelOnion() error = %v", err2)
				}

				if peelOnion.Payload != "secret message" {
					t.Fatalf("Expected onion.Onion to be 'test onion data', got %s", peelOnion.Payload)
				}

				slog.Info("Successfully received message", "message", peelOnion.Payload)

				return nil
			})
		})
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", receiverPort2),
			Handler: mux,
		}
		if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	err = SendOnion(addr, "sender", onion)
	if err != nil {
		slog.Error("SendOnion() error", err)
		t.Fatalf("SendOnion() error = %v", err)
	}
}

func TestReceiveCheckpointOnions(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	receiverPort1 := 8500
	receiverPort2 := 8501

	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	msg := structs.Message{
		Msg: "secret message",
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Marshal() error", err)
		t.Fatalf("Marshal() error: %v", err)
	}
	publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
	routingPath := []string{fmt.Sprintf("http://localhost:%d", receiverPort1), fmt.Sprintf("http://localhost:%d", receiverPort2)}

	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, 0)

	if err != nil {
		slog.Error("FormOnion() error", err)
		t.Fatalf("FormOnion() error = %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
			HandleReceiveOnion(w, r, func(onionStr string) error {
				defer wg.Done()
				peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM1)
				if err2 != nil {
					slog.Error("PeelOnion() error", err2)
					t.Fatalf("PeelOnion() error = %v", err2)
				}

				headerAdded, err3 := pi_t.AddHeader(peelOnion, 1, privateKeyPEM1, publicKeyPEM1)
				if err3 != nil {
					slog.Error("AddHeader() error", err3)
					t.Fatalf("AddHeader() error = %v", err3)
				}

				err4 := SendOnion(peelOnion.NextHop, fmt.Sprintf("http://localhost:%d", receiverPort1), headerAdded)
				if err4 != nil {
					slog.Error("SendOnion() error", err4)
					t.Fatalf("SendOnion() error = %v", err4)
				}

				return nil
			})
		})
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", receiverPort1),
			Handler: mux,
		}
		if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
			HandleReceiveOnion(w, r, func(onionStr string) error {
				defer wg.Done()
				peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM2)
				if err2 != nil {
					slog.Error("PeelOnion() error", err2)
					t.Fatalf("PeelOnion() error = %v", err2)
				}

				var msg structs.Message

				err := json.Unmarshal([]byte(peelOnion.Payload), &msg)
				if err != nil {
					slog.Error("Unmarshal() error", err)
					t.Fatalf("Unmarshal() error: %v", err)
				}

				if msg.Msg != "secret message" {
					t.Fatalf("Expected onion.Onion to be 'test onion data', got %s", msg.Msg)
				}

				slog.Info("Successfully received message", "message", msg.Msg)

				return nil
			})
		})
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", receiverPort2),
			Handler: mux,
		}
		if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	time.Sleep(1 * time.Second)

	err = SendOnion(addr, "sender", onion)
	if err != nil {
		slog.Error("SendOnion() error", err)
		t.Fatalf("SendOnion() error = %v", err)
	}

	wg.Wait()
}
