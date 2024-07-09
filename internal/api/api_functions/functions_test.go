package api_functions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"net"
	"net/http"
	"os"
	"strings"
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

var usedPorts sync.Map

// Helper function to get an available port
func getAvailablePort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		slog.Error("failed to listen", err)
		return -1
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			slog.Error("failed to close listener", err)
		}
	}(listener)
	port := listener.Addr().(*net.TCPAddr).Port

	// Check if port is already in use
	if _, ok := usedPorts.LoadOrStore(port, true); ok {
		return getAvailablePort()
	}
	return port
}

func TestReceiveOnion(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	receiverPort := getAvailablePort()

	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM, publicKeyPEM}
	routingPath := []string{fmt.Sprintf("http://localhost:%d", receiverPort), "node2"}

	o, err := pi_t.FORMONION(publicKeyPEM, privateKeyPEM, base64.StdEncoding.EncodeToString(payload), mixersAddr, gatekeepersAddr, destination.Address, publicKeys, metadata, config.GlobalConfig.D)

	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1, "")

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

	receiverPort1 := getAvailablePort()
	receiverPort2 := getAvailablePort()

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

	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1, "")

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

	receiverPort1 := getAvailablePort()
	receiverPort2 := getAvailablePort()

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

	addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, 0, "")

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

func TestReceiveOnionMultipleLayers2(t *testing.T) {
	for nnn := 0; nnn < 100; nnn++ {
		pl.SetUpLogrusAndSlog("debug")

		if err := config.InitGlobal(); err != nil {
			slog.Error("failed to init config", err)
			os.Exit(1)
		}

		var err error

		numNodes := 7

		type node struct {
			privateKeyPEM string
			publicKeyPEM  string
			address       string
			port          int
		}

		nodes := make([]node, numNodes)

		for i := 0; i < numNodes; i++ {
			privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
			if err != nil {
				t.Fatalf("KeyGen() error: %v", err)
			}
			port := getAvailablePort()
			nodes[i] = node{privateKeyPEM, publicKeyPEM, fmt.Sprintf("http://localhost:%d", port), port}
		}

		nodes[0].port = getAvailablePort()
		nodes[0].address = fmt.Sprintf("http://localhost:%d", nodes[0].port)

		nodes[numNodes-1].port = getAvailablePort()
		nodes[numNodes-1].address = fmt.Sprintf("http://localhost:%d", nodes[numNodes-1].port)

		shuffled := utils.Copy(nodes[1 : numNodes-1])
		utils.Shuffle(shuffled)
		for i, node := range shuffled {
			nodes[i+1] = node
		}

		slog.Info(strings.Join(utils.Map(nodes, func(n node) string { return config.AddressToName(n.address) }), " -> "))

		secretMessage := "secret message"

		payload, err := json.Marshal(structs.NewMessage(nodes[0].address, nodes[numNodes-1].address, secretMessage))
		if err != nil {
			slog.Error("json.Marshal() error", err)
			t.Fatalf("json.Marshal() error: %v", err)
		}

		publicKeys := utils.Map(nodes, func(n node) string { return n.publicKeyPEM })
		routingPath := utils.Map(nodes, func(n node) string { return n.address })

		_, onionStr, _, err := pi_t.FormOnion(nodes[0].privateKeyPEM, nodes[0].publicKeyPEM, payload, publicKeys[1:], routingPath[1:], -1, nodes[0].address)
		if err != nil {
			t.Fatalf("FormOnion() error: %v", err)
		}

		//slog.Info("Done forming onion")

		shutdownChans := make([]chan struct{}, numNodes-1)
		for i := range shutdownChans {
			shutdownChans[i] = make(chan struct{})
		}

		var wg sync.WaitGroup
		wg.Add(numNodes - 1)

		for i := 1; i < numNodes-1; i++ {
			i := i
			go func() {
				defer wg.Done()
				mux := http.NewServeMux()
				mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
					HandleReceiveOnion(w, r, func(onionStr string) error {
						peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, nodes[i].privateKeyPEM)
						if err2 != nil {
							slog.Error("PeelOnion() error", err2)
							t.Errorf("PeelOnion() error = %v", err2)
							return err2
						} else {
							//slog.Info("PeelOnion() success", "i", i)
						}

						if peelOnion.NextHop != nodes[i+1].address {
							pl.LogNewError("PeelOnion() expected next hop '%s', got %s", nodes[i+1].address, peelOnion.NextHop)
							t.Errorf("PeelOnion() expected next hop '', got %s", peelOnion.NextHop)
							return pl.NewError("PeelOnion() expected next hop '%s', got %s", nodes[i+1].address, peelOnion.NextHop)
						}
						if peelOnion.LastHop != nodes[i-1].address {
							pl.LogNewError("PeelOnion() expected last hop '%s', got %s", nodes[i-1].address, peelOnion.LastHop)
							t.Errorf("PeelOnion() expected last hop %s, got %s", nodes[i-1].address, peelOnion.LastHop)
							return pl.NewError("PeelOnion() expected last hop '%s', got %s", nodes[i-1].address, peelOnion.LastHop)
						}

						headerAdded, err3 := pi_t.AddHeader(peelOnion, 1, nodes[i].privateKeyPEM, nodes[i].publicKeyPEM)
						if err3 != nil {
							slog.Error("AddHeader() error", err3)
							t.Errorf("AddHeader() error = %v", err3)
							return err3
						}

						err4 := SendOnion(peelOnion.NextHop, nodes[i].address, headerAdded)
						if err4 != nil {
							slog.Error("SendOnion() error", err4)
							t.Errorf("SendOnion() error = %v", err4)
							return err4
						}

						return nil
					})
				})
				server := &http.Server{
					Addr:    fmt.Sprintf(":%d", nodes[i].port),
					Handler: mux,
				}
				go func() {
					<-shutdownChans[i-1]
					server.Shutdown(context.Background())
				}()
				if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
					slog.Error("failed to start HTTP server", err2)
				}
			}()
		}

		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
				HandleReceiveOnion(w, r, func(onionStr string) error {

					defer wg.Done()
					peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, nodes[numNodes-1].privateKeyPEM)
					if err2 != nil {
						slog.Error("PeelOnion() error", err2)
						t.Errorf("PeelOnion() error = %v", err2)
						return err2
					}

					var Msg structs.Message
					err = json.Unmarshal([]byte(peelOnion.Payload), &Msg)
					if err != nil {
						slog.Error("json.Unmarshal() error", err)
						t.Errorf("json.Unmarshal() error: %v", err)
						return err
					}
					if Msg.Msg != secretMessage {
						t.Errorf("PeelOnion() expected payload %s, got %s", string(payload), peelOnion.Payload)
						return pl.NewError("PeelOnion() expected payload %s, got %s", string(payload), peelOnion.Payload)
					}
					if Msg.To != nodes[numNodes-1].address {
						t.Errorf("PeelOnion() expected to address %s, got %s", nodes[numNodes-1].address, Msg.To)
						return pl.NewError("PeelOnion() expected to address %s, got %s", nodes[numNodes-1].address, Msg.To)
					}
					if Msg.From != nodes[0].address {
						t.Errorf("PeelOnion() expected from address %s, got %s", nodes[0].address, Msg.From)
						return pl.NewError("PeelOnion() expected from address %s, got %s", nodes[0].address, Msg.From)
					}

					slog.Info("Successfully received message", "message", peelOnion.Payload)

					// Signal all servers to shut down
					for _, ch := range shutdownChans {
						close(ch)
					}

					return nil
				})
			})
			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", nodes[numNodes-1].port),
				Handler: mux,
			}
			go func() {
				<-shutdownChans[numNodes-2]
				server.Shutdown(context.Background())
			}()
			if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
				slog.Error("failed to start HTTP server", err2)
				t.Errorf("failed to start HTTP server: %v", err2)
			}
		}()

		err = SendOnion(nodes[1].address, nodes[0].address, onionStr)
		if err != nil {
			slog.Error("SendOnion() error", err)
			t.Fatalf("SendOnion() error = %v", err)
		}

		wg.Wait()
	}
}
