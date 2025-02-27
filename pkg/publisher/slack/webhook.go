package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

type WebhookPublisher struct {
	webhookURL    string
	organization  string
	githubBaseURL string
	httpClient    *http.Client
}

func NewWebhookPublisher(webhookURL, organization, githubBaseURL string) *WebhookPublisher {
	return &WebhookPublisher{
		webhookURL:    webhookURL,
		organization:  organization,
		githubBaseURL: githubBaseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *WebhookPublisher) PublishScanResult(result *models.ScanResult) error {
	if p.webhookURL == "" {
		return fmt.Errorf("invalid configuration: missing Slack webhook URL")
	}

	// Slack 메시지 블록 생성
	blocks := p.createMessageBlocks(result)

	// 메시지 페이로드 생성
	payload := map[string]interface{}{
		"blocks": blocks,
	}

	// JSON으로 변환
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Webhook으로 전송
	resp, err := p.httpClient.Post(p.webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *WebhookPublisher) createMessageBlocks(result *models.ScanResult) []map[string]interface{} {
	// Slack 블록 형식으로 메시지 구성
	// 여기에 구현...
	return []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": fmt.Sprintf("GitHub Scheduled Workflows Scan: %s", p.organization),
			},
		},
		// 더 많은 블록 추가...
	}
}

func (p *WebhookPublisher) GetName() string {
	return "slack-webhook"
}
