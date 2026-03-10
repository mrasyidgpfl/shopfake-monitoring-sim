package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	internalgrpc "github.com/mrasyidgpfl/shopfake/internal/grpc"
	"github.com/mrasyidgpfl/shopfake/internal/metrics"
)

type Gateway struct {
	client *internalgrpc.EventClient
}

func (g *Gateway) handleRegionStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		metrics.GRPCRequestDuration.WithLabelValues("gateway_region_stats").Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats, err := g.client.GetRegionStats(ctx)
	if err != nil {
		metrics.GRPCRequestsTotal.WithLabelValues("gateway_region_stats", "error").Inc()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metrics.GRPCRequestsTotal.WithLabelValues("gateway_region_stats", "ok").Inc()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (g *Gateway) handleEventCounts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		metrics.GRPCRequestDuration.WithLabelValues("gateway_event_counts").Observe(time.Since(start).Seconds())
	}()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	counts, err := g.client.GetEventCounts(ctx)
	if err != nil {
		metrics.GRPCRequestsTotal.WithLabelValues("gateway_event_counts", "error").Inc()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metrics.GRPCRequestsTotal.WithLabelValues("gateway_event_counts", "ok").Inc()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(counts)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	grpcAddr := getEnv("GRPC_ADDR", "localhost:50051")
	httpPort := getEnv("HTTP_PORT", "8080")

	var err error
	var client *internalgrpc.EventClient
	for i := 0; i < 30; i++ {
		client, err = internalgrpc.NewEventClient(grpcAddr)
		if err == nil {
			break
		}
		log.Printf("waiting for processor... attempt %d/30: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to connect to processor after retries: %v", err)
	}
	defer client.Close()

	gw := &Gateway{client: client}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats/regions", gw.handleRegionStats)
	mux.HandleFunc("/api/stats/events", gw.handleEventCounts)
	mux.HandleFunc("/health", gw.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("gateway listening on :%s", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("gateway error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("gateway stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
