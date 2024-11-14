package main

import (
	"testing"

	"github.com/younsl/ghes-schedule-scanner/internal/config"
	"github.com/younsl/ghes-schedule-scanner/pkg/reporter"
	"github.com/younsl/ghes-schedule-scanner/pkg/scanner"
)

// 원본 함수들의 타입을 저장할 변수 선언
var (
	originalInitScanner  func(*config.Config) *scanner.Scanner
	originalInitReporter func() *reporter.Reporter
)

// 테스트용 mock 함수들
var (
	mockScanner  *scanner.Scanner
	mockReporter *reporter.Reporter
)

func mockInitializeScanner(_ *config.Config) *scanner.Scanner {
	return mockScanner
}

func mockInitializeReporter() *reporter.Reporter {
	return mockReporter
}

func TestMain(t *testing.T) {
	// 원래 함수들 백업
	originalScanner := originalInitScanner
	originalReporter := originalInitReporter

	// 테스트 후 복구
	defer func() {
		originalInitScanner = originalScanner
		originalInitReporter = originalReporter
	}()

	// mock 객체로 교체
	originalInitScanner = mockInitializeScanner
	originalInitReporter = mockInitializeReporter

	// TODO: main() 함수 테스트 로직 추가
}

func TestInitializeConfig(t *testing.T) {
	// 필요한 모든 환경변수 설정
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_ORGANIZATION", "test-org")
	t.Setenv("GITHUB_BASE_URL", "https://api.github.com")
	t.Setenv("CONCURRENT_SCANS", "5") // 기본값 설정
	t.Setenv("LOG_LEVEL", "INFO")     // 기본값 설정

	cfg, err := initializeConfig()
	if err != nil {
		t.Errorf("initializeConfig() failed: %v", err)
		return
	}
	if cfg == nil {
		t.Error("initializeConfig() returned nil config")
		return
	}

	// 설정값 검증 추가
	if cfg.GitHubToken != "test-token" {
		t.Errorf("expected token 'test-token', got %s", cfg.GitHubToken)
	}
	if cfg.GitHubOrganization != "test-org" {
		t.Errorf("expected org 'test-org', got %s", cfg.GitHubOrganization)
	}
}

func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "DEBUG"},
		{"info level", "INFO"},
		{"default level", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setLogLevel(tt.level)
			// 로그 설정이 에러 없이 적용되는지만 확인
		})
	}
}
