package main

import (
	"fmt"
	"log"

	"github.com/younsl/ghes-schedule-scanner/internal/config"
	"github.com/younsl/ghes-schedule-scanner/pkg/canvas"
	"github.com/younsl/ghes-schedule-scanner/pkg/reporter"
	"github.com/younsl/ghes-schedule-scanner/pkg/scanner"
)

type app struct {
	scanner   *scanner.Scanner
	reporter  *reporter.Reporter
	publisher *canvas.CanvasPublisher
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
	log.Println("Workflow scan completed successfully")
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Config values: %+v", cfg)
		return fmt.Errorf("failed to load config: %w", err)
	}

	scanner := initializeScanner(cfg)
	log.Printf("GitHub Base URL: %s", cfg.GitHubBaseURL)

	// Scan workflows
	result, err := scanner.ScanScheduledWorkflows(cfg.GitHubOrganization)
	if err != nil {
		return fmt.Errorf("workflow scan failed: %w", err)
	}

	publisher := initializeCanvasPublisher(cfg)

	// Publish results to Slack Canvas
	if err := publisher.PublishScanResult(result); err != nil {
		return fmt.Errorf("failed to publish to canvas: %w", err)
	}

	return nil
}

func initializeScanner(cfg *config.Config) *scanner.Scanner {
	client := scanner.InitializeGitHubClient(cfg.GitHubToken, cfg.GitHubBaseURL)
	return scanner.NewScanner(client, cfg.ConcurrentScans)
}

func initializeReporter() *reporter.Reporter {
	formatter := &reporter.ConsoleFormatter{}
	return reporter.NewReporter(formatter)
}

func initializeCanvasPublisher(cfg *config.Config) *canvas.CanvasPublisher {
	return canvas.NewCanvasPublisher(
		cfg.SlackBotToken,
		cfg.SlackChannelID,
		cfg.SlackCanvasID,
	)
}

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
