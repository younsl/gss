package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/younsl/ghes-schedule-scanner/internal/config"
	"github.com/younsl/ghes-schedule-scanner/pkg/canvas"
	"github.com/younsl/ghes-schedule-scanner/pkg/connectivity"
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
		logrus.Fatal(err)
	}
	logrus.Info("Workflow scan completed successfully")
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"config": fmt.Sprintf("%+v", cfg),
			"error":  err,
		}).Error("Failed to load config")
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logging level from config
	setLogLevel(cfg.LogLevel)

	// Verify connectivity to GitHub Enterprise Server before proceeding
	connectivityChecker := initializeConnectivityChecker(cfg)
	connectivityChecker.MustVerifyConnectivity()

	scanner := initializeScanner(cfg)
	logrus.WithField("baseURL", cfg.GitHubBaseURL).Info("GitHub Base URL configured")

	// Scan workflows
	result, err := scanner.ScanScheduledWorkflows(cfg.GitHubOrganization)
	if err != nil {
		logrus.WithError(err).Error("Workflow scan failed")
		return fmt.Errorf("workflow scan failed: %w", err)
	}

	publisher := initializeCanvasPublisher(cfg)

	// Publish results to Slack Canvas
	if err := publisher.PublishScanResult(result); err != nil {
		logrus.WithError(err).Error("Failed to publish to canvas")
		return fmt.Errorf("failed to publish to canvas: %w", err)
	}

	return nil
}

func initializeConnectivityChecker(cfg *config.Config) *connectivity.Checker {
	connectivityConfig := connectivity.Config{
		BaseURL:       cfg.GitHubBaseURL,
		MaxRetries:    cfg.ConnectivityMaxRetries,
		RetryInterval: cfg.ConnectivityRetryInterval,
		Timeout:       cfg.ConnectivityTimeout,
	}
	return connectivity.NewChecker(connectivityConfig)
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
		cfg.GitHubOrganization,
		cfg.GitHubBaseURL,
	)
}

func setLogLevel(level string) {
	switch level {
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Set JSON formatter
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
	})
}
