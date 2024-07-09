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
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
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
	//pl.SetUpLogrusAndSlog("debug")
	//
	//if err := config.InitGlobal(); err != nil {
	//	slog.Error("failed to init config", err)
	//	os.Exit(1)
	//}
	//
	//receiverPort := getAvailablePort()
	//
	//privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	//if err != nil {
	//	t.Fatalf("KeyGen() error: %v", err)
	//}
	//
	//payload := []byte("secret message")
	//publicKeys := []string{publicKeyPEM, publicKeyPEM}
	//routingPath := []string{fmt.Sprintf("http://localhost:%d", receiverPort), "node2"}
	//
	//o, err := pi_t.FORMONION(publicKeyPEM, privateKeyPEM, base64.StdEncoding.EncodeToString(payload), mixersAddr, gatekeepersAddr, destination.Address, publicKeys, metadata, config.GlobalConfig.D)
	//
	//addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1, "")
	//
	//if err != nil {
	//	slog.Error("FormOnion() error", err)
	//	t.Fatalf("FormOnion() error = %v", err)
	//}
	//
	//http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
	//	HandleReceiveOnion(w, r, func(onionStr string) error {
	//		peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM)
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
	//		return nil
	//	})
	//})
	//
	//go func() {
	//	address := fmt.Sprintf(":%d", receiverPort)
	//	if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
	//		slog.Error("failed to start HTTP server", err2)
	//	}
	//}()
	//
	//err = SendOnion(addr, "sender", onion)
	//if err != nil {
	//	slog.Error("SendOnion() error", err)
	//	t.Fatalf("SendOnion() error = %v", err)
	//}
}

func TestReceiveOnionMultipleLayers(t *testing.T) {
	//pl.SetUpLogrusAndSlog("debug")
	//
	//if err := config.InitGlobal(); err != nil {
	//	slog.Error("failed to init config", err)
	//	os.Exit(1)
	//}
	//
	//receiverPort1 := getAvailablePort()
	//receiverPort2 := getAvailablePort()
	//
	//var err error
	//
	//l1 := 5
	//l2 := 5
	//d := 3
	//l := l1 + l2 + 1
	//
	//type node struct {
	//	privateKeyPEM string
	//	publicKeyPEM  string
	//	address       string
	//}
	//
	//nodes := make([]node, l+1)
	//
	//for i := 0; i < l+1; i++ {
	//	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	//	if err != nil {
	//		t.Fatalf("KeyGen() error: %v", err)
	//	}
	//	nodes[i] = node{privateKeyPEM, publicKeyPEM, fmt.Sprintf("node%d", i)}
	//}
	//
	//secretMessage := "secret message"
	//
	//payload, err := json.Marshal(structs.Message{
	//	Msg:  secretMessage,
	//	To:   nodes[l].address,
	//	From: nodes[0].address,
	//})
	//if err != nil {
	//	slog.Error("json.Marshal() error", err)
	//	t.Fatalf("json.Marshal() error: %v", err)
	//}
	//
	//publicKeys := utils.Map(nodes[1:], func(n node) string { return n.publicKeyPEM })
	//routingPath := utils.Map(nodes[1:], func(n node) string { return n.address })
	//
	//metadata := make([]onion_model.Metadata, l+1)
	//for i := 0; i < l+1; i++ {
	//	metadata[i] = onion_model.Metadata{Example: fmt.Sprintf("example%d", i)}
	//}
	//
	//onions, err := pi_t.FORMONION(nodes[0].publicKeyPEM, nodes[0].privateKeyPEM, string(payload), routingPath[:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys, metadata, d)
	//if err != nil {
	//	slog.Error("FormOnion() error", err)
	//	t.Fatalf("FormOnion() error = %v", err)
	//}
	//
	//go func() {
	//	mux := http.NewServeMux()
	//	mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
	//		HandleReceiveOnion(w, r, func(onionStr string) error {
	//			peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM1)
	//			if err2 != nil {
	//				slog.Error("PeelOnion() error", err2)
	//				t.Fatalf("PeelOnion() error = %v", err2)
	//			}
	//
	//			headerAdded, err3 := pi_t.AddHeader(peelOnion, 1, privateKeyPEM1, publicKeyPEM1)
	//			if err3 != nil {
	//				slog.Error("AddHeader() error", err3)
	//				t.Fatalf("AddHeader() error = %v", err3)
	//			}
	//
	//			err4 := SendOnion(peelOnion.NextHop, fmt.Sprintf("http://localhost:%d", receiverPort1), headerAdded)
	//			if err4 != nil {
	//				slog.Error("SendOnion() error", err4)
	//				t.Fatalf("SendOnion() error = %v", err4)
	//			}
	//
	//			return nil
	//		})
	//	})
	//	server := &http.Server{
	//		Addr:    fmt.Sprintf(":%d", receiverPort1),
	//		Handler: mux,
	//	}
	//	if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
	//		slog.Error("failed to start HTTP server", err2)
	//	}
	//}()
	//
	//go func() {
	//	mux := http.NewServeMux()
	//	mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
	//		HandleReceiveOnion(w, r, func(onionStr string) error {
	//			peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM2)
	//			if err2 != nil {
	//				slog.Error("PeelOnion() error", err2)
	//				t.Fatalf("PeelOnion() error = %v", err2)
	//			}
	//
	//			if peelOnion.Payload != "secret message" {
	//				t.Fatalf("Expected onion.Onion to be 'test onion data', got %s", peelOnion.Payload)
	//			}
	//
	//			slog.Info("Successfully received message", "message", peelOnion.Payload)
	//
	//			return nil
	//		})
	//	})
	//	server := &http.Server{
	//		Addr:    fmt.Sprintf(":%d", receiverPort2),
	//		Handler: mux,
	//	}
	//	if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
	//		slog.Error("failed to start HTTP server", err2)
	//	}
	//}()
	//
	//err = SendOnion(addr, "sender", onion)
	//if err != nil {
	//	slog.Error("SendOnion() error", err)
	//	t.Fatalf("SendOnion() error = %v", err)
	//}
}

func TestReceiveCheckpointOnions(t *testing.T) {
	//pl.SetUpLogrusAndSlog("debug")
	//
	//if err := config.InitGlobal(); err != nil {
	//	slog.Error("failed to init config", err)
	//	os.Exit(1)
	//}
	//
	//receiverPort1 := getAvailablePort()
	//receiverPort2 := getAvailablePort()
	//
	//privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	//if err != nil {
	//	t.Fatalf("KeyGen() error: %v", err)
	//}
	//privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
	//if err != nil {
	//	t.Fatalf("KeyGen() error: %v", err)
	//}
	//privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
	//if err != nil {
	//	t.Fatalf("KeyGen() error: %v", err)
	//}
	//
	//msg := structs.Message{
	//	Msg: "secret message",
	//}
	//payload, err := json.Marshal(msg)
	//if err != nil {
	//	slog.Error("Marshal() error", err)
	//	t.Fatalf("Marshal() error: %v", err)
	//}
	//publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
	//routingPath := []string{fmt.Sprintf("http://localhost:%d", receiverPort1), fmt.Sprintf("http://localhost:%d", receiverPort2)}
	//
	//addr, onion, _, err := pi_t.FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, 0, "")
	//
	//if err != nil {
	//	slog.Error("FormOnion() error", err)
	//	t.Fatalf("FormOnion() error = %v", err)
	//}
	//
	//var wg sync.WaitGroup
	//wg.Add(2)
	//
	//go func() {
	//	mux := http.NewServeMux()
	//	mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
	//		HandleReceiveOnion(w, r, func(onionStr string) error {
	//			defer wg.Done()
	//			peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM1)
	//			if err2 != nil {
	//				slog.Error("PeelOnion() error", err2)
	//				t.Fatalf("PeelOnion() error = %v", err2)
	//			}
	//
	//			headerAdded, err3 := pi_t.AddHeader(peelOnion, 1, privateKeyPEM1, publicKeyPEM1)
	//			if err3 != nil {
	//				slog.Error("AddHeader() error", err3)
	//				t.Fatalf("AddHeader() error = %v", err3)
	//			}
	//
	//			err4 := SendOnion(peelOnion.NextHop, fmt.Sprintf("http://localhost:%d", receiverPort1), headerAdded)
	//			if err4 != nil {
	//				slog.Error("SendOnion() error", err4)
	//				t.Fatalf("SendOnion() error = %v", err4)
	//			}
	//
	//			return nil
	//		})
	//	})
	//	server := &http.Server{
	//		Addr:    fmt.Sprintf(":%d", receiverPort1),
	//		Handler: mux,
	//	}
	//	if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
	//		slog.Error("failed to start HTTP server", err2)
	//	}
	//}()
	//
	//go func() {
	//	mux := http.NewServeMux()
	//	mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
	//		HandleReceiveOnion(w, r, func(onionStr string) error {
	//			defer wg.Done()
	//			peelOnion, _, _, _, err2 := pi_t.PeelOnion(onionStr, privateKeyPEM2)
	//			if err2 != nil {
	//				slog.Error("PeelOnion() error", err2)
	//				t.Fatalf("PeelOnion() error = %v", err2)
	//			}
	//
	//			var msg structs.Message
	//
	//			err := json.Unmarshal([]byte(peelOnion.Payload), &msg)
	//			if err != nil {
	//				slog.Error("Unmarshal() error", err)
	//				t.Fatalf("Unmarshal() error: %v", err)
	//			}
	//
	//			if msg.Msg != "secret message" {
	//				t.Fatalf("Expected onion.Onion to be 'test onion data', got %s", msg.Msg)
	//			}
	//
	//			slog.Info("Successfully received message", "message", msg.Msg)
	//
	//			return nil
	//		})
	//	})
	//	server := &http.Server{
	//		Addr:    fmt.Sprintf(":%d", receiverPort2),
	//		Handler: mux,
	//	}
	//	if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
	//		slog.Error("failed to start HTTP server", err2)
	//	}
	//}()
	//
	//time.Sleep(1 * time.Second)
	//
	//err = SendOnion(addr, "sender", onion)
	//if err != nil {
	//	slog.Error("SendOnion() error", err)
	//	t.Fatalf("SendOnion() error = %v", err)
	//}
	//
	//wg.Wait()
}

func TestReceiveOnionMultipleLayers2(t *testing.T) {
	for nnn := 0; nnn < 100; nnn++ {
		pl.SetUpLogrusAndSlog("debug")

		if err := config.InitGlobal(); err != nil {
			slog.Error("failed to init config", err)
			os.Exit(1)
		}

		var err error

		l1 := 5
		l2 := 5
		d := 3
		l := l1 + l2 + 1

		type node struct {
			privateKeyPEM string
			publicKeyPEM  string
			address       string
			port          int
		}

		nodes := make([]node, l+1)

		for i := range nodes {
			privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
			if err != nil {
				t.Fatalf("KeyGen() error: %v", err)
			}
			port := getAvailablePort()
			nodes[i] = node{privateKeyPEM, publicKeyPEM, fmt.Sprintf("http://localhost:%d", port), port}
		}

		slog.Info(strings.Join(utils.Map(nodes, func(n node) string { return config.AddressToName(n.address) }), " -> "))

		secretMessage := "secret message"

		MsgStruct := structs.NewMessage(nodes[0].address, nodes[l].address, secretMessage)
		payload, err := json.Marshal(MsgStruct)
		if err != nil {
			slog.Error("json.Marshal() error", err)
			t.Fatalf("json.Marshal() error: %v", err)
		}

		publicKeys := utils.Map(nodes[1:], func(n node) string { return n.publicKeyPEM })
		routingPath := utils.Map(nodes[1:], func(n node) string { return n.address })

		metadata := make([]onion_model.Metadata, l+1)
		for i := 0; i < l+1; i++ {
			metadata[i] = onion_model.Metadata{Example: fmt.Sprintf("example%d", i)}
		}

		onions, err := pi_t.FORMONION(nodes[0].publicKeyPEM, nodes[0].privateKeyPEM, string(payload), routingPath[:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys, metadata, d)
		if err != nil {
			slog.Error("", err)
			t.Fatalf("failed")
		}
		//slog.Info("Done forming onion")

		shutdownChans := make([]chan struct{}, l)
		for i := range shutdownChans {
			shutdownChans[i] = make(chan struct{})
		}

		var wg sync.WaitGroup
		wg.Add(l)

		for i := 1; i < l; i++ {
			i := i
			go func() {
				defer wg.Done()
				mux := http.NewServeMux()
				mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
					HandleReceiveOnion(w, r, func(onionStr string) error {
						sharedKey, err := keys.ComputeSharedKey(nodes[i].privateKeyPEM, nodes[0].publicKeyPEM)
						if err != nil {
							slog.Error("ComputeSharedKey() error", err)
							t.Errorf("ComputeSharedKey() error = %v", err)
							return err
						}
						layer, _, peeled, nextDestination, err2 := pi_t.PeelOnion(onionStr, sharedKey)
						if err2 != nil {
							slog.Error("PeelOnion() error", err2)
							t.Errorf("PeelOnion() error = %v", err2)
							return err2
						} else {
							//slog.Info("PeelOnion() success", "i", i)
						}

						if nextDestination != nodes[i+1].address {
							pl.LogNewError("PeelOnion() expected next hop '%s', got %s", nodes[i+1].address, nextDestination)
							t.Errorf("PeelOnion() expected next hop '', got %s", nextDestination)
							return pl.NewError("PeelOnion() expected next hop '%s', got %s", nodes[i+1].address, nextDestination)
						}

						if layer != i {
							pl.LogNewError("PeelOnion() expected layer %d, got %d", i, layer)
							t.Errorf("PeelOnion() expected layer %d, got %d", i, layer)
							return pl.NewError("PeelOnion() expected layer %d, got %d", i, layer)
						}
						if i < l1 {
							peeled.Sepal.RemoveBlock()
						}

						err4 := SendOnion(nextDestination, nodes[i].address, peeled)
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
					sharedKey, err := keys.ComputeSharedKey(nodes[l].privateKeyPEM, nodes[0].publicKeyPEM)
					if err != nil {
						slog.Error("ComputeSharedKey() error", err)
						t.Errorf("ComputeSharedKey() error = %v", err)
						return err
					}
					layer, _, peeled, _, err2 := pi_t.PeelOnion(onionStr, sharedKey)
					if err2 != nil {
						slog.Error("PeelOnion() error", err2)
						t.Errorf("PeelOnion() error = %v", err2)
						return err2
					}

					payload, err := base64.StdEncoding.DecodeString(string(peeled.Content))
					if err != nil {
						slog.Error("base64.StdEncoding.DecodeString() error", err)
						t.Errorf("base64.StdEncoding.DecodeString() error: %v", err)
						return pl.WrapError(err, "base64.StdEncoding.DecodeString() error")
					}

					if layer != l {
						t.Errorf("PeelOnion() expected layer %d, got %d", l, layer)
						return pl.NewError("PeelOnion() expected layer %d, got %d", l, layer)
					}

					var Msg structs.Message
					err = json.Unmarshal(payload, &Msg)
					if err != nil {
						slog.Error("json.Unmarshal() error", err)
						t.Errorf("json.Unmarshal() error: %v", err)
						return err
					}
					if Msg.Msg != secretMessage {
						t.Errorf("PeelOnion() expected payload %s, got %s", secretMessage, Msg.Msg)
						return pl.NewError("PeelOnion() expected payload %s, got %s", secretMessage, Msg.Msg)
					}
					if Msg.To != nodes[l].address {
						t.Errorf("PeelOnion() expected to address %s, got %s", nodes[l].address, Msg.To)
						return pl.NewError("PeelOnion() expected to address %s, got %s", nodes[l].address, Msg.To)
					}
					if Msg.From != nodes[0].address {
						t.Errorf("PeelOnion() expected from address %s, got %s", nodes[0].address, Msg.From)
						return pl.NewError("PeelOnion() expected from address %s, got %s", nodes[0].address, Msg.From)
					}
					if Msg.Hash != MsgStruct.Hash {
						t.Errorf("PeelOnion() expected hash %s, got %s", MsgStruct.Hash, Msg.Hash)
						return pl.NewError("PeelOnion() expected hash %s, got %s", MsgStruct.Hash, Msg.Hash)
					}

					slog.Info("Successfully received message", "message", Msg.Msg)

					// Signal all servers to shut down
					for _, ch := range shutdownChans {
						close(ch)
					}

					return nil
				})
			})
			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", nodes[l].port),
				Handler: mux,
			}
			go func() {
				<-shutdownChans[l-1]
				server.Shutdown(context.Background())
			}()
			if err2 := server.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
				slog.Error("failed to start HTTP server", err2)
				t.Errorf("failed to start HTTP server: %v", err2)
			}
		}()

		err = SendOnion(nodes[1].address, nodes[0].address, onions[0][0])
		if err != nil {
			slog.Error("SendOnion() error", err)
			t.Fatalf("SendOnion() error = %v", err)
		}

		wg.Wait()
	}
}
