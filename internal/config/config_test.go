package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"GITHUB_TOKEN":        "test-token",
				"GITHUB_ORGANIZATION": "test-org",
				"GITHUB_BASE_URL":     "https://github.example.com/api/v3",
				"SLACK_BOT_TOKEN":     "xoxb-1-test",
				"SLACK_CHANNEL_ID":    "C12345678",
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			envVars: map[string]string{
				"GITHUB_TOKEN":        "test-token",
				"GITHUB_ORGANIZATION": "test-org",
				"SLACK_BOT_TOKEN":     "xoxb-1-test",
				"SLACK_CHANNEL_ID":    "C12345678",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Backup original env values
			oldEnv := map[string]string{}
			for k := range tt.envVars {
				oldEnv[k] = os.Getenv(k)
			}

			// Set test env values
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Restore original env values after test
			defer func() {
				for k, v := range oldEnv {
					os.Setenv(k, v)
				}
			}()

			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantError %v", err, tt.wantErr)
			}
		})
	}
}

func validateConfig(t *testing.T, cfg *Config, envVars map[string]string) {
	if token := envVars["GITHUB_TOKEN"]; token != "" && cfg.GitHubToken != token {
		t.Errorf("GitHubToken = %v, want %v", cfg.GitHubToken, token)
	}

	if org := envVars["GITHUB_ORGANIZATION"]; org != "" && cfg.GitHubOrganization != org {
		t.Errorf("GitHubOrganization = %v, want %v", cfg.GitHubOrganization, org)
	}

	if baseURL := envVars["GITHUB_BASE_URL"]; baseURL != "" {
		expectedURL := baseURL
		if !strings.HasSuffix(baseURL, "/api/v3") {
			expectedURL = strings.TrimSuffix(baseURL, "/") + "/api/v3"
		}
		if cfg.GitHubBaseURL != expectedURL {
			t.Errorf("GitHubBaseURL = %v, want %v", cfg.GitHubBaseURL, expectedURL)
		}
	}

	if logLevel := envVars["LOG_LEVEL"]; logLevel != "" && cfg.LogLevel != logLevel {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, logLevel)
	}
}

func TestGetIntEnvWithDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		defaultVal int
		want       int
	}{
		{
			name:       "valid value",
			key:        "TEST_INT",
			value:      "42",
			defaultVal: 10,
			want:       42,
		},
		{
			name:       "invalid value",
			key:        "TEST_INT",
			value:      "invalid",
			defaultVal: 10,
			want:       10,
		},
		{
			name:       "empty value",
			key:        "TEST_INT",
			value:      "",
			defaultVal: 10,
			want:       10,
		},
		{
			name:       "negative value",
			key:        "TEST_INT",
			value:      "-1",
			defaultVal: 10,
			want:       10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.key, tt.value)
			defer os.Unsetenv(tt.key)

			got := getIntEnvWithDefault(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getIntEnvWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
