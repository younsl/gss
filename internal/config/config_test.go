package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		wantError bool
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"GITHUB_TOKEN":        "test-token",
				"GITHUB_ORGANIZATION": "test-org",
				"GITHUB_BASE_URL":     "https://github.example.com",
				"LOG_LEVEL":           "DEBUG",
				"REQUEST_TIMEOUT":     "60",
				"CONCURRENT_SCANS":    "10",
			},
			wantError: false,
		},
		{
			name: "missing required token",
			envVars: map[string]string{
				"GITHUB_ORGANIZATION": "test-org",
				"GITHUB_BASE_URL":     "https://github.example.com",
			},
			wantError: true,
		},
		{
			name: "invalid timeout value",
			envVars: map[string]string{
				"GITHUB_TOKEN":        "test-token",
				"GITHUB_ORGANIZATION": "test-org",
				"GITHUB_BASE_URL":     "https://github.example.com",
				"REQUEST_TIMEOUT":     "invalid",
			},
			wantError: false, // 기본값 사용
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 환경변수 초기화
			os.Clearenv()

			// 테스트용 환경변수 설정
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := LoadConfig()
			if (err != nil) != tt.wantError {
				t.Errorf("LoadConfig() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				validateConfig(t, cfg, tt.envVars)
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
