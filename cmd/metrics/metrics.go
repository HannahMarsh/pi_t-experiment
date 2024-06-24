package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/metrics"
	_ "github.com/lib/pq"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	http.HandleFunc("/client/updateMessageQueue", metrics.HandleUpdateMessageQueue)
	http.HandleFunc("/startRun", metrics.HandleStartRun)
	http.HandleFunc("/client/sentOnion", metrics.HandleClientSentOnion)
	http.HandleFunc("/client/sentOnion", metrics.HandleClientReceivedOnion)
	http.HandleFunc("/node/sentOnion", metrics.HandleNodeSentOnion)
	http.HandleFunc("/node/sentOnion", metrics.HandleNodeReceivedOnion)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Metrics.Port), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 start metrics...", "address", fmt.Sprintf("%s:%d", cfg.Metrics.Host, cfg.Metrics.Port))

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
