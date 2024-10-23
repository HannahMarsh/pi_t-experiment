package metrics

import (
	"context"
	"errors"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"time"
)

var (
	PROCESSING_TIME      = "onionProcessingTime"
	LATENCY_BETWEEN_HOPS = "latencyBetweenHops"
	END_TO_END_LATENCY   = "endToEndLatency"
	//ONION_COUNT          = "onionCounter"
	//MSG_SENT             = "messageSentTimestamp"
	//MSG_RECEIVED         = "messageReceivedTimestamp"
	ONION_SIZE  = "onionSize"
	ONIONS_SENT = "onionsSent"

	ONIONS_RECEIVED = "onionsReceived"
	STARTTIME       int64
)

var collectors map[string]prometheus.Collector

func defineCollectors(startTime int64) {
	STARTTIME = startTime
	//timestampBuckets := []float64{0, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000}
	//onionSizeBuckets := []float64{1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000}

	collectors = map[string]prometheus.Collector{
		PROCESSING_TIME: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: PROCESSING_TIME,
				Help: "Processing time (milliseconds) of onions in seconds",
			},
			[]string{"node", "round", "id"},
		),
		LATENCY_BETWEEN_HOPS: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: LATENCY_BETWEEN_HOPS,
				Help: "Network latency (milliseconds) between hops",
			},
			[]string{"from", "to", "round", "id"},
		),
		END_TO_END_LATENCY: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: END_TO_END_LATENCY,
				Help: "Latency (milliseconds) between the time an onion is sent in round 0 to the time it was received in the last round",
			},
			[]string{"sender", "receiver", "checkpoint", "hash"},
		),
		ONION_SIZE: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ONION_SIZE,
				Help: "Size of onions in bytes, labeled by the round number",
			},
			[]string{"round", "id"},
		),
		ONIONS_SENT: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ONIONS_SENT,
				Help: "Timestamp (milliseconds) of when each onion was sent, labeled as checkpoint onion or not",
			},
			[]string{"sender", "intendedReceiver", "checkpoint", "hash"},
		),
		ONIONS_RECEIVED: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ONIONS_RECEIVED,
				Help: "Timestamp (milliseconds) of when each onion was received, labeled as checkpoint onion or not",
			},
			[]string{"sender", "receiver", "checkpoint", "hash"},
		),
		//ONION_SIZE: prometheus.NewHistogram(prometheus.HistogramOpts{
		//	Name:    ONION_SIZE,
		//	Help:    "Size of onions in bytes",
		//	Buckets: prometheus.DefBuckets,
		//}),
		//ONIONS_RECEIVED: prometheus.NewGaugeVec(
		//	prometheus.GaugeOpts{
		//		Name: ONIONS_RECEIVED,
		//		Help: "Unix timestamp (milliseconds) of when each message was received, labeled as checkpoint onion or not",
		//	},
		//	[]string{"hash"}, // Label to distinguish messages by their hash
		//),
		//ONION_COUNT: prometheus.NewCounterVec(
		//	prometheus.CounterOpts{
		//		Name: ONION_COUNT,
		//		Help: "Number of onions received across all rounds",
		//	},
		//	[]string{"round"},
		//),
		//MSG_SENT: prometheus.NewGaugeVec(
		//	prometheus.GaugeOpts{
		//		Name: MSG_SENT,
		//		Help: "Unix timestamp (milliseconds) of when each message was sent, labeled by message hash",
		//	},
		//	[]string{"hash"}, // Label to distinguish messages by their hash
		//),
		//MSG_RECEIVED: prometheus.NewGaugeVec(
		//	prometheus.GaugeOpts{
		//		Name: MSG_RECEIVED,
		//		Help: "Unix timestamp (milliseconds) of when each message was sent, labeled by message hash",
		//	},
		//	[]string{"hash"}, // Label to distinguish messages by their hash
		//),
	}
}

//func Observe(id string, value float64) {
//	if collector, ok := collectors[id].(prometheus.Observer); ok {
//		collector.Observe(value)
//	} else {
//		pl.LogNewError("Failed to find collector with id: " + id)
//	}
//}
//func Inc(id string, labels ...any) {
//	if labels == nil || len(labels) == 0 {
//		if collector, ok := collectors[id].(prometheus.Counter); ok {
//			collector.Inc()
//		} else {
//			pl.LogNewError("Failed to find counter with id: " + id)
//		}
//	}
//	if collector, ok := collectors[id].(*prometheus.CounterVec); ok {
//		collector.WithLabelValues(utils.Map(labels, func(label any) string {
//			return fmt.Sprintf("%v", label)
//		})...).Inc()
//	} else {
//		pl.LogNewError("Failed to find counterVec with id: " + id)
//	}
//}

func getTimestamp(t int64) float64 {
	return float64(t - STARTTIME)
}

func SetProcessingTime(value int64, node string, round int) {
	set(PROCESSING_TIME, float64(value), []string{node, fmt.Sprintf("%d", round), utils.GenerateUniqueHash()}...)
}

func SetLatencyBetweenHops(value int64, from, to string, round int) {
	set(LATENCY_BETWEEN_HOPS, float64(value), []string{from, to, fmt.Sprintf("%d", round), utils.GenerateUniqueHash()}...)
}

func SetEndToEndLatency(value int64, sender, receiver string, isCheckpoint bool, hash string) {
	set(END_TO_END_LATENCY, float64(value), []string{sender, receiver, fmt.Sprintf("%t", isCheckpoint), hash}...)
}

func SetOnionSize(value int64, round int) {
	set(ONION_SIZE, float64(value), []string{fmt.Sprintf("%d", round), utils.GenerateUniqueHash()}...)
}

func SetOnionsSent(timestamp int64, sender, intendedReceiver string, isCheckpoint bool, hash string) {
	set(ONIONS_SENT, getTimestamp(timestamp), []string{sender, intendedReceiver, fmt.Sprintf("%t", isCheckpoint), hash}...)
}

func SetOnionsReceived(timestamp int64, sender, intendedReceiver string, isCheckpoint bool, hash string) {
	set(ONIONS_RECEIVED, getTimestamp(timestamp), []string{sender, intendedReceiver, fmt.Sprintf("%t", isCheckpoint), hash}...)
}

func set(id string, value float64, labels ...string) {
	slog.Info("Setting", "id", id, "value", value)
	if labels == nil || len(labels) == 0 {
		if collector, ok := collectors[id].(prometheus.Gauge); ok {
			collector.Set(value)
		} else {
			observe(id, value, labels...)
		}
	} else {
		if collector, ok := collectors[id].(*prometheus.GaugeVec); ok {
			collector.WithLabelValues(labels...).Set(value)
		} else {
			observe(id, value, labels...)
		}
	}
}

func observe(id string, value float64, labels ...string) {
	if labels == nil || len(labels) == 0 {
		if collector, ok := collectors[id].(prometheus.Histogram); ok {
			collector.Observe(value)
		} else {
			pl.LogNewError("Failed to find collector with id: " + id)
		}
	} else {
		if collector, ok := collectors[id].(*prometheus.HistogramVec); ok {
			collector.WithLabelValues(labels...).Observe(value)
		} else {
			pl.LogNewError("Failed to find collector with id: " + id)
		}
	}
}

func ServeMetrics(startTime int64, prometheusPort int, collectorIds ...string) (shutdown func()) {

	// Register the default process and Go metrics collectors
	//prometheus.MustRegister(cols.NewProcessCollector(cols.ProcessCollectorOpts{}))
	//prometheus.MustRegister(cols.NewGoCollector())

	defineCollectors(startTime)

	// Register the histogram with Prometheus
	for _, id := range collectorIds {
		if collector, ok := collectors[id]; ok {
			prometheus.MustRegister(collector)
		} else {
			pl.LogNewError("Failed to find collector with id: " + id)
		}
	}
	// Create a new ServeMux and register the /visualizer endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", prometheusPort), // Bind to the specified port
		Handler: mux,                                // Use the mux with the /visualizer endpoint
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
