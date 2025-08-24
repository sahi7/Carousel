package kafka

import (
    "context"
    "log"
    "sync"
    "time"

    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
    "carousel/internal/config"
    "encoding/json"
)

// EventHandler defines the interface for handling Kafka events
type EventHandler interface {
    Handle(ctx context.Context, msg *kafka.Message) error
}

// Consumer manages Kafka consumption
type Consumer struct {
    config   config.Config
    consumer *kafka.Consumer
    producer *kafka.Producer
}

// NewConsumer initializes a Kafka consumer
func NewConsumer(cfg config.Config) (*Consumer, error) {
    consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers":  cfg.KafkaBootstrapServers,
        "group.id":           "carousel",
        "auto.offset.reset":  "earliest",
        // "auto.offset.reset":  "latest",
        "enable.auto.commit": false,
    })
    if err != nil {
        return nil, err
    }

    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": cfg.KafkaBootstrapServers,
    })
    if err != nil {
        return nil, err
    }

    return &Consumer{
        config:   cfg,
        consumer: consumer,
        producer: producer,
    }, nil
}

// Close cleans up resources
func (c *Consumer) Close() {
    c.consumer.Close()
    c.producer.Close()
}

// Run starts the consumer and dispatches events to handlers
func (c *Consumer) Run(handlers map[string]EventHandler) error {
    if err := c.consumer.SubscribeTopics([]string{c.config.KafkaTopic}, nil); err != nil {
        return err
    }

    eventChan := make(chan *kafka.Message, 1000)
    var wg sync.WaitGroup

    for i := 0; i < c.config.MaxWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            c.processWorker(eventChan, handlers)
        }()
    }

    for {
        msg, err := c.consumer.ReadMessage(10 * time.Millisecond)
        if err == nil {
            eventChan <- msg
        } else if err.(kafka.Error).Code() != kafka.ErrTimedOut {
            log.Printf("Consumer error: %v", err)
        }
    }
}

func (c *Consumer) processWorker(eventChan <-chan *kafka.Message, handlers map[string]EventHandler) {
    for msg := range eventChan {
        var event struct {
            Type     string `json:"type"`
            PeriodID int64  `json:"period_id"`
            Month    int    `json:"month"`
            Year     int    `json:"year"`
        }
        log.Printf("-event: %s", event)
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            log.Printf("Failed to parse event: %v", err)
            c.sendToDLQ(msg)
            continue
        }

        handler, exists := handlers[event.Type]
        if !exists {
            log.Printf("No handler for event type: %s", event.Type)
            c.sendToDLQ(msg)
            continue
        }

        for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
            err := handler.Handle(context.Background(), msg)
            if err == nil {
                c.consumer.CommitMessage(msg)
                break
            }
            log.Printf("Attempt %d failed for event %s: %v", attempt, event.Type, err)
            if attempt == c.config.RetryAttempts {
                c.sendToDLQ(msg)
            }
            time.Sleep(c.config.RetryDelay)
        }
    }
}

func (c *Consumer) CommitOffsets(partitions []kafka.TopicPartition) error {
    _, err := c.consumer.CommitOffsets(partitions)
    return err
}

func (c *Consumer) sendToDLQ(msg *kafka.Message) {
    c.producer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &c.config.KafkaDLQTopic, Partition: kafka.PartitionAny},
        Key:            msg.Key,
        Value:          msg.Value,
    }, nil)
    c.producer.Flush(1000)
    log.Printf("Sent message to DLQ: %s", string(msg.Key))
}