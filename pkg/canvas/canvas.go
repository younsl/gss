package canvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

type CanvasPublisher struct {
	client    *slack.Client
	channelID string
	apiToken  string
	baseURL   string
	canvasID  string
}

func NewCanvasPublisher(token, channelID, canvasID string) *CanvasPublisher {
	return &CanvasPublisher{
		client:    slack.New(token),
		channelID: channelID,
		apiToken:  token,
		baseURL:   "https://slack.com/api",
		canvasID:  canvasID,
	}
}

func (c *CanvasPublisher) PublishScanResult(result *models.ScanResult) error {
	fmt.Printf("Starting Canvas publication process for %d workflows (Canvas ID: %s)\n", len(result.Workflows), c.canvasID)

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

	fmt.Printf("Preparing Canvas blocks for Canvas ID: %s...\n", c.canvasID)
	blocks := c.createCanvasBlocks(result)
	fmt.Printf("Generated %d blocks for Canvas ID: %s\n", len(blocks), c.canvasID)

	fmt.Printf("Updating Canvas (ID: %s) with content...\n", c.canvasID)
	err := c.updateCanvas(blocks)
	if err != nil {
		return fmt.Errorf("failed to update canvas (ID: %s): %w", c.canvasID, err)
	}
	fmt.Printf("Successfully updated Canvas content (ID: %s)\n", c.canvasID)

	return nil
}

func (c *CanvasPublisher) setCanvasAccess(canvasID string) error {
	fmt.Printf("Setting Canvas access for canvas ID: %s\n", canvasID)
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
	fmt.Printf("Sending request to set access for channel: %s with write permission\n", c.channelID)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("Canvas access set API response: %s\n", string(body))

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

	fmt.Printf("Successfully set Canvas access for channel: %s\n", c.channelID)
	return nil
}

// updateCanvas updates the content of a Canvas: https://api.slack.com/methods/canvases.edit
func (c *CanvasPublisher) updateCanvas(blocks []slack.Block) error {
	markdown := convertBlocksToMarkdown(blocks)

	fmt.Printf("Generated Markdown Content for Canvas ID %s:\n%s\n", c.canvasID, markdown)

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
	fmt.Printf("Update Canvas Payload for Canvas ID %s:\n%s\n", c.canvasID, string(payloadBytes))

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

	fmt.Printf("Response Headers for Canvas ID %s: %v\n", c.canvasID, resp.Header)

	var result struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Canvas update API response for Canvas ID %s: %s\n", c.canvasID, string(body))
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

	return slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn",
			fmt.Sprintf("* **[%d]** **%s**\n"+
				"  * Workflow: `%s`\n"+
				"  * UTC Schedule: `%s`\n"+
				"  * KST Schedule: `%s`\n"+
				"  * Last Status: %s `%s`\n"+
				"  * Last Committer: `%s`\n",
				index,
				wf.RepoName,
				wf.WorkflowName,
				schedules,
				convertToKST(schedules),
				statusEmoji[wf.LastStatus],
				wf.LastStatus,
				wf.LastCommitter,
			),
			false, false,
		),
		nil, nil,
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
