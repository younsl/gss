package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/younsl/ghes-schedule-scanner/internal/config"
)

func TestRun(t *testing.T) {
	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_BASE_URL", "https://api.github.com")
	os.Setenv("GITHUB_ORGANIZATION", "test-org")
	os.Setenv("SLACK_BOT_TOKEN", "test-slack-token")
	os.Setenv("SLACK_CHANNEL_ID", "test-channel")
	os.Setenv("SLACK_CANVAS_ID", "test-canvas")
	os.Setenv("CONCURRENT_SCANS", "2")

	defer func() {
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("GITHUB_BASE_URL")
		os.Unsetenv("GITHUB_ORGANIZATION")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL_ID")
		os.Unsetenv("SLACK_CANVAS_ID")
		os.Unsetenv("CONCURRENT_SCANS")
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

	scanner := initializeScanner(cfg)
	assert.NotNil(t, scanner)
}

func TestInitializeReporter(t *testing.T) {
	reporter := initializeReporter()
	assert.NotNil(t, reporter)
}

func TestInitializeCanvasPublisher(t *testing.T) {
	cfg := &config.Config{
		SlackBotToken:  "test-token",
		SlackChannelID: "test-channel",
		SlackCanvasID:  "test-canvas",
	}

	publisher := initializeCanvasPublisher(cfg)
	assert.NotNil(t, publisher)
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
