package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/younsl/ghes-schedule-scanner/internal/config"
	"github.com/younsl/ghes-schedule-scanner/pkg/canvas"
	"github.com/younsl/ghes-schedule-scanner/pkg/connectivity"
	"github.com/younsl/ghes-schedule-scanner/pkg/reporter"
	"github.com/younsl/ghes-schedule-scanner/pkg/scanner"
)

type Config struct {
	LogLevel           string
	GitHubToken        string
	GitHubBaseURL      string
	GitHubOrganization string
	ConcurrentScans    int
	PublisherType      string
	SlackToken         string
	SlackChannelID     string
	SlackCanvasID      string
}

type app struct {
	scanner   *scanner.Scanner
	reporter  *reporter.Reporter
	publisher publisher.Publisher
}

func main() {
	if err := run(); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Workflow scan completed successfully")
}

func run() error {
	// 내부 config 패키지를 사용하도록 변경
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.WithError(err).Error("Failed to load configuration")
		return fmt.Errorf("failed to load configuration: %w", err)
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

	// 팩토리 패턴을 사용하여 Publisher 생성
	pub, err := initializePublisher(cfg.PublisherType, cfg)
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize publisher")
		return fmt.Errorf("failed to initialize publisher: %w", err)
	}

	// 결과 게시
	if err := pub.PublishScanResult(result); err != nil {
		logrus.WithError(err).Error("Failed to publish results")
		return fmt.Errorf("failed to publish results: %w", err)
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

func initializePublisher(publisherType string, cfg *config.Config) (publisher.Publisher, error) {
	factory := publisher.NewPublisherFactory()

	// 설정 맵 생성
	config := map[string]string{
		"slackBotToken":      cfg.SlackBotToken,
		"slackChannelID":     cfg.SlackChannelID,
		"slackCanvasID":      cfg.SlackCanvasID,
		"githubOrganization": cfg.GitHubOrganization,
		"githubBaseURL":      cfg.GitHubBaseURL,
	}

	pub, err := factory.CreatePublisher(publisherType, config)
	if err != nil {
		return nil, err
	}

	logrus.WithField("publisherType", pub.GetName()).Info("Publisher initialized")
	return pub, nil
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
