package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/cmd/global_config"
	"github.com/HannahMarsh/pi_t-experiment/cmd/node/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/repositories"
	"github.com/HannahMarsh/pi_t-experiment/internal/usecases"
	"github.com/HannahMarsh/pi_t-experiment/pkg/api/handlers"
	"github.com/HannahMarsh/pi_t-experiment/pkg/infrastructure/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"

	_ "github.com/lib/pq"
)

func main() {
	// Define command-line flags
	id := flag.Int("id", -1, "ID of the node (required)")
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0]);
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check if the required flag is provided
	if *id == 0 {
		if _, err := fmt.Fprintf(os.Stderr, "Error: the -id flag is required\n"); err != nil {
			slog.Error("Error: the -id flag is required\n", err);
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

	global_cfg, err := global_config.NewConfig()
	if err != nil {
		slog.Error("failed get global config", err)
		os.Exit(1)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("failed get node config", err)
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

	slog.Info("⚡ init node", "heartbeat_interval", global_cfg.HeartbeatInterval, "id", *id)

	// set up logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(*logLevel))

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	nodeRepo := &repositories.NodeRepositoryImpl{}
	nodeService := &usecases.NodeService{
		Repo:     nodeRepo,
		Interval: time.Duration(global_cfg.HeartbeatInterval) * time.Second, // Interval for each run
	}
	nodeHandler := &handlers.NodeHandler{
		Service: nodeService,
	}

	go nodeHandler.StartActions()

	http.HandleFunc("/receive", nodeHandler.)

	go func() {
		address := fmt.Sprintf(":%d", nodeConfig.Port)
		if err := http.ListenAndServe(address, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err)
		}
	}()

	slog.Info("🌏 start node...", "address", fmt.Sprintf("%s:%d", nodeConfig.IP, nodeConfig.Port))

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
