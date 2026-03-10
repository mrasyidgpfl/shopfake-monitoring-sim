package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	internalgrpc "github.com/mrasyidgpfl/shopfake/internal/grpc"
	"github.com/mrasyidgpfl/shopfake/internal/kafka"
	"github.com/mrasyidgpfl/shopfake/internal/metrics"
	"github.com/mrasyidgpfl/shopfake/internal/models"
	"github.com/mrasyidgpfl/shopfake/internal/store"
	pb "github.com/mrasyidgpfl/shopfake/proto/pb"
)

func main() {
	dsn := getEnv("POSTGRES_DSN", "postgres://shopfake:shopfake@localhost:5432/shopfake?sslmode=disable")
	brokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	topic := getEnv("KAFKA_TOPIC", "shopfake-events")
	grpcPort := getEnv("GRPC_PORT", "50051")

	// postgres
	var err error
	var pg *store.PostgresStore
	for i := 0; i < 30; i++ {
		pg, err = store.NewPostgresStore(dsn)
		if err == nil {
			break
		}
		log.Printf("waiting for postgres... attempt %d/30: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to connect to postgres after retries: %v", err)
	}

	if err := pg.Migrate(); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	log.Println("database migrated")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// kafka consumer
	handler := func(ctx context.Context, event *models.Event) error {
		start := time.Now()

		if err := pg.InsertEvent(ctx, event); err != nil {
			metrics.EventsFailedToProcess.WithLabelValues(string(event.Region), string(event.Type)).Inc()
			return err
		}

		duration := time.Since(start).Seconds()
		metrics.EventsConsumed.WithLabelValues(string(event.Region), string(event.Type)).Inc()
		metrics.EventProcessingDuration.WithLabelValues(string(event.Region), string(event.Type)).Observe(duration)

		if event.Type == models.PaymentSuccess {
			metrics.OrderRevenue.WithLabelValues(string(event.Region), event.Currency).Inc()
		}

		return nil
	}

	var consumer *kafka.Consumer
	for i := 0; i < 30; i++ {
		consumer, err = kafka.NewConsumer(brokers, "shopfake-processor", []string{topic}, handler)
		if err == nil {
			break
		}
		log.Printf("waiting for kafka... attempt %d/30: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to create consumer after retries: %v", err)
	}
	defer consumer.Close()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()
	log.Println("kafka consumer started")

	// grpc server
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	eventServer := internalgrpc.NewEventServer(pg)
	pb.RegisterEventServiceServer(grpcServer, eventServer)

	go func() {
		log.Printf("grpc server listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc server error: %v", err)
		}
	}()

	// prometheus metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("processor metrics server listening on :2113")
		if err := http.ListenAndServe(":2113", nil); err != nil {
			log.Fatalf("metrics server error: %v", err)
		}
	}()

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down processor...")
	cancel()
	grpcServer.GracefulStop()
	log.Println("processor stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
