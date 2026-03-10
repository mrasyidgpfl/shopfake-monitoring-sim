package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mrasyidgpfl/shopfake/internal/kafka"
	"github.com/mrasyidgpfl/shopfake/internal/metrics"
	"github.com/mrasyidgpfl/shopfake/internal/models"
)

var currencies = map[models.Region]string{
	models.RegionID: "IDR",
	models.RegionSG: "SGD",
	models.RegionMY: "MYR",
	models.RegionTH: "THB",
	models.RegionVN: "VND",
	models.RegionPH: "PHP",
	models.RegionTW: "TWD",
	models.RegionBR: "BRL",
}

var eventWeights = []struct {
	eventType models.EventType
	weight    int
}{
	{models.OrderPlaced, 30},
	{models.OrderCompleted, 25},
	{models.OrderFailed, 5},
	{models.PaymentSuccess, 20},
	{models.PaymentFailed, 3},
	{models.UserSignup, 10},
	{models.VoucherRedeemed, 7},
}

func pickEventType() models.EventType {
	total := 0
	for _, w := range eventWeights {
		total += w.weight
	}
	r := rand.Intn(total)
	cumulative := 0
	for _, w := range eventWeights {
		cumulative += w.weight
		if r < cumulative {
			return w.eventType
		}
	}
	return models.OrderPlaced
}

func generateEvent(region models.Region) *models.Event {
	eventType := pickEventType()
	amount := 0.0

	switch eventType {
	case models.OrderPlaced, models.OrderCompleted, models.PaymentSuccess:
		amount = float64(rand.Intn(50000)+1000) / 100.0
	case models.VoucherRedeemed:
		amount = float64(rand.Intn(2000)+100) / 100.0
	}

	return &models.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Region:    region,
		UserID:    fmt.Sprintf("user_%s_%d", region, rand.Intn(100000)),
		Amount:    amount,
		Currency:  currencies[region],
		Timestamp: time.Now(),
	}
}

func simulateRegion(ctx context.Context, wg *sync.WaitGroup, producer *kafka.Producer, region models.Region) {
	defer wg.Done()
	metrics.ActiveGoroutines.Inc()
	defer metrics.ActiveGoroutines.Dec()

	log.Printf("starting simulator for region %s", region)

	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping simulator for region %s", region)
			return
		default:
			event := generateEvent(region)

			start := time.Now()
			if err := producer.SendEvent(ctx, event); err != nil {
				log.Printf("[%s] failed to send event: %v", region, err)
				time.Sleep(time.Second)
				continue
			}
			metrics.KafkaProduceLatency.Observe(time.Since(start).Seconds())
			metrics.EventsProduced.WithLabelValues(string(region), string(event.Type)).Inc()

			// random delay between 50ms and 500ms per region
			delay := time.Duration(50+rand.Intn(450)) * time.Millisecond
			time.Sleep(delay)
		}
	}
}

func main() {
	brokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	topic := getEnv("KAFKA_TOPIC", "shopfake-events")

	var err error
	var producer *kafka.Producer
	for i := 0; i < 30; i++ {
		producer, err = kafka.NewProducer(brokers, topic)
		if err == nil {
			break
		}
		log.Printf("waiting for kafka... attempt %d/30: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to create producer after retries: %v", err)
	}
	defer producer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// one goroutine per region — concurrent traffic simulation
	for _, region := range models.AllRegions {
		wg.Add(1)
		go simulateRegion(ctx, &wg, producer, region)
	}

	// prometheus metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("simulator metrics server listening on :2112")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("metrics server error: %v", err)
		}
	}()

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down simulator...")
	cancel()
	wg.Wait()
	log.Println("simulator stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
