package config

import (
	"os"
	"strconv"
	"time"
)

// Application settings
type Config struct {
	Server   ServerConfig
	Logging  LoggingConfig
	ETL      ETLConfig
	External ExternalConfig
}

// Server settings
type ServerConfig struct {
	Port string
}

type ETLConfig struct {
	WorkerPoolSize     int
	BatchSize          int
	RequestTimeout     time.Duration
	MaxRetries         int
	RetryBackoff       time.Duration
	RateLimitPerSecond int
}

type ExternalConfig struct {
	AdsAPIURL  string
	CRMAPIURL  string
	SinkURL    string
	SinkSecret string
}

// Logging settings
type LoggingConfig struct {
	Level string
}

func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		ETL: ETLConfig{
			WorkerPoolSize:     getIntEnv("WORKER_POOL_SIZE", 10),
			BatchSize:          getIntEnv("BATCH_SIZE", 100),
			RequestTimeout:     getDurationEnv("REQUEST_TIMEOUT", "30s"),
			MaxRetries:         getIntEnv("MAX_RETRIES", 3),
			RetryBackoff:       getDurationEnv("RETRY_BACKOFF", "2s"),
			RateLimitPerSecond: getIntEnv("RATE_LIMIT_PER_SECOND", 100),
		},
		External: ExternalConfig{
			AdsAPIURL:  getEnv("ADS_API_URL", ""),
			CRMAPIURL:  getEnv("CRM_API_URL", ""),
			SinkURL:    getEnv("SINK_URL", ""),
			SinkSecret: getEnv("SINK_SECRET", ""),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}
