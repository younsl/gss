package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/younsl/ghes-schedule-scanner/internal/config"
)

func TestRun(t *testing.T) {
	// 환경 변수 설정
	envVars := map[string]string{
		"GITHUB_TOKEN":        "test-token",
		"GITHUB_BASE_URL":     "https://api.github.com",
		"GITHUB_ORGANIZATION": "test-org",
		"SLACK_BOT_TOKEN":     "test-slack-token",
		"SLACK_CHANNEL_ID":    "test-channel",
		"SLACK_CANVAS_ID":     "test-canvas",
		"CONCURRENT_SCANS":    "2",
	}

	// 환경 변수 설정
	for k, v := range envVars {
		os.Setenv(k, v)
	}

	// 테스트 후 환경 변수 정리
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	err := run()
	assert.Error(t, err) // GitHub API 호출이 실제로 이뤄지지 않으므로 에러 발생 예상
}

func TestInitializeScanner(t *testing.T) {
	cfg := &config.Config{
		GitHubToken:     "test-token",
		GitHubBaseURL:   "https://api.github.com",
		ConcurrentScans: 2,
	}

	s := initializeScanner(cfg)
	assert.NotNil(t, s)
}

func TestInitializeReporter(t *testing.T) {
	r := initializeReporter()
	assert.NotNil(t, r)
}

func TestInitializeCanvasPublisher(t *testing.T) {
	cfg := &config.Config{
		SlackBotToken:  "test-token",
		SlackChannelID: "test-channel",
		SlackCanvasID:  "test-canvas",
	}

	p := initializeCanvasPublisher(cfg)
	assert.NotNil(t, p)
}

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"Debug Level", "DEBUG"},
		{"Info Level", "INFO"},
		{"Default Level", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setLogLevel(tt.level)
			// 로그 설정이 변경되었는지 직접적으로 테스트하기는 어려우므로
			// 함수 호출이 패닉을 일으키지 않는지만 확인
		})
	}
}
