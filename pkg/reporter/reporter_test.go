package reporter

import (
	"strings"
	"testing"

	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

// TestFormatReport tests if ConsoleFormatter's FormatReport function
// correctly formats the scan results
func TestFormatReport(t *testing.T) {
	formatter := &ConsoleFormatter{}

	tests := []struct {
		name     string
		input    *models.ScanResult
		expected []string
	}{
		{
			name: "single workflow",
			input: &models.ScanResult{
				Workflows: []models.WorkflowInfo{
					{
						RepoName:      "test-repo",
						WorkflowName:  "nightly-build",
						CronSchedules: []string{"0 0 * * *"},
						LastStatus:    "success",
					},
				},
				TotalRepos: 1,
			},
			expected: []string{
				"Scheduled Workflows Summary:",
				"NO    REPOSITORY                            WORKFLOW                              UTC SCHEDULE    KST SCHEDULE    LAST STATUS",
				"1     test-repo                             nightly-build                         0 0 * * *       0 9 * * *       success",
				"Scanned: 1 repos, 1 workflows",
			},
		},
		{
			name: "empty result",
			input: &models.ScanResult{
				Workflows:  []models.WorkflowInfo{},
				TotalRepos: 0,
			},
			expected: []string{
				"No workflows found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.FormatReport(tt.input)

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("\nexpected output to contain:\n%s\n\ngot:\n%s", expected, output)
				}
			}
		})
	}
}

// TestTruncateString tests if long strings are truncated correctly
func TestTruncateString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "test",
			maxLen: 10,
			want:   "test      ",
		},
		{
			name:   "long string",
			input:  "very-long-repository-name",
			maxLen: 10,
			want:   "very-lon..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestConvertToKST tests if UTC cron expressions are correctly converted to KST
func TestConvertToKST(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "UTC midnight",
			input: "0 0 * * *",
			want:  "0 9 * * *",
		},
		{
			name:  "date change",
			input: "0 20 * * 1",
			want:  "0 5 * * 2",
		},
		{
			name:  "invalid cron",
			input: "invalid",
			want:  "invalid",
		},
		{
			name:  "complex schedule",
			input: "*/15 */2 * * *",
			want:  "*/15 9 * * *",
		},
		{
			name:  "range values",
			input: "0 1-3 * * *",
			want:  "0 10 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToKST(tt.input)
			if got != tt.want {
				t.Errorf("convertToKST() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsValidCronExpression tests cron expression validation
func TestIsValidCronExpression(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  bool
	}{
		{
			name:  "valid cron",
			parts: []string{"0", "0", "*", "*", "*"},
			want:  true,
		},
		{
			name:  "invalid minute",
			parts: []string{"60", "0", "*", "*", "*"},
			want:  false,
		},
		{
			name:  "invalid hour",
			parts: []string{"0", "24", "*", "*", "*"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidCronExpression(tt.parts)
			if got != tt.want {
				t.Errorf("isValidCronExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAtoi tests string to integer conversion functionality
func TestAtoi(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"wildcard", "*", 0},
		{"number", "5", 5},
		{"invalid input", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := atoi(tt.input)
			if got != tt.want {
				t.Errorf("atoi() = %v, want %v", got, tt.want)
			}
		})
	}
}
