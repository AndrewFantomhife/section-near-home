package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	MaxBotToken    string
	MockAPIURL     string
	ServerPort     string
	MaxAPIBaseURL  string
	LogLevel       string
	LogFormat      string
	StorageKind    string
	MaxBotUsername string
}

func Load() (*Config, error) {
	cfg := &Config{
		MaxBotToken:    os.Getenv("MAX_BOT_TOKEN"),
		MockAPIURL:     getEnvOrDefault("MOCK_API_URL", "http://localhost:3001"),
		ServerPort:     getEnvOrDefault("SERVER_PORT", "8080"),
		MaxAPIBaseURL:  getEnvOrDefault("MAX_API_BASE_URL", "https://platform-api2.max.ru"),
		LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:      getEnvOrDefault("LOG_FORMAT", "text"),
		StorageKind:    getEnvOrDefault("STORAGE_KIND", "memory"),
		MaxBotUsername: getEnvOrDefault("MAX_BOT_USERNAME", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}
func (c *Config) validate() error {
	var missingFields []string

	if strings.TrimSpace(c.MaxBotToken) == "" {
		missingFields = append(missingFields, "MAX_BOT_TOKEN")
	}

	if strings.TrimSpace(c.MockAPIURL) == "" {
		missingFields = append(missingFields, "MOCK_API_URL")
	}

	if strings.TrimSpace(c.ServerPort) == "" {
		missingFields = append(missingFields, "SERVER_PORT")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("required environment variables not set: %s",
			strings.Join(missingFields, ", "))
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
