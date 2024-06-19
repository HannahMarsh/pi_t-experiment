package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/cmd/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/pkg/infrastructure/logger"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Define command-line flags
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// set GOMAXPROCS
	_, err := maxprocs.Set()
	if err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err = config.InitConfig(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	cfg := config.GlobalConfig

	slog.Info("⚡ init client", "heartbeat_interval", cfg.HeartbeatInterval)

	// set up logrus
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(*logLevel))

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	node_addresses := make(map[int][]string, 0)
	for _, n := range cfg.Nodes {
		if _, ok := node_addresses[n.ID]; !ok {
			node_addresses[n.ID] = make([]string, 0)
		}
		node_addresses[n.ID] = append(node_addresses[n.ID], fmt.Sprintf("http://%s:%d", n.Host, n.Port))
	}

	for {
		for id, addresses := range node_addresses {
			for _, addr := range addresses {
				addr := addr
				go func() {
					var msgs []api.Message = make([]api.Message, 0)
					for i, _ := range node_addresses {
						if i != id {
							msgs = append(msgs, api.Message{
								From: id,
								To:   i,
								Msg:  fmt.Sprintf("msg %d", i),
							})
						}
					}
					if data, err2 := json.Marshal(msgs); err2 != nil {
						slog.Error("failed to marshal msgs", err2)
					} else {
						url := addr + "/requestMsg"
						slog.Info("Sending add msg request.", "url", url, "num_onions", len(msgs))
						if resp, err3 := http.Post(url, "application/json", bytes.NewBuffer(data)); err3 != nil {
							slog.Error("failed to send POST request with msgs to node", err3)
						} else {
							defer func(Body io.ReadCloser) {
								if err4 := Body.Close(); err4 != nil {
									fmt.Printf("error closing response body: %v\n", err2)
								}
							}(resp.Body)
							if resp.StatusCode != http.StatusCreated {
								slog.Info("failed to send msgs to node", "status_code", resp.StatusCode, "status", resp.Status)
							}
						}
					}
				}()
			}
		}
		time.Sleep(time.Duration(2 * time.Second))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		cancel()
		slog.Info("signal.Notify", v)
	case done := <-ctx.Done():
		slog.Info("ctx.Done", done)
	}

}
