package scanner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

// TestExtractSchedules verifies the extraction of cron schedules from workflow configurations.
// Tests single schedule, multiple schedules, and no schedule cases.
func TestExtractSchedules(t *testing.T) {
	tests := []struct {
		name     string
		workflow map[string]interface{}
		want     []string
	}{
		{
			name: "단일 크론 스케줄",
			workflow: map[string]interface{}{
				"on": map[string]interface{}{
					"schedule": []interface{}{
						map[string]interface{}{
							"cron": "0 0 * * *",
						},
					},
				},
			},
			want: []string{"0 0 * * *"},
		},
		{
			name: "여러 크론 스케줄",
			workflow: map[string]interface{}{
				"on": map[string]interface{}{
					"schedule": []interface{}{
						map[string]interface{}{"cron": "0 0 * * *"},
						map[string]interface{}{"cron": "0 12 * * *"},
					},
				},
			},
			want: []string{"0 0 * * *", "0 12 * * *"},
		},
		{
			name:     "스케줄 없음",
			workflow: map[string]interface{}{},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSchedules(tt.workflow)
			if len(got) != len(tt.want) {
				t.Errorf("extractSchedules() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractSchedules()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestScanRepository tests the complete repository scanning workflow including
// API interactions, content retrieval, and result validation.
func TestScanRepository(t *testing.T) {
	// 테스트 서버 설정
	server := setupTestServer(t)
	defer server.Close()

	// 테스트 실행
	results := runScanTest(t, server)

	// 결과 검증
	validateTestResults(t, results)
}

// setupTestServer initializes a mock HTTP server that simulates GitHub Enterprise API endpoints.
// Returns configured test server instance.
func setupTestServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// API 핸들러 등록
	registerWorkflowsHandler(t, mux, server.URL)
	registerDirectoryHandler(t, mux, server.URL)
	registerFileContentHandler(t, mux)
	registerWorkflowRunsHandler(t, mux)
	register404Handler(t, mux)

	return server
}

// registerWorkflowsHandler sets up a mock endpoint for workflow listing API.
// Responds with a single test workflow configuration.
func registerWorkflowsHandler(t *testing.T, mux *http.ServeMux, baseURL string) {
	mux.HandleFunc("/api/v3/repos/testorg/testrepo/actions/workflows", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflows request")
		w.Header().Set("Content-Type", "application/json")

		workflows := &github.Workflows{
			TotalCount: github.Int(1),
			Workflows: []*github.Workflow{
				{
					ID:      github.Int64(1),
					Name:    github.String("Test Workflow"),
					Path:    github.String(".github/workflows/test.yml"),
					HTMLURL: github.String(baseURL + "/testorg/testrepo/blob/master/.github/workflows/test.yml"),
				},
			},
		}
		if err := json.NewEncoder(w).Encode(workflows); err != nil {
			t.Fatal(err)
		}
	})
}

// registerDirectoryHandler sets up a mock endpoint for workflow directory contents.
// Returns a single test.yml workflow file listing.
func registerDirectoryHandler(t *testing.T, mux *http.ServeMux, baseURL string) {
	mux.HandleFunc("/api/v3/repos/testorg/testrepo/contents/.github/workflows", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflows directory request")
		w.Header().Set("Content-Type", "application/json")

		files := []*github.RepositoryContent{
			{
				Name:    github.String("test.yml"),
				Path:    github.String(".github/workflows/test.yml"),
				Type:    github.String("file"),
				Size:    github.Int(100),
				SHA:     github.String("abc123"),
				Content: nil,
				URL:     github.String(baseURL + "/api/v3/repos/testorg/testrepo/contents/.github/workflows/test.yml"),
				HTMLURL: github.String(baseURL + "/testorg/testrepo/blob/master/.github/workflows/test.yml"),
				GitURL:  github.String(baseURL + "/api/v3/repos/testorg/testrepo/git/blobs/abc123"),
			},
		}
		if err := json.NewEncoder(w).Encode(files); err != nil {
			t.Fatal(err)
		}
	})
}

// registerFileContentHandler sets up a mock endpoint for workflow file content retrieval.
// Returns base64 encoded YAML content with a test schedule.
func registerFileContentHandler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/api/v3/repos/testorg/testrepo/contents/.github/workflows/test.yml", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflow file content request")
		w.Header().Set("Content-Type", "application/json")

		content := `name: Test Workflow
on:
  schedule:
    - cron: "0 0 * * *"`

		response := &github.RepositoryContent{
			Type:     github.String("file"),
			Encoding: github.String("base64"),
			Size:     github.Int(len(content)),
			Name:     github.String("test.yml"),
			Path:     github.String(".github/workflows/test.yml"),
			Content:  github.String(base64.StdEncoding.EncodeToString([]byte(content))),
			SHA:      github.String("abc123"),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatal(err)
		}
	})
}

// registerWorkflowRunsHandler sets up a mock endpoint for workflow run history.
// Returns a single completed workflow run.
func registerWorkflowRunsHandler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/api/v3/repos/testorg/testrepo/actions/workflows/1/runs", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflow runs request")
		w.Header().Set("Content-Type", "application/json")

		runs := &github.WorkflowRuns{
			TotalCount: github.Int(1),
			WorkflowRuns: []*github.WorkflowRun{
				{
					ID:     github.Int64(1),
					Status: github.String("completed"),
				},
			},
		}
		if err := json.NewEncoder(w).Encode(runs); err != nil {
			t.Fatal(err)
		}
	})
}

// register404Handler sets up a catch-all handler for unmatched routes.
// Logs unhandled requests and returns 404 status.
func register404Handler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("WARNING - Unhandled request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
}

// runScanTest executes the repository scan with mock server configuration.
// Returns the scan results for validation.
func runScanTest(t *testing.T, server *httptest.Server) []models.WorkflowInfo {
	scanner := NewScanner("test-token", server.URL, 1)

	repo := &github.Repository{
		Name:     github.String("testrepo"),
		Owner:    &github.User{Login: github.String("testorg")},
		FullName: github.String("testorg/testrepo"),
	}

	var results []models.WorkflowInfo
	var resultMutex sync.Mutex

	ctx := context.Background()
	err := scanner.scanRepository(ctx, "testorg", repo, &results, &resultMutex)
	if err != nil {
		t.Fatalf("scanRepository() failed: %v", err)
	}

	return results
}

// validateTestResults checks if scan results match expected workflow information.
// Verifies repository name, workflow details, schedules and status.
func validateTestResults(t *testing.T, results []models.WorkflowInfo) {
	if len(results) == 0 {
		t.Fatal("scanRepository() got no results, want at least 1")
	}

	want := models.WorkflowInfo{
		RepoName:      "testrepo",
		WorkflowName:  "Test Workflow",
		WorkflowID:    1,
		CronSchedules: []string{"0 0 * * *"},
		LastStatus:    "completed",
	}

	if !reflect.DeepEqual(results[0], want) {
		t.Errorf("scanRepository() got = %+v, want %+v", results[0], want)
	}
}
