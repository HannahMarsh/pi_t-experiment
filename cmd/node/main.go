package main

import (
	"context"
	"fmt"
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
	// set GOMAXPROCS
	_, err := maxprocs.Set()
	if err != nil {
		slog.Error("failed set max procs", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("failed get config", err)
	}

	slog.Info("⚡ init app", "name", cfg.Name, "version", cfg.Version)

	// set up logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(cfg.Log.Level))

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	nodeRepo := &repositories.NodeRepositoryImpl{}
	nodeService := &usecases.NodeService{
		repo:     nodeRepo,
		interval: 10 * time.Second, // Interval for each run
	}
	nodeHandler := &handlers.NodeHandler{
		service: nodeService,
	}

	go nodeHandler.StartActions()

	http.HandleFunc("/register", nodeHandler.RegisterNode)
	http.ListenAndServe(":8081", nil)

	slog.Info("🌏 start node...", "address", fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		cleanup()
		slog.Info("signal.Notify", v)
	case done := <-ctx.Done():
		cleanup()
		slog.Info("ctx.Done", done)
	}
}
