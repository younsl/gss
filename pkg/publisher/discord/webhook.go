package discord

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
		return fmt.Errorf("invalid configuration: missing Discord webhook URL")
	}

	// Discord 임베드 생성
	embeds := p.createEmbeds(result)

	// 메시지 페이로드 생성
	payload := map[string]interface{}{
		"content": fmt.Sprintf("GitHub Scheduled Workflows Scan: %s", p.organization),
		"embeds":  embeds,
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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *WebhookPublisher) createEmbeds(result *models.ScanResult) []map[string]interface{} {
	// Discord 임베드 형식으로 메시지 구성
	// 여기에 구현...
	return []map[string]interface{}{
		{
			"title": "Scan Summary",
			"color": 3447003, // 파란색
			"fields": []map[string]interface{}{
				{
					"name":   "Total Repositories",
					"value":  fmt.Sprintf("%d", result.TotalRepos),
					"inline": true,
				},
				{
					"name":   "Workflows with Schedules",
					"value":  fmt.Sprintf("%d", len(result.Workflows)),
					"inline": true,
				},
			},
		},
		// 더 많은 임베드 추가...
	}
}

func (p *WebhookPublisher) GetName() string {
	return "discord-webhook"
}
