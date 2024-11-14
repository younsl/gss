package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GitHubToken        string
	GitHubOrganization string
	GitHubBaseURL      string
	LogLevel           string
	RequestTimeout     int
	ConcurrentScans    int
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	var err error

	cfg.GitHubToken, err = getEnv("GITHUB_TOKEN", true)
	if err != nil {
		return nil, err
	}

	cfg.GitHubOrganization, err = getEnv("GITHUB_ORGANIZATION", true)
	if err != nil {
		return nil, err
	}

	baseURL, err := getEnv("GITHUB_BASE_URL", true)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(baseURL, "/api/v3") {
		baseURL = strings.TrimSuffix(baseURL, "/") + "/api/v3"
	}
	cfg.GitHubBaseURL = baseURL

	cfg.LogLevel = getEnvWithDefault("LOG_LEVEL", "INFO")
	cfg.RequestTimeout = getIntEnvWithDefault("REQUEST_TIMEOUT", 30)
	cfg.ConcurrentScans = getIntEnvWithDefault("CONCURRENT_SCANS", 5)

	return cfg, nil
}

func getEnv(key string, required bool) (string, error) {
	value := os.Getenv(key)
	if value == "" && required {
		return "", fmt.Errorf("%s is not set", key)
	}
	return value, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvWithDefault(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil && value > 0 {
			return value
		}
	}
	return defaultValue
}
