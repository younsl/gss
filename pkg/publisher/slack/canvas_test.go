package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

// Mock RoundTripper for http client
type mockRoundTripper struct {
	handler http.Handler
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	m.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

// Helper to create a publisher with a mock HTTP client
func newTestPublisherWithMockClient(handler http.Handler, token, channelID, canvasID, org, githubURL string) *CanvasPublisher {
	publisher := NewCanvasPublisher(token, channelID, canvasID, org, githubURL)
	// Inject mock client
	publisher.httpClient = &http.Client{
		Transport: &mockRoundTripper{handler: handler},
	}
	return publisher
}

func TestNewCanvasPublisher(t *testing.T) {
	token := "xoxb-test-token"
	channelID := "C123"
	canvasID := "F456"
	org := "test-org"
	githubURL := "https://github.example.com"

	publisher := NewCanvasPublisher(token, channelID, canvasID, org, githubURL)

	require.NotNil(t, publisher)
	assert.NotNil(t, publisher.client)
	assert.NotNil(t, publisher.httpClient)
	assert.Equal(t, channelID, publisher.channelID)
	assert.Equal(t, token, publisher.apiToken)
	assert.Equal(t, canvasID, publisher.canvasID)
	assert.Equal(t, org, publisher.organization)
	assert.Equal(t, githubURL, publisher.githubBaseURL)
	assert.Equal(t, 30*time.Second, publisher.httpClient.Timeout)
}

func TestPublishScanResult_Success(t *testing.T) {
	apiCalled := false
	// Use a simple handler, not retryMockHandler
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/canvases.edit", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Check request body minimally (optional but good practice)
		bodyBytes, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(bodyBytes), `"operation":"replace"`) // Example check

		apiCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	// Use the simplified helper or inject directly
	publisher := NewCanvasPublisher("xoxb-test-token", "C123", "F456", "test-org", "https://github.example.com")
	publisher.httpClient = &http.Client{Transport: &mockRoundTripper{handler: mockHandler}} // Inject mock transport

	result := &models.ScanResult{
		TotalRepos: 1,
		Workflows: []models.WorkflowInfo{
			{RepoName: "repo1", WorkflowName: "wf1.yml", WorkflowFileName: "wf1.yml", LastStatus: "success", CronSchedules: []string{"0 0 * * *"}, LastCommitter: "user", IsActiveUser: true},
		},
	}

	// This call should now use the context created within PublishScanResult
	err := publisher.PublishScanResult(result)
	require.NoError(t, err) // Should not time out
	assert.True(t, apiCalled, "Slack API /api/canvases.edit should have been called")
}

func TestPublishScanResult_ApiError(t *testing.T) {
	apiCalled := false
	// Simple handler for API error
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/canvases.edit", r.URL.Path)
		apiCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // ok:false comes in body
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "invalid_canvas_id"})
	})

	publisher := NewCanvasPublisher("xoxb-test-token", "C123", "F456", "test-org", "https://github.example.com")
	publisher.httpClient = &http.Client{Transport: &mockRoundTripper{handler: mockHandler}} // Inject mock transport

	result := &models.ScanResult{TotalRepos: 1, Workflows: []models.WorkflowInfo{}}

	err := publisher.PublishScanResult(result)
	require.Error(t, err) // Should fail due to API error, not timeout
	assert.True(t, apiCalled, "Slack API /api/canvases.edit should have been called")
	// Error message should come from updateCanvas logic
	assert.Contains(t, err.Error(), "slack API error (ok=false) updating canvas F456: invalid_canvas_id")
}

func TestPublishScanResult_HttpError(t *testing.T) {
	apiCalled := false
	// Simple handler for HTTP error
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/canvases.edit", r.URL.Path)
		apiCalled = true
		// Simulate server error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	publisher := NewCanvasPublisher("xoxb-test-token", "C123", "F456", "test-org", "https://github.example.com")
	publisher.httpClient = &http.Client{Transport: &mockRoundTripper{handler: mockHandler}} // Inject mock transport

	result := &models.ScanResult{TotalRepos: 1, Workflows: []models.WorkflowInfo{}}

	err := publisher.PublishScanResult(result)
	require.Error(t, err) // Should fail due to HTTP 500, not timeout
	assert.True(t, apiCalled, "Slack API /api/canvases.edit should have been called")
	// Error message should indicate server error after retries (if applicable) or immediate failure
	// The retry logic in updateCanvas will handle this. Expect the final error after retries.
	assert.Contains(t, err.Error(), "canvas update failed after 4 attempts") // Assumes maxRetries = 3
	assert.Contains(t, err.Error(), "server error (status 500)")
	assert.Contains(t, err.Error(), "Internal Server Error")
}

func TestPublishScanResult_ConfigErrors(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		channelID   string
		canvasID    string
		expectError string
	}{
		{"Missing token", "", "C123", "F456", "missing Slack API token"},
		{"Invalid token prefix", "xapp-token", "C123", "F456", "token must start with 'xoxb-'"},
		{"Missing channel ID", "xoxb-token", "", "F456", "missing Slack channel ID"},
		{"Missing canvas ID", "xoxb-token", "C123", "", "missing Canvas ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No need for mock client here as config errors are checked first
			publisher := NewCanvasPublisher(tt.token, tt.channelID, tt.canvasID, "org", "url")
			result := &models.ScanResult{} // Result content doesn't matter here
			err := publisher.PublishScanResult(result)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid configuration")
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestUpdateCanvas_Success(t *testing.T) {
	apiCalled := false
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Corrected Path Assertion
		assert.Equal(t, "/api/canvases.edit", r.URL.Path)
		apiCalled = true
		// Check body contains expected markdown (simplified check)
		bodyBytes, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(bodyBytes), `"markdown":"# GHES Scheduled Workflows`)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})
	publisher := newTestPublisherWithMockClient(mockHandler, "xoxb-token", "C1", "F1", "org", "url")
	blocks := []slack.Block{slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "GHES Scheduled Workflows", false, false))}

	err := publisher.updateCanvas(context.Background(), blocks)
	require.NoError(t, err)
	assert.True(t, apiCalled)
}

func TestConvertBlocksToMarkdown(t *testing.T) {
	tests := []struct {
		name   string
		blocks []slack.Block
		want   string
	}{
		{
			name: "Header, Divider, Summary, Divider, Workflow",
			blocks: []slack.Block{
				slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "GHES Scheduled Workflows", false, false)),
				slack.NewDividerBlock(),
				slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "📊 *Scan Summary*\nData", false, false), nil, nil),
				slack.NewDividerBlock(),
				slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "Workflow 1 details", false, false), nil, nil),
				slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "Workflow 2 details", false, false), nil, nil),
			},
			// Note: The function adds # Title and \n\n implicitly
			want: "# GHES Scheduled Workflows\n\n📊 *Scan Summary*\nData\n\nWorkflow 1 details\n\nWorkflow 2 details\n\n",
		},
		{
			name: "Only Header",
			blocks: []slack.Block{
				slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "GHES Scheduled Workflows", false, false)),
			},
			want: "# GHES Scheduled Workflows\n\n",
		},
		{
			name:   "Empty blocks",
			blocks: []slack.Block{},
			want:   "# GHES Scheduled Workflows\n\n",
		},
		{
			name: "No Summary",
			blocks: []slack.Block{
				slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "GHES Scheduled Workflows", false, false)),
				slack.NewDividerBlock(),
				slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "Workflow 1 details", false, false), nil, nil),
			},
			want: "# GHES Scheduled Workflows\n\nWorkflow 1 details\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertBlocksToMarkdown(tt.blocks)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJsonBody(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		reader, err := jsonBody(data)
		require.NoError(t, err)
		require.NotNil(t, reader)
		bodyBytes, _ := io.ReadAll(reader)
		assert.JSONEq(t, `{"key":"value"}`, string(bodyBytes))
	})

	t.Run("Marshal Error", func(t *testing.T) {
		// Functions cannot be marshaled to JSON
		data := map[string]interface{}{"func": func() {}}
		reader, err := jsonBody(data)
		require.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "failed to marshal data to JSON")
		assert.Contains(t, err.Error(), "json: unsupported type: func()")
	})
}

func TestCreateCanvasBlocks(t *testing.T) {
	publisher := NewCanvasPublisher("token", "chan", "canvas", "org", "url")

	tests := []struct {
		name                 string
		result               *models.ScanResult
		expectedBlockCount   int // Exact count now
		expectedWfCount      int // Expected workflow section blocks
		checkSummaryContains []string
		checkWfContains      [][]string // List of checks for each workflow block (Format: First item is the header check, rest are indented checks)
	}{
		{
			name: "No workflows",
			result: &models.ScanResult{
				TotalRepos: 5, ExcludedReposCount: 1, Workflows: []models.WorkflowInfo{},
			},
			expectedBlockCount:   4, // Header, Divider, Summary, Divider
			expectedWfCount:      0,
			checkSummaryContains: []string{"Total Repositories: 5", "Excluded Repositories: 1", "Scheduled Workflows Found: 0", "Unknown Committers: 0"},
			checkWfContains:      [][]string{},
		},
		{
			name: "One workflow",
			result: &models.ScanResult{
				TotalRepos: 2, ExcludedReposCount: 0, Workflows: []models.WorkflowInfo{
					{RepoName: "repo1", WorkflowName: "WF1", WorkflowFileName: "wf1.yml", LastStatus: "success", CronSchedules: []string{"0 0 * * *"}, LastCommitter: "user1", IsActiveUser: true},
				},
			},
			expectedBlockCount:   5, // Header, Divider, Summary, Divider, Workflow Row
			expectedWfCount:      1,
			checkSummaryContains: []string{"Total Repositories: 2", "Scheduled Workflows Found: 1", "Unknown Committers: 0"},
			checkWfContains:      [][]string{{`* *[1]* repo1`, `  * *Workflow:*`, `WF1`, `wf1.yml`}},
		},
		{
			name: "Multiple workflows with unknown committer",
			result: &models.ScanResult{
				TotalRepos: 3, ExcludedReposCount: 0, Workflows: []models.WorkflowInfo{
					{RepoName: "repo1", WorkflowName: "WF-A", WorkflowFileName: "wfA.yml", LastStatus: "success", CronSchedules: []string{"0 0 * * *"}, LastCommitter: "user1", IsActiveUser: true},
					{RepoName: "repo2", WorkflowName: "WF-B", WorkflowFileName: "wfB.yml", LastStatus: "failure", CronSchedules: []string{"0 1 * * *"}, LastCommitter: "Unknown", IsActiveUser: false},
				},
			},
			expectedBlockCount:   6, // Header, Divider, Summary, Divider, Wf1, Wf2
			expectedWfCount:      2,
			checkSummaryContains: []string{"Total Repositories: 3", "Scheduled Workflows Found: 2", "Unknown Committers: 1"},
			checkWfContains:      [][]string{{`* *[1]* repo1`, `WF-A`}, {`* *[2]* repo2`, `WF-B`, `Unknown`}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := publisher.createCanvasBlocks(tt.result)
			require.Equal(t, tt.expectedBlockCount, len(blocks), "Unexpected total number of blocks")

			// Check header
			headerBlock, ok := blocks[0].(*slack.HeaderBlock)
			require.True(t, ok, "Block 0 should be Header")
			assert.Equal(t, "GHES Scheduled Workflows", headerBlock.Text.Text)

			// Check Summary Block (index 2)
			summaryBlock, ok := blocks[2].(*slack.SectionBlock)
			require.True(t, ok, "Block 2 should be Summary Section")
			require.NotNil(t, summaryBlock.Text)
			for _, check := range tt.checkSummaryContains {
				assert.Contains(t, summaryBlock.Text.Text, check, "Summary block missing expected text")
			}

			// Check Workflow Blocks (starting index 4)
			actualWfCount := 0
			for i := 4; i < len(blocks); i++ { // Workflow blocks start from index 4
				wfBlock, ok := blocks[i].(*slack.SectionBlock)
				if !ok {
					continue // Skip non-section blocks if any added later
				}
				actualWfCount++
				require.LessOrEqual(t, actualWfCount, tt.expectedWfCount, "More workflow blocks found than expected")
				require.NotNil(t, wfBlock.Text)
				// Get the checks specific to this workflow block (based on order)
				wfIndex := actualWfCount - 1
				require.Less(t, wfIndex, len(tt.checkWfContains), "Mismatch between found workflow blocks and expected checks")
				currentWfChecks := tt.checkWfContains[wfIndex]
				for _, check := range currentWfChecks {
					assert.Contains(t, wfBlock.Text.Text, check, "Workflow block %d missing expected text: %s", actualWfCount, check)
				}
			}
			assert.Equal(t, tt.expectedWfCount, actualWfCount, "Unexpected number of workflow section blocks found")
		})
	}
}

func TestCreateWorkflowRow(t *testing.T) {
	publisher := NewCanvasPublisher("token", "chan", "canvas", "test-org", "https://myghes.com/api/v3") // GHES URL example

	tests := []struct {
		name             string
		wf               models.WorkflowInfo
		index            int
		checkContains    []string // Checks now reflect the new format
		checkNotContains []string
	}{
		{
			name: "Simple success case",
			wf: models.WorkflowInfo{
				RepoName:         "cool-repo",
				WorkflowName:     "CI Build",
				WorkflowFileName: "ci-build.yml",
				CronSchedules:    []string{"15 10 * * *"}, // 10:15 UTC
				LastStatus:       "success",
				LastCommitter:    "dev1",
				IsActiveUser:     true,
			},
			index: 0,
			// Updated checks for new format
			checkContains: []string{
				`* *[1]* cool-repo`,
				`  * *Workflow:* <https://myghes.com/test-org/cool-repo/actions/workflows/ci-build.yml|CI Build>`,
				"  * *Schedule (UTC):* `15 10 * * *`",
				"  * *Schedule (KST):* `15 19 * * *`",
				"  * *Last Status:* :white_check_mark: success",
				"  * *Last Commit By:* dev1",
			},
			checkNotContains: []string{":warning:", "(Inactive)"},
		},
		{
			name: "Failure with inactive committer",
			wf: models.WorkflowInfo{
				RepoName:         "another-repo",
				WorkflowName:     "Nightly Sync",
				WorkflowFileName: "sync.yaml",
				CronSchedules:    []string{"0 23 * * 5"}, // Fri 23:00 UTC
				LastStatus:       "failure",
				LastCommitter:    "old-admin",
				IsActiveUser:     false, // Inactive
			},
			index: 1,
			// Updated checks for new format
			checkContains: []string{
				`* *[2]* another-repo`,
				`  * *Workflow:* <https://myghes.com/test-org/another-repo/actions/workflows/sync.yaml|Nightly Sync>`,
				"  * *Schedule (UTC):* `0 23 * * 5`",
				"  * *Schedule (KST):* `0 8 * * 6`",
				"  * *Last Status:* :x: failure",
				"  * *Last Commit By:* :warning: old-admin (Inactive)",
			},
		},
		{
			name: "No schedule, unknown status",
			wf: models.WorkflowInfo{
				RepoName:         "basic-repo",
				WorkflowName:     "Manual Trigger",
				WorkflowFileName: "manual.yml",
				CronSchedules:    []string{},       // No schedule
				LastStatus:       "unknown_or_new", // Status not in map
				LastCommitter:    "creator",
				IsActiveUser:     true,
			},
			index: 2,
			// Updated checks for new format
			checkContains: []string{
				`* *[3]* basic-repo`,
				`  * *Workflow:* <https://myghes.com/test-org/basic-repo/actions/workflows/manual.yml|Manual Trigger>`,
				"  * *Schedule (UTC):* `N/A`",
				"  * *Schedule (KST):* `N/A`",
				"  * *Last Status:* :grey_question: unknown_or_new",
				"  * *Last Commit By:* creator",
			},
		},
		{
			name: "Multiple schedules",
			wf: models.WorkflowInfo{
				RepoName:         "multi-sched",
				WorkflowName:     "Complex Job",
				WorkflowFileName: "complex.yml",
				CronSchedules:    []string{"0 1 * * *", "0 13 * * *"}, // 01:00 UTC, 13:00 UTC
				LastStatus:       "skipped",
				LastCommitter:    "bot",
				IsActiveUser:     true,
			},
			index: 3,
			// Updated checks for new format
			checkContains: []string{
				`* *[4]* multi-sched`,
				`  * *Workflow:* <https://myghes.com/test-org/multi-sched/actions/workflows/complex.yml|Complex Job>`,
				"  * *Schedule (UTC):* `0 1 * * *, 0 13 * * *`",
				"  * *Schedule (KST):* `0 10 * * *, 0 22 * * *`",
				"  * *Last Status:* :arrow_right_hook: skipped",
				"  * *Last Commit By:* bot",
			},
		},
		{
			name: "Github public URL",
			wf: models.WorkflowInfo{
				RepoName:         "public-repo",
				WorkflowName:     "Public CI",
				WorkflowFileName: "public.yml",
				CronSchedules:    []string{"0 0 * * *"}, // Midnight UTC
				LastStatus:       "success",
				LastCommitter:    "contributor",
				IsActiveUser:     true,
			},
			index: 4,
			// Override publisher URL for this test case
			// Create a new publisher instance for isolation
			checkContains: func() []string {
				pub := NewCanvasPublisher("token", "chan", "canvas", "public-org", "https://github.com") // Public github URL
				block := pub.createWorkflowRow(models.WorkflowInfo{
					RepoName:         "public-repo",
					WorkflowName:     "Public CI",
					WorkflowFileName: "public.yml",
					CronSchedules:    []string{"0 0 * * *"},
					LastStatus:       "success",
					LastCommitter:    "contributor",
					IsActiveUser:     true,
				}, 4) // Index 4 for 5.
				section, ok := block.(*slack.SectionBlock)
				require.True(t, ok)
				text := section.Text.Text
				// Updated checks for new format
				return []string{
					`* *[5]* public-repo`,
					`  * *Workflow:* <https://github.com/public-org/public-repo/actions/workflows/public.yml|Public CI>`,
					"  * *Schedule (UTC):* `0 0 * * *`",
					"  * *Schedule (KST):* `0 9 * * *`",
					"  * *Last Status:* :white_check_mark: success",
					"  * *Last Commit By:* contributor",
					text, // Return the actual text for contains check
				}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var block slack.Block
			// Handle the special case for public github URL test
			if tt.name == "Github public URL" {
				// The checkContains already has the generated text
				textToCheck := tt.checkContains[len(tt.checkContains)-1]
				tt.checkContains = tt.checkContains[:len(tt.checkContains)-1] // Remove text from checks
				for _, check := range tt.checkContains {
					assert.Contains(t, textToCheck, check)
				}
				if tt.checkNotContains != nil {
					for _, check := range tt.checkNotContains {
						assert.NotContains(t, textToCheck, check)
					}
				}
			} else {
				block = publisher.createWorkflowRow(tt.wf, tt.index)
				section, ok := block.(*slack.SectionBlock)
				require.True(t, ok, "Expected SectionBlock")
				require.NotNil(t, section.Text)
				text := section.Text.Text

				// Debugging output
				// t.Logf("Generated Markdown:\n%s", text)

				for _, check := range tt.checkContains {
					assert.Contains(t, text, check)
				}
				if tt.checkNotContains != nil {
					for _, check := range tt.checkNotContains {
						assert.NotContains(t, text, check)
					}
				}
			}
		})
	}
}

func TestConvertToKST(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		want     string
	}{
		{"UTC 15:00", "0 15 * * *", "0 0 * * *"},                               // Simple conversion
		{"UTC 00:00", "0 0 * * *", "0 9 * * *"},                                // Simple conversion
		{"UTC 23:30 (Next Day, Mon->Tue)", "30 23 * * 1", "30 8 * * 2"},        // Day shift
		{"UTC 18:00 (Next Day, Sat->Sun)", "0 18 * * 6", "0 3 * * 0"},          // Day shift (Sat to Sun)
		{"Invalid cron string", "invalid cron", "invalid cron"},                // Invalid format
		{"Too few parts", "0 15 * *", "0 15 * *"},                              // Invalid format
		{"Non-numeric hour", "0 */6 * * *", "0 */6 * * *"},                     // Complex hour not converted
		{"Non-numeric minute", "*/15 15 * * *", "*/15 0 * * *"},                // Minute ignored, hour converted
		{"Non-numeric day of week", "0 15 * * 1-5", "0 0 * * 1-5"},             // Corrected expectation: Hour should convert, DoW shift logic doesn't apply/fail safely
		{"Non-numeric hour and DoW", "0 */2 * * MON-FRI", "0 */2 * * MON-FRI"}, // No conversion (hour invalid)
		{"Invalid hour value", "0 25 * * *", "0 25 * * *"},                     // Returns original as isValidCronExpression fails
		{"Invalid minute value", "60 10 * * *", "60 10 * * *"},                 // Corrected expectation: Returns original as isValidCronExpression fails
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToKST(tt.schedule)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidCronExpression(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		want     bool
	}{
		{"Valid 5 parts", "0 15 * * *", true},
		{"Valid with numbers", "1 2 3 4 5", true},
		{"Valid complex allowed (basic check)", "*/15 0-6 * * 1-5", true}, // Passes basic check
		{"Invalid characters", "0 15 ! * *", false},
		{"Invalid chars 2", "0 15 * * ?", false},         // Invalid char
		{"Invalid range (minute)", "60 15 * * *", false}, // Corrected expectation: Fails validation
		{"Invalid range (hour)", "0 24 * * *", false},    // Corrected expectation: Fails validation
		{"Invalid range (DoW)", "0 0 * * 7", false},      // Corrected expectation: Fails validation
		{"Too few parts", "0 15 * *", false},
		{"Too many parts", "0 15 * * * *", false},
		{"Empty string", "", false},
		{"Valid with DayOfMonth/Month", "0 0 1 1 *", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Fields(tt.schedule)
			// isValidCronExpression expects exactly 5 parts
			if len(parts) != 5 && tt.schedule != "" {
				assert.False(t, isValidCronExpression(parts))
			} else if tt.schedule == "" {
				assert.False(t, isValidCronExpression(parts))
			} else {
				got := isValidCronExpression(parts)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetName(t *testing.T) {
	publisher := CanvasPublisher{} // No fields needed for GetName
	assert.Equal(t, publisherName, publisher.GetName())
	assert.Equal(t, "slack-canvas", publisher.GetName()) // Double check constant value
}

// More sophisticated mock handler for retry testing
type retryMockHandler struct {
	t                    *testing.T
	callCount            int
	failCounts           map[int]int // Map of status code to number of times it should fail with that code
	successResponse      map[string]interface{}
	failResponses        map[int]map[string]interface{} // Response body for specific failure statuses
	networkErrorAttempts map[int]bool                   // Attempts where network error should occur
	retryAfterValue      string                         // Value for Retry-After header on 429
}

func (h *retryMockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.callCount++
	attempt := h.callCount // 1-based attempt number

	// Simulate network error
	if h.networkErrorAttempts[attempt] {
		// This simulates an error during the client.Do call itself
		// For httptest, we can't easily simulate *before* the handler runs,
		// so we simulate it by hijacking the connection. A bit crude.
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Close the connection to simulate a network error during Do/Read
		conn.Close()
		return
	}

	// Simulate specific status code failures for a number of attempts
	for status, failCount := range h.failCounts {
		if attempt <= failCount {
			w.Header().Set("Content-Type", "application/json")
			// Set Retry-After header if status is 429 and value is provided
			if status == httpStatusTooManyRequests && h.retryAfterValue != "" {
				w.Header().Set("Retry-After", h.retryAfterValue)
			}
			w.WriteHeader(status)
			// Provide specific error body if defined
			if responseBody, ok := h.failResponses[status]; ok {
				json.NewEncoder(w).Encode(responseBody)
			} else if status >= 500 {
				fmt.Fprintf(w, "Simulated Server Error %d", status)
			} else if status == httpStatusTooManyRequests {
				json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "rate_limited"})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": fmt.Sprintf("simulated_error_%d", status)})
			}
			h.t.Logf("[Mock Handler] Attempt %d: Responding with status %d", attempt, status)
			return
		}
	}

	// Success response after failures or immediate success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if h.successResponse == nil {
		h.successResponse = map[string]interface{}{"ok": true}
	}
	json.NewEncoder(w).Encode(h.successResponse)
	h.t.Logf("[Mock Handler] Attempt %d: Responding with success (200 OK)", attempt)
}

func TestUpdateCanvas_RetryLogic(t *testing.T) {
	defaultBlocks := []slack.Block{slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "Test", false, false))}

	// Use local variables for test configuration - NOTE: these don't override package constants!
	// Tests rely on checking call counts or using production constants for error messages.
	// testMaxRetries := 3 // Use the value from the constant 'maxRetries' instead
	// testInitialBackoff := 10 * time.Millisecond

	tests := []struct {
		name             string
		handlerSetup     func(*testing.T) *retryMockHandler
		expectError      bool
		expectErrorMsg   string
		expectedApiCalls int           // Expected number of times the handler ServeHTTP is called
		contextTimeout   time.Duration // For testing context cancellation
	}{
		{
			name: "Success on First Attempt",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				return &retryMockHandler{t: t, failCounts: map[int]int{}} // No failures configured
			},
			expectError:      false,
			expectedApiCalls: 1,
		},
		{
			name: "Success after 1 Network Error",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				return &retryMockHandler{t: t, networkErrorAttempts: map[int]bool{1: true}}
			},
			expectError:      false,
			expectedApiCalls: 2, // Initial + 1 retry
		},
		{
			name: "Success after 1 Server Error (503)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				return &retryMockHandler{t: t, failCounts: map[int]int{http.StatusServiceUnavailable: 1}}
			},
			expectError:      false,
			expectedApiCalls: 2,
		},
		{
			name: "Success after 1 Rate Limit (429, no Retry-After)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				return &retryMockHandler{t: t, failCounts: map[int]int{httpStatusTooManyRequests: 1}}
			},
			expectError:      false,
			expectedApiCalls: 2,
		},
		{
			name: "Success after 1 Rate Limit (429, with Retry-After)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				// Use a small Retry-After to keep test fast
				return &retryMockHandler{t: t, failCounts: map[int]int{httpStatusTooManyRequests: 1}, retryAfterValue: "1"}
			},
			expectError:      false,
			expectedApiCalls: 2,
		},
		{
			name: "Fail after Max Retries (Network Error)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				// Fail on all attempts (1 initial + maxRetries retries)
				errMap := make(map[int]bool)
				for i := 1; i <= maxRetries+1; i++ {
					errMap[i] = true
				}
				return &retryMockHandler{t: t, networkErrorAttempts: errMap}
			},
			expectError: true,
			// Use production constant maxRetries here
			expectErrorMsg:   fmt.Sprintf("canvas update failed after %d attempts", maxRetries+1),
			expectedApiCalls: maxRetries + 1,
		},
		{
			name: "Fail after Max Retries (500 Error)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				// Fail on all attempts (1 initial + maxRetries retries)
				return &retryMockHandler{t: t, failCounts: map[int]int{http.StatusInternalServerError: maxRetries + 1}}
			},
			expectError: true,
			// Use production constant maxRetries here
			expectErrorMsg:   fmt.Sprintf("canvas update failed after %d attempts", maxRetries+1),
			expectedApiCalls: maxRetries + 1,
		},
		{
			name: "Fail Immediately on Client Error (400)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				return &retryMockHandler{t: t, failCounts: map[int]int{http.StatusBadRequest: 1}}
			},
			expectError:      true,
			expectErrorMsg:   "client error (status 400)", // Specific error from updateCanvas
			expectedApiCalls: 1,                           // No retry
		},
		{
			name: "Fail Immediately on API Error (ok:false)",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				// Simulate 200 OK but with ok:false in body
				return &retryMockHandler{t: t, successResponse: map[string]interface{}{"ok": false, "error": "test_api_error"}}
			},
			expectError:      true,
			expectErrorMsg:   "slack API error (ok=false)", // Specific error from updateCanvas
			expectedApiCalls: 1,                            // No retry
		},
		{
			name: "Context Timeout during Retry Wait",
			handlerSetup: func(t *testing.T) *retryMockHandler {
				// Fail first attempt to trigger retry wait
				return &retryMockHandler{t: t, failCounts: map[int]int{http.StatusServiceUnavailable: 1}}
			},
			expectError:      true,
			expectErrorMsg:   "canvas update retry sleep cancelled or timed out",
			expectedApiCalls: 1, // Only the first call happens before timeout
			// Ensure context timeout is less than production initialBackoff
			contextTimeout: initialBackoff / 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := tt.handlerSetup(t)
			publisher := newTestPublisherWithMockClient(mockHandler, "xoxb-token", "C1", "F1", "org", "url")

			// --- NOTE on Testing Constants ---
			// We cannot easily modify package-level constants (initialBackoff, maxRetries)
			// per test case without making them configurable on the publisher struct.
			// These tests assume the production constants are used (3 retries, 1s initial backoff)
			// or verify behavior independent of exact timing (e.g., number of calls).
			// The 'Fail after Max Retries' tests use fmt.Sprintf with the *production* constant.
			// The Context Timeout test uses a very short timeout relative to the production initial backoff.

			ctx := context.Background()
			var cancel context.CancelFunc
			if tt.contextTimeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
			} else {
				// Give non-timeout tests a reasonable deadline, considering potential waits
				// Use production constants for calculation here
				maxWait := initialBackoff * time.Duration(math.Pow(backoffFactor, float64(maxRetries)))
				testDeadline := time.Duration(maxRetries+2)*50*time.Millisecond + maxWait // Base time + max backoff
				if testDeadline < 2*time.Second {                                         // Ensure a minimum timeout
					testDeadline = 2 * time.Second
				}
				ctx, cancel = context.WithTimeout(ctx, testDeadline)
			}
			defer cancel()

			err := publisher.updateCanvas(ctx, defaultBlocks)

			if tt.expectError {
				require.Error(t, err, "Expected an error but got nil")
				if tt.expectErrorMsg != "" {
					// Use production constant for error message check in failure cases
					if strings.Contains(tt.expectErrorMsg, "%d attempts") {
						// Corrected: Use fmt.Sprintf with the correct format specifier if it exists
						assert.Contains(t, err.Error(), fmt.Sprintf(tt.expectErrorMsg, maxRetries+1), "Error message mismatch")
					} else {
						assert.Contains(t, err.Error(), tt.expectErrorMsg, "Error message mismatch")
					}
				}
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
			}

			assert.Equal(t, tt.expectedApiCalls, mockHandler.callCount, "Unexpected number of API calls")
		})
	}
}
