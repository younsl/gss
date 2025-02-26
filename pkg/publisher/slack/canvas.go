package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

type CanvasPublisher struct {
	client        *slack.Client
	channelID     string
	apiToken      string
	baseURL       string
	canvasID      string
	organization  string
	githubBaseURL string
}

func NewCanvasPublisher(token, channelID, canvasID, organization, githubBaseURL string) *CanvasPublisher {
	return &CanvasPublisher{
		client:        slack.New(token),
		channelID:     channelID,
		apiToken:      token,
		baseURL:       "https://slack.com/api",
		canvasID:      canvasID,
		organization:  organization,
		githubBaseURL: githubBaseURL,
	}
}

func (c *CanvasPublisher) PublishScanResult(result *models.ScanResult) error {
	logrus.WithFields(logrus.Fields{
		"workflowCount": len(result.Workflows),
		"canvasID":      c.canvasID,
	}).Info("Starting Canvas publication process")

	if c.apiToken == "" {
		return fmt.Errorf("invalid configuration: missing Slack API token")
	}
	if !strings.HasPrefix(c.apiToken, "xoxb-") {
		return fmt.Errorf("invalid configuration: Slack API token must start with 'xoxb-'")
	}
	if c.channelID == "" {
		return fmt.Errorf("invalid configuration: missing Slack channel ID")
	}
	if c.canvasID == "" {
		return fmt.Errorf("invalid configuration: missing Canvas ID")
	}

	logrus.WithField("canvasID", c.canvasID).Debug("Preparing Canvas blocks")
	blocks := c.createCanvasBlocks(result)
	logrus.WithFields(logrus.Fields{
		"blockCount": len(blocks),
		"canvasID":   c.canvasID,
	}).Debug("Generated Canvas blocks")

	logrus.WithField("canvasID", c.canvasID).Info("Updating Canvas with content")
	if err := c.updateCanvas(blocks); err != nil {
		logrus.WithFields(logrus.Fields{
			"error":    err,
			"canvasID": c.canvasID,
		}).Error("Failed to update canvas")
		return fmt.Errorf("failed to update canvas (ID: %s): %w", c.canvasID, err)
	}

	logrus.WithField("canvasID", c.canvasID).Info("Successfully updated Canvas content")
	return nil
}

func (c *CanvasPublisher) setCanvasAccess(canvasID string) error {
	logrus.WithField("canvasID", canvasID).Debug("Setting Canvas access")
	url := fmt.Sprintf("%s/canvases.access.set", strings.TrimRight(c.baseURL, "/"))

	payload := map[string]interface{}{
		"token":        c.apiToken,
		"canvas_id":    canvasID,
		"access_level": "write",
		"channel_ids":  []string{c.channelID},
	}

	req, err := http.NewRequest("POST", url, jsonBody(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	logrus.WithField("channelID", c.channelID).Info("Sending request to set access for channel with write permission")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	logrus.WithField("response", string(body)).Info("Canvas access set API response")

	var result struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	if !result.Ok {
		return fmt.Errorf("failed to set canvas access: %s", result.Error)
	}

	logrus.WithField("channelID", c.channelID).Info("Successfully set Canvas access for channel")
	return nil
}

// updateCanvas updates the content of a Canvas: https://api.slack.com/methods/canvases.edit
func (c *CanvasPublisher) updateCanvas(blocks []slack.Block) error {
	markdown := convertBlocksToMarkdown(blocks)

	logrus.WithField("markdown", markdown).Debug("Generated Markdown Content for Canvas")

	url := fmt.Sprintf("%s/canvases.edit", strings.TrimRight(c.baseURL, "/"))

	payload := map[string]interface{}{
		"canvas_id": c.canvasID,
		"token":     c.apiToken,
		"changes": []map[string]interface{}{
			{
				"operation": "replace",
				"document_content": map[string]interface{}{
					// You can optionally specify a section_id or omit it to replace the entire canvas.
					"type":     "markdown",
					"markdown": markdown,
				},
			},
		},
	}

	payloadBytes, _ := json.MarshalIndent(payload, "", "  ")
	logrus.WithField("payload", string(payloadBytes)).Debug("Update Canvas Payload")

	req, err := http.NewRequest("POST", url, jsonBody(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	logrus.WithField("responseHeaders", resp.Header).Debug("Response Headers for Canvas")

	var result struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}

	body, _ := io.ReadAll(resp.Body)
	logrus.WithField("response", string(body)).Debug("Canvas update API response")
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	if !result.Ok {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func convertBlocksToMarkdown(blocks []slack.Block) string {
	var markdown strings.Builder
	var summaryContent string
	var workflowContent strings.Builder

	markdown.WriteString("# GHES Scheduled Workflows\n\n")

	// Separate workflow and summary content processing
	for i := 0; i < len(blocks); i++ {
		if section, ok := blocks[i].(*slack.SectionBlock); ok {
			if section.Text != nil {
				text := section.Text.Text
				if strings.Contains(text, "*Scan Summary*") {
					summaryContent = text + "\n\n"
				} else {
					workflowContent.WriteString(text)
					workflowContent.WriteString("\n\n")
				}
			}
		}
	}

	// Last Updated time and summary are added first
	parts := strings.SplitN(markdown.String(), "\n\n", 2)
	markdown.Reset()
	markdown.WriteString(parts[0] + "\n\n") // Title
	if len(parts) > 1 {
		markdown.WriteString(parts[1]) // Last Updated
	}
	markdown.WriteString(summaryContent)           // Summary
	markdown.WriteString(workflowContent.String()) // Workflows

	return markdown.String()
}

func jsonBody(v interface{}) io.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewBuffer(data)
}

func (c *CanvasPublisher) createCanvasBlocks(result *models.ScanResult) []slack.Block {
	// Count workflows with unknown committer
	unknownCommitters := 0
	for _, wf := range result.Workflows {
		if wf.LastCommitter == "Unknown" {
			unknownCommitters++
		}
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", "GHES Scheduled Workflows", false, false)),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn",
				fmt.Sprintf("📊 *Scan Summary*\n"+
					"• Total Repositories: %d\n"+
					"• Scheduled Workflows: %d\n"+
					"• Unknown Committers: %d\n\n"+
					"*Last Updated:* %s by GHES Schedule Scanner",
					result.TotalRepos,
					len(result.Workflows),
					unknownCommitters,
					time.Now().Format(time.RFC3339),
				),
				false, false,
			),
			nil, nil,
		),
		slack.NewDividerBlock(),
	}

	// Add workflow data in bullet list format
	for i, wf := range result.Workflows {
		blocks = append(blocks, c.createWorkflowRow(wf, i+1))
	}

	return blocks
}

func (c *CanvasPublisher) createWorkflowRow(wf models.WorkflowInfo, index int) slack.Block {
	schedules := strings.Join(wf.CronSchedules, ", ")

	statusEmoji := map[string]string{
		"completed": "✅",
		"failed":    "❌",
		"cancelled": "⛔",
		"skipped":   "⏭️",
		"Unknown":   "❓",
	}

	// Last Committer 포맷
	committerStatus := ""
	if wf.LastCommitter == "Unknown" {
		committerStatus = "Unknown"
	} else if wf.IsActiveUser {
		committerStatus = fmt.Sprintf("%s (Active)", wf.LastCommitter)
	} else {
		committerStatus = fmt.Sprintf("%s (Inactive)", wf.LastCommitter)
	}

	// 워크플로우 파일 경로에서 실제 파일명만 추출
	workflowFileName := wf.WorkflowFileName
	if strings.HasPrefix(workflowFileName, ".github/workflows/") {
		workflowFileName = strings.TrimPrefix(workflowFileName, ".github/workflows/")
	}

	// GitHub Enterprise Server URL 생성
	baseURL := strings.TrimSuffix(c.githubBaseURL, "/api/v3")
	workflowURL := fmt.Sprintf("%s/%s/%s/actions/workflows/%s",
		baseURL,
		c.organization,
		wf.RepoName,
		workflowFileName)

	return slack.NewSectionBlock(
		&slack.TextBlockObject{
			Type: "mrkdwn",
			Text: fmt.Sprintf("* **[%d] %s**\n"+
				"  * Workflow: `%s`\n"+
				"  * Workflow URL: %s\n"+
				"  * UTC Schedule: `%s`\n"+
				"  * KST Schedule: `%s`\n"+
				"  * Last Status: %s `%s`\n"+
				"  * Last Committer: `%s`\n",
				index,
				wf.RepoName,
				wf.WorkflowName,
				workflowURL,
				schedules,
				convertToKST(schedules),
				statusEmoji[wf.LastStatus],
				wf.LastStatus,
				committerStatus,
			),
		},
		nil,
		nil,
	)
}

// convertToKST converts UTC cron schedule to KST
func convertToKST(schedule string) string {
	parts := strings.Fields(schedule)
	if len(parts) != 5 || !isValidCronExpression(parts) {
		return schedule
	}

	minute := parts[0]
	hour := atoi(parts[1])
	dom := parts[2]
	month := parts[3]
	dow := parts[4]

	newHour := (hour + 9) % 24
	dayShift := (hour + 9) / 24

	if dayShift > 0 && dow != "*" {
		dowNum := atoi(dow)
		if dowNum >= 0 && dowNum <= 6 {
			newDow := (dowNum + 1) % 7
			dow = fmt.Sprintf("%d", newDow)
		}
	}

	return fmt.Sprintf("%s %d %s %s %s", minute, newHour, dom, month, dow)
}

func isValidCronExpression(parts []string) bool {
	if parts[0] != "*" && (atoi(parts[0]) < 0 || atoi(parts[0]) > 59) {
		return false
	}
	if parts[1] != "*" && (atoi(parts[1]) < 0 || atoi(parts[1]) > 23) {
		return false
	}
	if parts[4] != "*" && (atoi(parts[4]) < 0 || atoi(parts[4]) > 6) {
		return false
	}
	return true
}

func atoi(s string) int {
	if s == "*" {
		return 0
	}
	i := 0
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0
	}
	return i
}

func (c *CanvasPublisher) GetName() string {
	return "slack-canvas"
}
