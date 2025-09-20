package config

import (
	"os"
)

// Application settings
type Config struct {
	Server  ServerConfig
	Logging LoggingConfig
}

// Server settings
type ServerConfig struct {
	Port string
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
