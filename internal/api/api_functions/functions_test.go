package api_functions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
)

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

func TestReceiveOnionMultipleLayers(t *testing.T) {
	for nnn := 0; nnn < 10; nnn++ {
		pl.SetUpLogrusAndSlog("Error")

		if err, _ := config.InitGlobal(); err != nil {
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

		slog.Info(strings.Join(utils.Map(nodes, func(n node) string { return n.address }), " -> "))

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

		onions, err := pi_t.FORMONION(string(payload), routingPath[:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys, metadata, d)
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
					HandleReceiveOnion(w, r, func(oApi structs.OnionApi) error {
						onionStr := oApi.Onion
						_, layer, _, peeled, nextDestination, err2 := pi_t.PeelOnion(onionStr, nodes[i].privateKeyPEM)
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
							peeled.Sepal = peeled.Sepal.RemoveBlock()
						}

						err4 := SendOnion(nextDestination, nodes[i].address, peeled, -1)
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
				HandleReceiveOnion(w, r, func(oApi structs.OnionApi) error {
					onionStr := oApi.Onion
					defer wg.Done()
					_, layer, _, peeled, _, err2 := pi_t.PeelOnion(onionStr, nodes[l].privateKeyPEM)
					if err2 != nil {
						slog.Error("PeelOnion() error", err2)
						t.Errorf("PeelOnion() error = %v", err2)
						return err2
					}

					payload := string(peeled.Content)

					if layer != l {
						t.Errorf("PeelOnion() expected layer %d, got %d", l, layer)
						return pl.NewError("PeelOnion() expected layer %d, got %d", l, layer)
					}

					var Msg structs.Message
					err = json.Unmarshal([]byte(payload), &Msg)
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

		err = SendOnion(nodes[1].address, nodes[0].address, onions[0][0], -1)
		if err != nil {
			slog.Error("SendOnion() error", err)
			t.Fatalf("SendOnion() error = %v", err)
		}

		wg.Wait()
	}
}
