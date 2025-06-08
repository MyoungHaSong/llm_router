package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	APIKeys        map[string]string
	Port           string
	RequestTimeout time.Duration

	LogLevel string
}

func LoadConfig() *Config {
	return &Config{
		APIKeys:        loadAPIKeys(),
		Port:           getEnv("PORT", "8080"),
		RequestTimeout: loadRequestTimeout(),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}
}
func loadAPIKeys() map[string]string {
	apiKeys := make(map[string]string)
	apiKeys["openai"] = getEnv("OPENAI_API_KEY", "abc")
	apiKeys["anthropic"] = getEnv("ANTHROPIC_API_KEY", "abc")
	return apiKeys
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func loadRequestTimeout() time.Duration {
	timeoutStr := getEnv("REQUEST_TIMEOUT", "30")
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return 30 * time.Second
	}
	return time.Duration(timeout) * time.Second
}
