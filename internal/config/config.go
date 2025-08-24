package config

import (
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Config holds service configuration
type Config struct {
	KafkaBootstrapServers string
	KafkaTopicGenerated   string
	KafkaTopic            string
	KafkaDLQTopic         string
	PostgresDSN           string
	RedisAddr             string
	MaxWorkers            int
	RetryAttempts         int
	RetryDelay            time.Duration
}

// Load reads configuration from environment variables
func Load() (Config, error) {
	kafkaServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if kafkaServers == "" {
		return Config{}, errors.New("KAFKA_BOOTSTRAP_SERVERS is required")
	}
	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		postgresDSN = "postgres://postgres:secret@localhost:5432/postgres"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	maxWorkers, _ := strconv.Atoi(os.Getenv("MAX_WORKERS"))
	if maxWorkers == 0 {
		maxWorkers = 1000
	}
	retryAttempts, _ := strconv.Atoi(os.Getenv("RETRY_ATTEMPTS"))
	if retryAttempts == 0 {
		retryAttempts = 3
	}
	retryDelay, _ := time.ParseDuration(os.Getenv("RETRY_DELAY"))
	if retryDelay == 0 {
		retryDelay = 2 * time.Second
	}

	return Config{
		KafkaBootstrapServers: kafkaServers,
		KafkaTopic:            "rms.events",
		KafkaDLQTopic:         "rms.events.dlq",
		PostgresDSN:           postgresDSN,
		RedisAddr:             redisAddr,
		MaxWorkers:            maxWorkers,
		RetryAttempts:         retryAttempts,
		RetryDelay:            retryDelay,
		KafkaTopicGenerated:   "payroll.generated",
	}, nil
}