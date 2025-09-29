package main

import (
	"context"
	"log"
	"sync"

	"carousel/internal/config"
	"carousel/internal/db"
	"carousel/internal/kafka"
	"carousel/internal/services"
	"carousel/internal/services/payroll"
)

func main() {
	ctx := context.Background()
    var wg sync.WaitGroup

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Postgres
    postgres, err := db.NewPostgres(cfg.PostgresDSN, cfg.RedisAddr)
    if err != nil {
        log.Fatalf("Failed to initialize Postgres: %v", err)
    }
    defer postgres.Pool.Close()

	consumer, err := kafka.NewConsumer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka consumer: %v", err)
	}
	defer consumer.Close()

	// handlers := map[string]kafka.EventHandler{
	// 	"payroll.generate": payroll.NewHandler(cfg),
	// 	// Add future modules, e.g., "inventory.update": inventory.NewHandler(cfg)
	// }

	// log.Println("Starting Kafka Consumer...")
	// if err := consumer.Run(handlers); err != nil {
	// 	log.Fatalf("Carousel Processor failed: %v", err)
	// }

	kafkaHandlers := map[string]kafka.EventHandler{
        "payroll.generate": payroll.NewHandler(cfg, postgres),
    }

	wg.Add(1)
    go func() {
        defer wg.Done()
        log.Println("Starting Kafka Consumer...")
        if err := consumer.Run(kafkaHandlers); err != nil {
            log.Printf("Kafka Consumer failed: %v", err)
        }
    }()

	// Start Redis Streams processor
    processor := services.NewSubscriptionProcessor(postgres)
    wg.Add(1)
    go func() {
        defer wg.Done()
        log.Println("Starting Subscription Processor...")
        if err := processor.Start(ctx); err != nil {
            log.Printf("Subscription Processor failed: %v", err)
        }
    }()

    wg.Wait()
}