package main

import (
	"log"

	"carousel/internal/config"
	"carousel/internal/kafka"
	"carousel/internal/services/payroll"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	consumer, err := kafka.NewConsumer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka consumer: %v", err)
	}
	defer consumer.Close()

	handlers := map[string]kafka.EventHandler{
		"payroll.generate": payroll.NewHandler(cfg),
		// Add future modules, e.g., "inventory.update": inventory.NewHandler(cfg)
	}

	log.Println("Starting Carousel Processor...")
	if err := consumer.Run(handlers); err != nil {
		log.Fatalf("Carousel Processor failed: %v", err)
	}
}