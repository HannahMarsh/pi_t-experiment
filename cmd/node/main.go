package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/cmd/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/usecases"
	"github.com/HannahMarsh/pi_t-experiment/pkg/api/handlers"
	"github.com/HannahMarsh/pi_t-experiment/pkg/infrastructure/logger"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
)

func main() {
	// Define command-line flags
	id := flag.Int("id", -1, "ID of the node (required)")
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check if the required flag is provided
	if *id == 0 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: the -id flag is required\n"); err != nil {
			slog.Error("Error: the -id flag is required\n", err)
		}
		flag.Usage()
		os.Exit(2)
	}

	// set GOMAXPROCS
	_, err := maxprocs.Set()
	if err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("failed get config", err)
		os.Exit(1)
	}

	var nodeConfig *config.Node
	for _, node := range cfg.Nodes {
		if node.ID == *id {
			nodeConfig = &node
			break
		}
	}

	if nodeConfig == nil {
		slog.Error("invalid id", errors.New(fmt.Sprintf("failed to get node config for id=%d", *id)))
		os.Exit(1)
	}

	slog.Info("⚡ init node", "heartbeat_interval", cfg.HeartbeatInterval, "id", *id)

	// set up logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(*logLevel))

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	nodeHandler := &handlers.NodeHandler{
		Service: usecases.Init(nodeConfig.ID, nodeConfig.Host, nodeConfig.Port, nodeConfig.PublicKey),
	}

	http.HandleFunc("/receive", nodeHandler.Receive)

	go func() {
		address := fmt.Sprintf(":%d", nodeConfig.Port)
		if err := http.ListenAndServe(address, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err)
		}
	}()

	slog.Info("🌏 start node...", "address", fmt.Sprintf("%s:%d", nodeConfig.Host, nodeConfig.Port))

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
