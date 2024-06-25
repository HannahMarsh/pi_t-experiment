package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	_ "github.com/lib/pq"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	//isMixer := flag.Bool("mixer", false, "Included if this node is a mixer")
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	pl.SetUpLogrusAndSlog(*logLevel)

	// set GOMAXPROCS
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	cfg := config.GlobalConfig

	slog.Info("⚡ init metrics", "host", cfg.Metrics.Host, "port", cfg.Metrics.Port)

	http.HandleFunc("/data", serveData)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.Handle("/clients", http.FileServer(http.Dir("./static/client")))

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Metrics.Port), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 start metrics...", "address", fmt.Sprintf(" %s:%d ", cfg.Metrics.Host, cfg.Metrics.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		config.GlobalCancel()
		slog.Info("signal.Notify", v)
	case done := <-config.GlobalCtx.Done():
		slog.Info("ctx.Done", done)
	}

}

type Data struct {
	Clients  map[string]api.ClientStatus
	Messages []Message
	mu       sync.RWMutex
}

type Message struct {
	From         string
	To           string
	RoutingPath  []api.PublicNodeApi
	Msg          string
	TimeSent     time.Time
	TimeReceived time.Time
}

var (
	data Data = Data{
		Clients:  make(map[string]api.ClientStatus),
		Messages: make([]Message, 0),
	}
)

func serveData(w http.ResponseWriter, r *http.Request) {
	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	data.mu.Lock()
	defer data.mu.Unlock()

	for _, client := range config.GlobalConfig.Clients {
		addr := fmt.Sprintf("http://%s:%d/status", client.Host, client.Port)
		resp, err := http.Get(addr)
		if err != nil {
			slog.Error("failed to get client status", err)
		} else {
			defer resp.Body.Close()
			var status api.ClientStatus
			if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
				slog.Error("failed to decode client status", err)
			} else {
				data.Clients[addr] = status
			}
		}
	}

	m := make(map[string]Message)

	for _, client := range data.Clients {
		for _, sent := range client.MessagesSent {
			mstr := sent.Message.From + sent.Message.To + sent.Message.Msg
			if _, present := m[mstr]; present {
				msg := Message{
					From:         sent.Message.From,
					To:           sent.Message.To,
					RoutingPath:  sent.RoutingPath,
					Msg:          sent.Message.Msg,
					TimeSent:     sent.TimeSent,
					TimeReceived: m[sent.Message.Msg].TimeReceived,
				}
				m[mstr] = msg
			} else {
				m[mstr] = Message{
					From:         sent.Message.From,
					To:           sent.Message.To,
					RoutingPath:  sent.RoutingPath,
					Msg:          sent.Message.Msg,
					TimeSent:     sent.TimeSent,
					TimeReceived: time.Time{},
				}
			}
		}
		for _, received := range client.MessagesReceived {
			mstr := received.Message.From + received.Message.To + received.Message.Msg
			if _, present := m[mstr]; present {
				msg := Message{
					From:         received.Message.From,
					To:           received.Message.To,
					RoutingPath:  m[mstr].RoutingPath,
					Msg:          received.Message.Msg,
					TimeSent:     m[mstr].TimeSent,
					TimeReceived: received.TimeReceived,
				}
				m[mstr] = msg
			} else {
				m[mstr] = Message{
					From:         received.Message.From,
					To:           received.Message.To,
					RoutingPath:  make([]api.PublicNodeApi, 0),
					Msg:          received.Message.Msg,
					TimeSent:     time.Time{},
					TimeReceived: received.TimeReceived,
				}
			}
		}
	}

	data.Messages = make([]Message, 0)
	for _, msg := range m {
		data.Messages = append(data.Messages, msg)
	}

	// Encode the data as JSON and write to the response
	str, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal data", err)
	} else {
		if _, err = w.Write(str); err != nil {
			slog.Error("failed to write data", err)
		}
	}
}
