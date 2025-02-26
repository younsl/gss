package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

func TestCanvasPublisher_PublishScanResult(t *testing.T) {
	// Setup mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Log request details for debugging
		t.Logf("Received request: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/canvases.create":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":        true,
				"canvas_id": "test_canvas_123",
				"error":     "",
			})
		case "/canvases.update":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    true,
				"error": "",
			})
		case "/canvases.edit":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    true,
				"error": "",
			})
		default:
			t.Logf("Unhandled path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    false,
				"error": "invalid_path",
			})
		}
	}))
	defer mockServer.Close()

	// Create test publisher with mock server URL
	publisher := &CanvasPublisher{
		client:        slack.New("xoxb-test-token"),
		channelID:     "test-channel",
		apiToken:      "xoxb-test-token",
		baseURL:       mockServer.URL,
		canvasID:      "test-canvas",
		organization:  "test-org",
		githubBaseURL: "https://github.example.com/api/v3",
	}

	result := &models.ScanResult{
		TotalRepos: 2,
		Workflows: []models.WorkflowInfo{
			{
				RepoName:      "test-repo-1",
				WorkflowName:  "workflow1.yml",
				CronSchedules: []string{"0 15 * * *"},
				LastStatus:    "success",
				LastCommitter: "user1",
			},
		},
	}

	err := publisher.PublishScanResult(result)
	if err != nil {
		t.Logf("Error details: %v", err)
	}
	assert.NoError(t, err)
}

func TestConvertToKST(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		want     string
	}{
		{
			name:     "Convert UTC 15:00 to KST",
			schedule: "0 15 * * *",
			want:     "0 0 * * *",
		},
		{
			name:     "Convert UTC 23:00 to KST (next day)",
			schedule: "30 23 * * 1",
			want:     "30 8 * * 2",
		},
		{
			name:     "Invalid cron expression",
			schedule: "invalid cron",
			want:     "invalid cron",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToKST(tt.schedule)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateCanvasBlocks(t *testing.T) {
	publisher := NewCanvasPublisher("test-token", "test-channel", "Test Canvas", "test-org", "https://github.example.com/api/v3")
	result := &models.ScanResult{
		TotalRepos: 1,
		Workflows: []models.WorkflowInfo{
			{
				RepoName:      "test-repo",
				WorkflowName:  "test.yml",
				CronSchedules: []string{"0 15 * * *"},
				LastStatus:    "success",
				LastCommitter: "user1",
			},
		},
	}

	blocks := publisher.createCanvasBlocks(result)

	// Debug: Print block types
	for i, block := range blocks {
		t.Logf("Block %d type: %T", i, block)
	}

	// Verify blocks length first
	assert.True(t, len(blocks) >= 5, "Should have at least 5 blocks")

	// Verify header block
	headerBlock, ok := blocks[0].(*slack.HeaderBlock)
	if assert.True(t, ok, "First block should be HeaderBlock") {
		assert.Equal(t, "GHES Scheduled Workflows", headerBlock.Text.Text)
	}

	// Verify first divider block
	_, ok = blocks[1].(*slack.DividerBlock)
	assert.True(t, ok, "Second block should be DividerBlock")

	// Verify summary section block
	summaryBlock, ok := blocks[2].(*slack.SectionBlock)
	if assert.True(t, ok, "Third block should be SectionBlock") {
		assert.Contains(t, summaryBlock.Text.Text, "Last Updated")
		assert.Contains(t, summaryBlock.Text.Text, "Total Repositories")
	}

	// Verify second divider block
	_, ok = blocks[3].(*slack.DividerBlock)
	assert.True(t, ok, "Fourth block should be DividerBlock")

	// Verify workflow data block
	if assert.True(t, len(blocks) > 4, "Should have more than 4 blocks") {
		workflowBlock, ok := blocks[4].(*slack.SectionBlock)
		if assert.True(t, ok, "Fifth block should be SectionBlock") {
			text := workflowBlock.Text.Text
			assert.Contains(t, text, "test-repo")
			assert.Contains(t, text, "test.yml")
		}
	}
}

func TestNewCanvasPublisherWithMissingEnv(t *testing.T) {
	// Backup original env values
	originalToken := os.Getenv("SLACK_BOT_TOKEN")
	originalChannel := os.Getenv("SLACK_CHANNEL_ID")
	defer func() {
		// Restore original env values after test
		os.Setenv("SLACK_BOT_TOKEN", originalToken)
		os.Setenv("SLACK_CHANNEL_ID", originalChannel)
	}()

	// Clear env variables
	os.Unsetenv("SLACK_BOT_TOKEN")
	os.Unsetenv("SLACK_CHANNEL_ID")

	tests := []struct {
		name        string
		token       string
		channelID   string
		canvasName  string
		expectError bool
	}{
		{
			name:        "Missing both token and channel ID",
			token:       "",
			channelID:   "",
			canvasName:  "Test Canvas",
			expectError: true,
		},
		{
			name:        "Missing token only",
			token:       "",
			channelID:   "C123456",
			canvasName:  "Test Canvas",
			expectError: true,
		},
		{
			name:        "Missing channel ID only",
			token:       "xapp-1-test-token",
			channelID:   "",
			canvasName:  "Test Canvas",
			expectError: true,
		},
		{
			name:        "Invalid token format (not xoxb)",
			token:       "xapp-test-token",
			channelID:   "C123456",
			canvasName:  "Test Canvas",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := NewCanvasPublisher(tt.token, tt.channelID, tt.canvasName, "test-org", "https://github.example.com/api/v3")
			result := &models.ScanResult{
				TotalRepos: 1,
				Workflows: []models.WorkflowInfo{
					{
						RepoName:      "test-repo",
						WorkflowName:  "test.yml",
						CronSchedules: []string{"0 15 * * *"},
					},
				},
			}

			err := publisher.PublishScanResult(result)
			if tt.expectError {
				assert.Error(t, err, "Expected error for missing or invalid credentials")
				assert.Contains(t, err.Error(), "invalid configuration")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
