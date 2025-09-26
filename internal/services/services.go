package services

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// EventHandler defines the interface for handling Kafka events
type EventHandler interface {
	Handle(ctx context.Context, msg *kafka.Message) error
}