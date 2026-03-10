package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/mrasyidgpfl/shopfake/internal/models"
)

type EventHandler func(ctx context.Context, event *models.Event) error

type Consumer struct {
	group   sarama.ConsumerGroup
	topics  []string
	handler EventHandler
}

type consumerGroupHandler struct {
	handler EventHandler
}

func NewConsumer(brokers []string, groupID string, topics []string, handler EventHandler) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	group, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &Consumer{
		group:   group,
		topics:  topics,
		handler: handler,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	cgh := &consumerGroupHandler{handler: c.handler}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.group.Consume(ctx, c.topics, cgh); err != nil {
				return fmt.Errorf("consumer error: %w", err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.group.Close()
}

// sarama.ConsumerGroupHandler interface implementation

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var event models.Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("failed to unmarshal event: %v", err)
			session.MarkMessage(msg, "")
			continue
		}

		if err := h.handler(session.Context(), &event); err != nil {
			log.Printf("failed to handle event %s: %v", event.ID, err)
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
