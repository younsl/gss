package main

import (
	"fmt"
	"log"
	"os"

	"github.com/younsl/ghes-schedule-scanner/internal/config"
	"github.com/younsl/ghes-schedule-scanner/pkg/reporter"
	"github.com/younsl/ghes-schedule-scanner/pkg/scanner"
)

func main() {
	cfg, err := initializeConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	scanner, err := initializeScanner(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize scanner: %v", err)
	}

	reporter := initializeReporter()

	result, err := scanner.ScanScheduledWorkflows(cfg.GitHubOrganization)
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	if err := reporter.GenerateReport(result); err != nil {
		log.Fatalf("Report generation failed: %v", err)
	}

	log.Println("Workflow scan completed successfully")
	os.Exit(0)
}

func initializeConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	setLogLevel(cfg.LogLevel)
	return cfg, nil
}

func initializeScanner(cfg *config.Config) (*scanner.Scanner, error) {
	scanner, err := scanner.NewScanner(cfg.GitHubToken, cfg.GitHubBaseURL, cfg.ConcurrentScans)
	if err != nil {
		return nil, fmt.Errorf("failed to create scanner: %w", err)
	}
	log.Printf("GitHub Base URL: %s", cfg.GitHubBaseURL)
	return scanner, nil
}

func initializeReporter() *reporter.Reporter {
	formatter := &reporter.ConsoleFormatter{}
	return reporter.NewReporter(formatter)
}

// setLogLevel 함수는 로그 레벨을 설정합니다.
func setLogLevel(level string) {
	switch level {
	case "DEBUG":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "INFO":
		log.SetFlags(log.LstdFlags)
	default:
		log.SetFlags(0)
	}
}
