package metrics

import (
	"context"
	"errors"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
	"net/http"
	"time"
)

var (
	PROCESSING_TIME = "onionProcessingTime"
	ONION_COUNT     = "onionCounter"
	MSG_SENT        = "messageSentTimestamp"
	MSG_RECEIVED    = "messageReceivedTimestamp"
)

var collectors = map[string]prometheus.Collector{
	PROCESSING_TIME: prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    PROCESSING_TIME,
		Help:    "Processing time of onions in seconds",
		Buckets: prometheus.DefBuckets,
	}),
	ONION_COUNT: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: ONION_COUNT,
			Help: "Number of onions received across all rounds",
		},
		[]string{"round"},
	),
	MSG_SENT: prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MSG_SENT,
			Help: "Unix timestamp of when each message was sent, labeled by message hash",
		},
		[]string{"hash"}, // Label to distinguish messages by their hash
	),
	MSG_RECEIVED: prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: MSG_RECEIVED,
			Help: "Unix timestamp of when each message was sent, labeled by message hash",
		},
		[]string{"hash"}, // Label to distinguish messages by their hash
	),
}

func Observe(id string, value float64) {
	if collector, ok := collectors[id].(prometheus.Observer); ok {
		collector.Observe(value)
	} else {
		pl.LogNewError("Failed to find collector with id: " + id)
	}
}

func Inc(id string, labels ...any) {
	if labels == nil || len(labels) == 0 {
		if collector, ok := collectors[id].(prometheus.Counter); ok {
			collector.Inc()
		} else {
			pl.LogNewError("Failed to find counter with id: " + id)
		}
	}
	if collector, ok := collectors[id].(*prometheus.CounterVec); ok {
		collector.WithLabelValues(utils.Map(labels, func(label any) string {
			return fmt.Sprintf("%v", label)
		})...).Inc()
	} else {
		pl.LogNewError("Failed to find counterVec with id: " + id)
	}
}

func Set(id string, value float64, labels ...string) {
	if labels == nil || len(labels) == 0 {
		if collector, ok := collectors[id].(prometheus.Gauge); ok {
			collector.Set(value)
		} else {
			pl.LogNewError("Failed to find gauge with id: " + id)
		}
	} else {
		if collector, ok := collectors[id].(*prometheus.GaugeVec); ok {
			collector.WithLabelValues(labels...).Set(value)
		} else {
			pl.LogNewError("Failed to find gaugeVec with id: " + id)
		}
	}
}

func ServeMetrics(prometheusPort int, collectorIds ...string) (shutdown func()) {
	// Register the histogram with Prometheus
	for _, id := range collectorIds {
		if collector, ok := collectors[id]; ok {
			prometheus.MustRegister(collector)
		} else {
			pl.LogNewError("Failed to find collector with id: " + id)
		}
	}
	// Create a new ServeMux and register the /metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", prometheusPort), // Bind to the specified port
		Handler: mux,                                // Use the mux with the /metrics endpoint
	}

	// Run the first server in a goroutine
	go func(server *http.Server) {
		slog.Info("Starting Prometheus server", "Addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Failed to start Prometheus server", err)
		}
	}(server)

	return func() {
		// Graceful shutdown
		slog.Info("Shutting down Prometheus server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Prometheus server forced to shutdown", err)
		} else {
			slog.Info("Prometheus server gracefully stopped")
		}
	}
}
