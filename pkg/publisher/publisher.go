// pkg/publisher/publisher.go
package publisher

import (
	"fmt"

	"github.com/younsl/ghes-schedule-scanner/pkg/models"
	"github.com/younsl/ghes-schedule-scanner/pkg/publisher/console"
	"github.com/younsl/ghes-schedule-scanner/pkg/publisher/discord"
	"github.com/younsl/ghes-schedule-scanner/pkg/publisher/html"
	"github.com/younsl/ghes-schedule-scanner/pkg/publisher/json"
	"github.com/younsl/ghes-schedule-scanner/pkg/publisher/slack"
)

// Publisher 인터페이스는 스캔 결과를 다양한 대상에 게시하는 기능을 정의합니다
type Publisher interface {
	// PublishScanResult는 스캔 결과를 게시하고 오류가 있으면 반환합니다
	PublishScanResult(*models.ScanResult) error

	// GetName은 Publisher의 이름을 반환합니다
	GetName() string
}

// Factory는 설정에 따라 적절한 Publisher를 생성합니다
type Factory struct {
	// 필요한 설정 필드들
}

// NewPublisherFactory는 새로운 Factory 인스턴스를 생성합니다
func NewPublisherFactory() *Factory {
	return &Factory{}
}

// CreatePublisher는 설정에 따라 적절한 Publisher를 생성합니다
func (f *Factory) CreatePublisher(publisherType string, config map[string]string) (Publisher, error) {
	switch publisherType {
	case "slack-canvas":
		return slack.NewCanvasPublisher(
			config["slackBotToken"],
			config["slackChannelID"],
			config["slackCanvasID"],
			config["githubOrganization"],
			config["githubBaseURL"],
		), nil
	case "slack-webhook":
		return slack.NewWebhookPublisher(
			config["slackWebhookURL"],
			config["githubOrganization"],
			config["githubBaseURL"],
		), nil
	case "discord-webhook":
		return discord.NewWebhookPublisher(
			config["discordWebhookURL"],
			config["githubOrganization"],
			config["githubBaseURL"],
		), nil
	case "json":
		return json.NewJSONPublisher(
			config["jsonOutputPath"],
		), nil
	case "html":
		return html.NewHTMLPublisher(
			config["htmlOutputPath"],
			config["htmlTemplatePath"],
			config["githubOrganization"],
			config["githubBaseURL"],
		), nil
	case "console":
		return console.NewConsolePublisher(), nil
	default:
		return nil, fmt.Errorf("unknown publisher type: %s", publisherType)
	}
}
