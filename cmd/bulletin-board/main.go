package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/cmd/bulletin-board/config"
	"github.com/HannahMarsh/pi_t-experiment/cmd/global_config"
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
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	global_cfg, err := global_config.NewConfig()
	if err != nil {
		slog.Error("failed get global config", err)
		os.Exit(1)
	}

	host := global_cfg.BulletinBoard.Host
	port := global_cfg.BulletinBoard.Port

	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("failed get bulletin board config", err)
	}

	slog.Info("⚡ init Bulletin board")

	// set up logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(cfg.LogLevel))

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	bulletinBoardRepo := &repositories.BulletinBoardRepositoryImpl{}
	bulletinBoardService := &usecases.BulletinBoardService{
		Repo:     bulletinBoardRepo,
		Interval: time.Duration(global_cfg.HeartbeatInterval) * time.Second, // Interval for each run
	}
	bulletinBoardHandler := &handlers.BulletinBoardHandler{
		Service: bulletinBoardService,
	}

	go bulletinBoardHandler.StartRuns()

	http.HandleFunc("/register", bulletinBoardHandler.RegisterNode)

	go func() {
		address := fmt.Sprintf(":%d", port)
		if err := http.ListenAndServe(address, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err)
		}
	}()

	slog.Info("🌏 start node...", "address", fmt.Sprintf("%s:%d", host, port))

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
