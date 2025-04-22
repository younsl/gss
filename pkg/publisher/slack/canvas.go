package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

const (
	slackAPIBaseURL            = "https://slack.com/api"
	canvasAccessSetEndpoint    = "/canvases.access.set"
	canvasEditEndpoint         = "/canvases.edit"
	canvasOperationReplace     = "replace"
	canvasDocumentTypeMarkdown = "markdown"
	canvasAccessLevelWrite     = "write"
	defaultStatusIcon          = ":grey_question:"
	publisherName              = "slack-canvas"

	// Retry constants
	maxRetries                = 3
	initialBackoff            = 1 * time.Second
	backoffFactor             = 2.0
	httpStatusTooManyRequests = 429
)

var workflowStatusIcons = map[string]string{
	"success":     ":white_check_mark:",
	"failure":     ":x:",
	"cancelled":   ":no_entry_sign:",
	"skipped":     ":arrow_right_hook:",
	"timed_out":   ":stopwatch:",
	"in_progress": ":running:",
	"queued":      ":hourglass_flowing_sand:",
	"requested":   ":bell:",
	"waiting":     ":clock3:",
	"pending":     ":pause_button:",
	"completed":   ":white_check_mark:",
}

type CanvasPublisher struct {
	client        *slack.Client
	httpClient    *http.Client
	channelID     string
	apiToken      string
	canvasID      string
	organization  string
	githubBaseURL string
}

func NewCanvasPublisher(token, channelID, canvasID, organization, githubBaseURL string) *CanvasPublisher {
	return &CanvasPublisher{
		client:        slack.New(token),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		channelID:     channelID,
		apiToken:      token,
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
	ctx := context.Background()
	if err := c.updateCanvas(ctx, blocks); err != nil {
		return err
	}

	logrus.WithField("canvasID", c.canvasID).Info("Successfully updated Canvas content")
	return nil
}

func (c *CanvasPublisher) updateCanvas(ctx context.Context, blocks []slack.Block) error {
	markdown := convertBlocksToMarkdown(blocks)
	logrus.WithField("markdown", markdown).Debug("Generated Markdown Content for Canvas")

	url := fmt.Sprintf("%s%s", slackAPIBaseURL, canvasEditEndpoint)

	payload := map[string]interface{}{
		"canvas_id": c.canvasID,
		"token":     c.apiToken,
		"changes": []map[string]interface{}{
			{
				"operation": canvasOperationReplace,
				"document_content": map[string]interface{}{
					"type":     canvasDocumentTypeMarkdown,
					"markdown": markdown,
				},
			},
		},
	}

	var lastErr error
	currentBackoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("canvas update cancelled or timed out (attempt %d): %w", attempt, ctx.Err())
		default:
		}

		if attempt > 0 {
			logrus.Warnf("Retrying canvas update for canvas %s (attempt %d/%d) after error: %v", c.canvasID, attempt, maxRetries, lastErr)
			select {
			case <-time.After(currentBackoff):
			case <-ctx.Done():
				return fmt.Errorf("canvas update retry sleep cancelled or timed out (attempt %d): %w", attempt, ctx.Err())
			}
			currentBackoff = time.Duration(float64(currentBackoff) * backoffFactor)
		}

		bodyReader, err := jsonBody(payload)
		if err != nil {
			lastErr = fmt.Errorf("failed to marshal canvas update payload (non-retryable) for canvas %s: %w", c.canvasID, err)
			logrus.WithError(lastErr).Warn("Error marshaling canvas update payload")
			return lastErr
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
		if err != nil {
			lastErr = fmt.Errorf("failed to create HTTP request for canvas update (non-retryable) (%s): %w", url, err)
			logrus.WithError(lastErr).Warn("Error creating HTTP request for canvas update")
			return lastErr
		}

		req.Header.Set("Authorization", "Bearer "+c.apiToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error on attempt %d updating canvas %s: %w", attempt, c.canvasID, err)
			logrus.WithError(lastErr).Warn("Network error during canvas update")
			continue // Retry on network error
		}

		statusCode := resp.StatusCode
		respBodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close() // Close body immediately after reading

		if readErr != nil {
			lastErr = fmt.Errorf("failed to read response body on attempt %d for canvas update %s: %w", attempt, c.canvasID, readErr)
			logrus.WithError(lastErr).Error("Error reading canvas update response body")
			return lastErr // Non-retryable
		}

		logrus.WithFields(logrus.Fields{
			"statusCode": statusCode,
			"response":   string(respBodyBytes),
			"attempt":    attempt,
			"canvasID":   c.canvasID,
		}).Debug("Canvas update API response received")

		if statusCode == httpStatusTooManyRequests {
			retryAfterStr := resp.Header.Get("Retry-After")
			retryAfterSec, parseErr := strconv.Atoi(retryAfterStr)
			waitDuration := currentBackoff

			if parseErr == nil && retryAfterSec > 0 {
				waitDuration = time.Duration(retryAfterSec) * time.Second
				logrus.Warnf("Rate limited on canvas update (attempt %d). Retrying after %d seconds (from header).", attempt, retryAfterSec)
			} else {
				logrus.Warnf("Rate limited on canvas update (attempt %d). Retrying after %v (calculated backoff).", attempt, waitDuration)
			}
			lastErr = fmt.Errorf("rate limited (status 429) on attempt %d updating canvas %s", attempt, c.canvasID)
			select {
			case <-time.After(waitDuration):
			case <-ctx.Done():
				return fmt.Errorf("canvas update rate limit wait cancelled or timed out (attempt %d): %w", attempt, ctx.Err())
			}
			continue // Retry after waiting
		}

		if statusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d) on attempt %d updating canvas %s: %s", statusCode, attempt, c.canvasID, string(respBodyBytes))
			logrus.WithError(lastErr).Warn("Server error during canvas update")
			continue // Retry on server errors
		}

		if statusCode >= 400 {
			lastErr = fmt.Errorf("client error (status %d) on attempt %d updating canvas %s: %s", statusCode, attempt, c.canvasID, string(respBodyBytes))
			logrus.WithError(lastErr).Error("Client error during canvas update")
			return lastErr // Do not retry client errors
		}

		var result struct {
			Ok    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBodyBytes, &result); err != nil {
			lastErr = fmt.Errorf("failed to decode response body (status %d) on attempt %d for canvas update %s: %w, body: %s", statusCode, attempt, c.canvasID, err, string(respBodyBytes))
			logrus.WithError(lastErr).Error("Error decoding canvas update response")
			return lastErr // Non-retryable decode error
		}

		if !result.Ok {
			lastErr = fmt.Errorf("slack API error (ok=false) updating canvas %s: %s", c.canvasID, result.Error)
			logrus.WithError(lastErr).Error("Slack API error during canvas update")
			return lastErr // Do not retry logical API errors
		}

		logrus.Infof("Successfully updated canvas %s on attempt %d", c.canvasID, attempt)
		return nil // Success
	}

	logrus.Errorf("Canvas update failed after %d attempts for canvas %s. Last error: %v", maxRetries+1, c.canvasID, lastErr)
	return fmt.Errorf("canvas update failed after %d attempts for canvas %s: %w", maxRetries+1, c.canvasID, lastErr)
}

func convertBlocksToMarkdown(blocks []slack.Block) string {
	var markdown strings.Builder
	var summaryContent string
	var workflowContent strings.Builder

	markdown.WriteString("# GHES Scheduled Workflows\n\n")

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

	parts := strings.SplitN(markdown.String(), "\n\n", 2)
	markdown.Reset()
	markdown.WriteString(parts[0] + "\n\n")
	if len(parts) > 1 {
		markdown.WriteString(parts[1])
	}
	markdown.WriteString(summaryContent)
	markdown.WriteString(workflowContent.String())

	return markdown.String()
}

func jsonBody(v interface{}) (io.Reader, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}
	return bytes.NewBuffer(data), nil
}

func (c *CanvasPublisher) createCanvasBlocks(result *models.ScanResult) []slack.Block {
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
					"• Excluded Repositories: %d\n"+
					"• Scheduled Workflows Found: %d\n"+
					"• Unknown Committers: %d\n\n"+
					"*Last Updated:* %s by GHES Schedule Scanner",
					result.TotalRepos,
					result.ExcludedReposCount,
					len(result.Workflows),
					unknownCommitters,
					time.Now().Format("2006-01-02 15:04:05 MST")),
				false,
				false,
			),
			nil, nil,
		),
		slack.NewDividerBlock(),
	}

	for i, wf := range result.Workflows {
		blocks = append(blocks, c.createWorkflowRow(wf, i))
	}

	return blocks
}

func (c *CanvasPublisher) createWorkflowRow(wf models.WorkflowInfo, index int) slack.Block {
	baseURL := strings.TrimSuffix(c.githubBaseURL, "/api/v3")
	workflowFilenameBase := filepath.Base(wf.WorkflowFileName)
	workflowURL := fmt.Sprintf("%s/%s/%s/actions/workflows/%s",
		strings.TrimRight(baseURL, "/"),
		c.organization,
		wf.RepoName,
		workflowFilenameBase,
	)

	workflowLink := fmt.Sprintf("<%s|%s>", workflowURL, wf.WorkflowName)

	schedules := "N/A"
	if len(wf.CronSchedules) > 0 {
		schedules = strings.Join(wf.CronSchedules, ", ")
	}

	kstSchedules := ""
	if schedules != "N/A" {
		var kstParts []string
		for _, schedule := range wf.CronSchedules {
			kstParts = append(kstParts, convertToKST(schedule))
		}
		kstSchedules = strings.Join(kstParts, ", ")
	} else {
		kstSchedules = "N/A"
	}

	statusIcon, ok := workflowStatusIcons[wf.LastStatus]
	if !ok {
		statusIcon = defaultStatusIcon
	}

	committerInfo := wf.LastCommitter
	if !wf.IsActiveUser {
		committerInfo = fmt.Sprintf(":warning: %s (Inactive)", wf.LastCommitter)
	}

	var mdText strings.Builder
	mdText.WriteString(fmt.Sprintf("* *[%d]* %s\n", index+1, wf.RepoName))
	mdText.WriteString(fmt.Sprintf("  * *Workflow:* %s\n", workflowLink))
	mdText.WriteString(fmt.Sprintf("  * *Schedule (UTC):* `%s`\n", schedules))
	mdText.WriteString(fmt.Sprintf("  * *Schedule (KST):* `%s`\n", kstSchedules))
	mdText.WriteString(fmt.Sprintf("  * *Last Status:* %s %s\n", statusIcon, wf.LastStatus))
	mdText.WriteString(fmt.Sprintf("  * *Last Commit By:* %s", committerInfo))

	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, mdText.String(), false, false),
		nil,
		nil,
	)
}

func convertToKST(schedule string) string {
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return schedule
	}
	if !isValidCronExpression(parts) {
		logrus.Warnf("Invalid cron expression format detected: %s", schedule)
		return schedule
	}

	minute := parts[0]
	hourStr := parts[1]
	dom := parts[2]
	month := parts[3]
	originalDow := parts[4]
	currentDow := originalDow

	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return schedule
	}

	newHour := (hour + 9) % 24
	dayShift := (hour + 9) / 24

	if dayShift > 0 && originalDow != "*" {
		dowNum, err := strconv.Atoi(originalDow)
		if err == nil && dowNum >= 0 && dowNum <= 6 {
			newDow := (dowNum + dayShift) % 7
			currentDow = fmt.Sprintf("%d", newDow)
		}
	}

	return fmt.Sprintf("%s %d %s %s %s", minute, newHour, dom, month, currentDow)
}

func isValidCronExpression(parts []string) bool {
	if len(parts) != 5 {
		return false
	}
	validate := func(part string, min, max int) bool {
		if part == "*" {
			return true
		}
		val, err := strconv.Atoi(part)
		if err == nil {
			return val >= min && val <= max
		}
		if strings.ContainsAny(part, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&()_=+{}[]|\\:;\"'<>.?") {
			return false
		}
		return true
	}

	if !validate(parts[0], 0, 59) {
		return false
	}
	if !validate(parts[1], 0, 23) {
		return false
	}
	if !validate(parts[2], 1, 31) {
		return false
	}
	if !validate(parts[3], 1, 12) {
		return false
	}
	if !validate(parts[4], 0, 6) {
		return false
	}

	return true
}

func (c *CanvasPublisher) GetName() string {
	return publisherName
}
