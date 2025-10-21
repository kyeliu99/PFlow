package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds application configuration values sourced from environment variables.
type Config struct {
	HTTPPort           string
	DatabaseURL        string
	CamundaURL         string
	CamundaProcessKey  string
	MQURL              string
	MQTicketExchange   string
	MQTicketQueue      string
	WorkerLockDuration time.Duration
}

// Load reads environment variables and produces a Config with sane defaults for local development.
func Load() Config {
	cfg := Config{
		HTTPPort:          getEnv("API_HTTP_PORT", ":8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://pflow:pflow@db:5432/pflow?sslmode=disable"),
		CamundaURL:        getEnv("CAMUNDA_URL", "http://camunda:8080/engine-rest"),
		CamundaProcessKey: getEnv("CAMUNDA_PROCESS_KEY", "ticket_approval"),
		MQURL:             getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		MQTicketExchange:  getEnv("RABBITMQ_TICKET_EXCHANGE", "ticket.events"),
		MQTicketQueue:     getEnv("RABBITMQ_TICKET_QUEUE", "ticket.events.queue"),
		WorkerLockDuration: func() time.Duration {
			v := getEnv("WORKER_LOCK_DURATION", "30s")
			d, err := time.ParseDuration(v)
			if err != nil {
				log.Printf("invalid WORKER_LOCK_DURATION %q, defaulting to 30s: %v", v, err)
				return 30 * time.Second
			}
			return d
		}(),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// MustGetInt reads an environment variable and converts it to int with default fallback.
func MustGetInt(key string, fallback int) int {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("failed to parse %s=%q as int: %v", key, val, err)
		return fallback
	}
	return i
}
