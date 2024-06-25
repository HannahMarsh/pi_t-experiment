package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/bulletin_board"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	host := cfg.BulletinBoard.Host
	port := cfg.BulletinBoard.Port
	url := fmt.Sprintf("https://%s:%d", host, port)

	slog.Info("⚡ init Bulletin board")

	bulletinBoard := bulletin_board.NewBulletinBoard(cfg)

	go func() {
		err := bulletinBoard.StartRuns()
		if err != nil {
			slog.Error("failed to start runs", err)
			config.GlobalCancel()
		}
	}()

	http.HandleFunc("/registerNode", bulletinBoard.HandleRegisterNode)
	http.HandleFunc("/registerClient", bulletinBoard.HandleRegisterClient)
	http.HandleFunc("/registerIntentToSend", bulletinBoard.HandleRegisterIntentToSend)
	http.HandleFunc("/updateNode", bulletinBoard.HandleUpdateNodeInfo)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 starting bulletin board...", "address", url)

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
