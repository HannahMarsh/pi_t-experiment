package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/cmd/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/bulletin_board"
	"github.com/HannahMarsh/pi_t-experiment/pkg/infrastructure/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"

	_ "github.com/lib/pq"
)

func main() {
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// set GOMAXPROCS
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := config.InitConfig(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	cfg := config.GlobalConfig

	host := cfg.BulletinBoard.Host
	port := cfg.BulletinBoard.Port

	slog.Info("⚡ init Bulletin board")

	// set up logrus
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(*logLevel))

	// Add stack trace hook
	logrus.AddHook(&logger.ErrorsStackHook{})

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	bulletinBoard := bulletin_board.NewBulletinBoard(cfg)

	go bulletinBoard.StartRuns()

	http.HandleFunc("/register", bulletinBoard.HandleRegisterNode)
	http.HandleFunc("/update", bulletinBoard.HandleUpdateNodeInfo)
	http.HandleFunc("/nodes", bulletinBoard.HandleGetActiveNodes)

	go func() {
		address := fmt.Sprintf(":%d", port)
		if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	slog.Info("🌏 start node...", "address", fmt.Sprintf("https://%s:%d", host, port))

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
