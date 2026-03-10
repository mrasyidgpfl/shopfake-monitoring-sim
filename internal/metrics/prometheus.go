package metrics
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsProduced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shopfake_events_produced_total",
			Help: "Total number of events produced to Kafka",
		},
		[]string{"region", "event_type"},
	)

	EventsConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shopfake_events_consumed_total",
			Help: "Total number of events consumed from Kafka",
		},
		[]string{"region", "event_type"},
	)

	EventsFailedToProcess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shopfake_events_failed_total",
			Help: "Total number of events that failed processing",
		},
		[]string{"region", "event_type"},
	)

	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "shopfake_event_processing_duration_seconds",
			Help:    "Time taken to process an event",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"region", "event_type"},
	)

	OrderRevenue = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shopfake_order_revenue_total",
			Help: "Total revenue from successful payments",
		},
		[]string{"region", "currency"},
	)

	ActiveGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "shopfake_active_goroutines",
			Help: "Number of active simulator goroutines",
		},
	)

	KafkaProduceLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "shopfake_kafka_produce_latency_seconds",
			Help:    "Time taken to produce a message to Kafka",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
	)

	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "shopfake_grpc_request_duration_seconds",
			Help:    "Time taken to handle a gRPC request",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shopfake_grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)
)