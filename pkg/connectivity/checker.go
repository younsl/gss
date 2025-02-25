package connectivity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds configuration for the connectivity checker
type Config struct {
	// BaseURL is the GitHub Enterprise Server base URL to check
	BaseURL string

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryInterval is the duration to wait between retries in seconds
	RetryInterval int

	// Timeout is the timeout for each connection attempt in seconds
	Timeout int
}

// ServerInfo holds information about the GitHub Enterprise Server
type ServerInfo struct {
	VerifiablePasswordAuth bool   `json:"verifiable_password_authentication"`
	InstalledVersion       string `json:"installed_version"`
}

// Checker provides functionality to verify network connectivity to GitHub Enterprise Server
type Checker struct {
	config Config
	client *http.Client
}

// NewChecker creates a new connectivity checker with the provided configuration
func NewChecker(config Config) *Checker {
	// Set default values if not provided
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 5
	}

	return &Checker{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// VerifyConnectivity checks if the GitHub Enterprise Server is reachable
// Returns nil if connectivity is successful, otherwise returns an error
func (c *Checker) VerifyConnectivity() error {
	logrus.Info("Starting GitHub Enterprise Server connectivity check")

	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid GitHub Enterprise Server URL: %w", err)
	}

	// Use the meta API endpoint to get server information
	apiURL := fmt.Sprintf("%s://%s/api/v3/meta", baseURL.Scheme, baseURL.Host)

	var serverInfo *ServerInfo

	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		logrus.WithFields(logrus.Fields{
			"attempt": attempt,
			"url":     apiURL,
		}).Debug("Attempting to connect to GitHub Enterprise Server")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.Timeout)*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			logrus.WithError(err).Error("Failed to create request")
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.client.Do(req)
		if err == nil {
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				// Try to parse server information
				body, readErr := io.ReadAll(resp.Body)
				if readErr == nil {
					info := &ServerInfo{}
					if jsonErr := json.Unmarshal(body, info); jsonErr == nil {
						serverInfo = info
						logrus.WithFields(logrus.Fields{
							"installedVersion": info.InstalledVersion,
						}).Info("Successfully connected to GitHub Enterprise Server")
					} else {
						logrus.WithError(jsonErr).Warn("Failed to parse server information")
					}
				} else {
					logrus.WithError(readErr).Warn("Failed to read response body")
				}

				if serverInfo == nil {
					logrus.Info("Successfully connected to GitHub Enterprise Server")
				}
				return nil
			}
			logrus.WithField("statusCode", resp.StatusCode).Warn("Received non-success status code")
		} else {
			logrus.WithError(err).Warn("Connection attempt failed")
		}

		if attempt < c.config.MaxRetries {
			sleepDuration := time.Duration(c.config.RetryInterval) * time.Second
			logrus.WithField("retryIn", sleepDuration.String()).Debug("Retrying connection")
			time.Sleep(sleepDuration)
		}
	}

	return fmt.Errorf("failed to connect to GitHub Enterprise Server after %d attempts", c.config.MaxRetries)
}

// GetServerInfo retrieves information about the GitHub Enterprise Server
// Returns server information if successful, otherwise returns an error
func (c *Checker) GetServerInfo() (*ServerInfo, error) {
	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub Enterprise Server URL: %w", err)
	}

	apiURL := fmt.Sprintf("%s://%s/api/v3/meta", baseURL.Scheme, baseURL.Host)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	info := &ServerInfo{}
	if err := json.Unmarshal(body, info); err != nil {
		return nil, fmt.Errorf("failed to parse server information: %w", err)
	}

	return info, nil
}

// MustVerifyConnectivity checks connectivity and panics if it fails
// This is useful for application initialization where connectivity is required
func (c *Checker) MustVerifyConnectivity() {
	if err := c.VerifyConnectivity(); err != nil {
		logrus.WithError(err).Error("GitHub Enterprise Server connectivity check failed")
		panic("GitHub Enterprise Server is not reachable, application cannot start")
	}

	// Try to get and log server information
	info, err := c.GetServerInfo()
	if err != nil {
		logrus.WithError(err).Warn("Failed to retrieve GitHub Enterprise Server information")
	} else {
		logrus.WithFields(logrus.Fields{
			"installedVersion":       info.InstalledVersion,
			"verifiablePasswordAuth": info.VerifiablePasswordAuth,
		}).Info("GitHub Enterprise Server information")
	}

	logrus.Info("GitHub Enterprise Server connectivity check passed")
}
