package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// SlackBotToken is a Bot User OAuth Token starting with 'xoxb-'
	// Required OAuth Scopes: channels:manage, chat:write, canvases:write
	SlackBotToken string

	// SlackChannelID is the target Slack channel ID where the canvas will be created
	SlackChannelID string

	// SlackCanvasID is the ID of the existing canvas to update
	SlackCanvasID string

	// GitHubToken is a personal access token for GitHub API authentication
	GitHubToken string

	// GitHubOrganization is the target GitHub organization name
	GitHubOrganization string

	// GitHubBaseURL is the base URL for GitHub Enterprise Server API
	GitHubBaseURL string

	// LogLevel sets the logging level (e.g., "DEBUG", "INFO", "WARN", "ERROR")
	LogLevel string

	// RequestTimeout is the timeout duration in seconds for HTTP requests for GitHub API
	RequestTimeout int

	// ConcurrentScans is the maximum number of concurrent repository scans
	ConcurrentScans int

	// ConnectivityMaxRetries is the maximum number of retry attempts for connectivity check
	ConnectivityMaxRetries int

	// ConnectivityRetryInterval is the duration to wait between retries in seconds for connectivity check
	ConnectivityRetryInterval int

	// ConnectivityTimeout is the timeout for each connection attempt in seconds for connectivity check
	ConnectivityTimeout int
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Required environment variables
	requiredVars := map[string]*string{
		"GITHUB_TOKEN":        &cfg.GitHubToken,
		"GITHUB_ORGANIZATION": &cfg.GitHubOrganization,
		"GITHUB_BASE_URL":     &cfg.GitHubBaseURL,
	}

	for envKey, configVar := range requiredVars {
		value, err := getEnv(envKey, true)
		if err != nil {
			return nil, err
		}
		*configVar = value
	}

	// Optional environment variables
	cfg.SlackBotToken = getEnvWithDefault("SLACK_BOT_TOKEN", "")
	if cfg.SlackBotToken != "" && !strings.HasPrefix(cfg.SlackBotToken, "xoxb-") {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN must start with 'xoxb-'")
	}

	cfg.SlackChannelID = getEnvWithDefault("SLACK_CHANNEL_ID", "")
	cfg.SlackCanvasID = getEnvWithDefault("SLACK_CANVAS_ID", "")

	cfg.LogLevel = getEnvWithDefault("LOG_LEVEL", "INFO")
	cfg.RequestTimeout = getIntEnvWithDefault("REQUEST_TIMEOUT", 30)
	cfg.ConcurrentScans = getIntEnvWithDefault("CONCURRENT_SCANS", 10)

	// Connectivity check configuration
	cfg.ConnectivityMaxRetries = getIntEnvWithDefault("CONNECTIVITY_MAX_RETRIES", 3)
	cfg.ConnectivityRetryInterval = getIntEnvWithDefault("CONNECTIVITY_RETRY_INTERVAL", 5)
	cfg.ConnectivityTimeout = getIntEnvWithDefault("CONNECTIVITY_TIMEOUT", 5)

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
