package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/metrics"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/relay"
	"go.uber.org/automaxprocs/maxprocs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Define command-line flags
	id := flag.Int("id", -1, "ID of the new relay (required)")
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check if the required flag is provided
	if *id == -1 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: the -id flag is required\n")
		flag.Usage()
		os.Exit(2)
	}

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

	var relayConfig *config.Relay
	for _, n := range cfg.Relays {
		if n.ID == *id {
			relayConfig = &n
			break
		}
	}

	if relayConfig == nil {
		slog.Error("invalid id", errors.New(fmt.Sprintf("failed to get newRelay config for id=%d", *id)))
		os.Exit(1)
	}

	slog.Info("⚡ init newRelay", "id", *id)

	baddress := fmt.Sprintf("http://%s:%d", cfg.BulletinBoard.Host, cfg.BulletinBoard.Port)

	var newRelay *relay.Relay
	for {
		if n, err := relay.NewRelay(relayConfig.ID, relayConfig.Host, relayConfig.Port, baddress); err != nil {
			slog.Error("failed to create newRelay. Trying again in 5 seconds. ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			newRelay = n
			break
		}
	}

	http.HandleFunc("/receive", newRelay.HandleReceiveOnion)
	http.HandleFunc("/start", newRelay.HandleStartRun)
	http.HandleFunc("/status", newRelay.HandleGetStatus)

	shutdownMetrics := metrics.ServeMetrics(relayConfig.PrometheusPort, metrics.PROCESSING_TIME, metrics.ONION_COUNT)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", relayConfig.Port), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 start newRelay...", "address", fmt.Sprintf("http://%s:%d", relayConfig.Host, relayConfig.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		config.GlobalCancel()
		shutdownMetrics()
		slog.Info("", "signal.Notify", v)
	case done := <-config.GlobalCtx.Done():
		slog.Info("", "ctx.Done", done)
	}

}
