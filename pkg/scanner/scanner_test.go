package scanner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	// Common test values
	testOrg         = "test-org"
	testRepo        = "test-repo"
	testUser        = "test-user"
	testWorkflowID  = int64(1)
	testWorkflowSHA = "abc123"

	// Test workflow values
	testWorkflowName = "Test Workflow"
	testWorkflowPath = ".github/workflows/test.yml"
	testCronSchedule = "0 0 * * *"

	// Test API paths
	apiV3Prefix = "/api/v3"
)

// getAPIPath returns the full API path for a given endpoint
func getAPIPath(endpoint string) string {
	return apiV3Prefix + endpoint
}

// getRepoPath returns the full repository API path
func getRepoPath(endpoint string) string {
	return getAPIPath(fmt.Sprintf("/repos/%s/%s%s", testOrg, testRepo, endpoint))
}

// TestExtractSchedules verifies the extraction of cron schedules from workflow configurations.
func TestExtractSchedules(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
		errMsg  string
	}{
		{
			name: "single schedule",
			content: fmt.Sprintf(`name: %s
on:
  schedule:
    - cron: "%s"`, testWorkflowName, testCronSchedule),
			want:    []string{testCronSchedule},
			wantErr: false,
		},
		{
			name: "no schedule",
			content: `name: Test Workflow
on:
  push:
    branches: [ main ]`,
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			content: `name: Test Workflow
on:
  schedule:
    - cron: [invalid]`,
			wantErr: true,
			errMsg:  "failed to unmarshal workflow file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workflow map[string]interface{}
			err := yaml.Unmarshal([]byte(tt.content), &workflow)
			if err != nil && !tt.wantErr {
				t.Fatalf("failed to unmarshal test content: %v", err)
			}
			if tt.wantErr {
				return // Skip schedule extraction for invalid YAML
			}

			got := extractSchedules(workflow)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractSchedules() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestScanRepository tests the complete repository scanning workflow
func TestScanRepository(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := github.NewClient(&http.Client{})
	client.BaseURL, _ = url.Parse(server.URL + "/api/v3/")

	scanner := &Scanner{
		client: client,
	}

	var results []models.WorkflowInfo
	var resultMutex sync.Mutex

	repo := &github.Repository{
		Name:  github.String(testRepo),
		Owner: &github.User{Login: github.String(testOrg)},
	}

	err := scanner.scanRepository(context.Background(), testOrg, repo, &results, &resultMutex)
	if err != nil {
		t.Fatalf("scanRepository() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 schedule, got %d", len(results))
	}

	want := models.WorkflowInfo{
		RepoName:      testRepo,
		WorkflowName:  testWorkflowName,
		WorkflowID:    testWorkflowID,
		CronSchedules: []string{testCronSchedule},
		LastStatus:    "completed",
	}

	if !reflect.DeepEqual(results[0], want) {
		t.Errorf("scanRepository() got = %+v, want %+v", results[0], want)
	}
}

// setupTestServer initializes a mock HTTP server that simulates GitHub Enterprise API endpoints.
func setupTestServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	registerWorkflowsHandler(t, mux, server.URL)
	registerDirectoryHandler(t, mux, server.URL)
	registerFileContentHandler(t, mux)
	registerWorkflowRunsHandler(t, mux)
	registerUserHandler(t, mux)
	registerReposHandler(t, mux)
	register404Handler(t, mux)

	return server
}

// registerWorkflowsHandler sets up a mock endpoint for workflow listing API.
func registerWorkflowsHandler(t *testing.T, mux *http.ServeMux, baseURL string) {
	mux.HandleFunc(getRepoPath("/actions/workflows"), func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflows request")
		w.Header().Set("Content-Type", "application/json")

		workflows := &github.Workflows{
			TotalCount: github.Int(1),
			Workflows: []*github.Workflow{
				{
					ID:      github.Int64(testWorkflowID),
					Name:    github.String(testWorkflowName),
					Path:    github.String(testWorkflowPath),
					HTMLURL: github.String(fmt.Sprintf("%s/%s/%s/blob/master/%s", baseURL, testOrg, testRepo, testWorkflowPath)),
				},
			},
		}
		if err := json.NewEncoder(w).Encode(workflows); err != nil {
			t.Fatal(err)
		}
	})
}

// registerDirectoryHandler sets up a mock endpoint for workflow directory contents.
func registerDirectoryHandler(t *testing.T, mux *http.ServeMux, baseURL string) {
	mux.HandleFunc(getRepoPath("/contents/.github/workflows"), func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflows directory request")
		w.Header().Set("Content-Type", "application/json")

		files := []*github.RepositoryContent{
			{
				Name:    github.String("test.yml"),
				Path:    github.String(testWorkflowPath),
				Type:    github.String("file"),
				Size:    github.Int(100),
				SHA:     github.String(testWorkflowSHA),
				URL:     github.String(fmt.Sprintf("%s%s", baseURL, getRepoPath("/contents/"+testWorkflowPath))),
				HTMLURL: github.String(fmt.Sprintf("%s/%s/%s/blob/master/%s", baseURL, testOrg, testRepo, testWorkflowPath)),
				GitURL:  github.String(fmt.Sprintf("%s%s/git/blobs/%s", baseURL, getRepoPath(""), testWorkflowSHA)),
			},
		}
		if err := json.NewEncoder(w).Encode(files); err != nil {
			t.Fatal(err)
		}
	})
}

// registerFileContentHandler sets up a mock endpoint for workflow file content.
func registerFileContentHandler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc(getRepoPath("/contents/"+testWorkflowPath), func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflow file content request")
		w.Header().Set("Content-Type", "application/json")

		content := fmt.Sprintf(`name: %s
on:
  schedule:
    - cron: "%s"`, testWorkflowName, testCronSchedule)

		response := &github.RepositoryContent{
			Type:     github.String("file"),
			Encoding: github.String("base64"),
			Size:     github.Int(len(content)),
			Name:     github.String("test.yml"),
			Path:     github.String(testWorkflowPath),
			Content:  github.String(base64.StdEncoding.EncodeToString([]byte(content))),
			SHA:      github.String(testWorkflowSHA),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatal(err)
		}
	})
}

// registerWorkflowRunsHandler sets up a mock endpoint for workflow runs API.
func registerWorkflowRunsHandler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc(fmt.Sprintf("%s/actions/workflows/%d/runs", getRepoPath(""), testWorkflowID), func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling workflow runs request")
		w.Header().Set("Content-Type", "application/json")

		runs := &github.WorkflowRuns{
			TotalCount: github.Int(1),
			WorkflowRuns: []*github.WorkflowRun{
				{
					ID:     github.Int64(testWorkflowID),
					Name:   github.String(testWorkflowName),
					Status: github.String("completed"),
					CreatedAt: &github.Timestamp{
						Time: time.Now(),
					},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(runs); err != nil {
			t.Fatal(err)
		}
	})
}

// registerUserHandler sets up a mock endpoint for user information.
func registerUserHandler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc(getAPIPath("/user"), func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling user request")
		w.Header().Set("Content-Type", "application/json")

		user := &github.User{
			Login: github.String(testUser),
			Type:  github.String("User"),
		}
		if err := json.NewEncoder(w).Encode(user); err != nil {
			t.Fatal(err)
		}
	})
}

// registerReposHandler sets up a mock endpoint for repository listing.
func registerReposHandler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc(getAPIPath("/user/repos"), func(w http.ResponseWriter, r *http.Request) {
		t.Logf("DEBUG - Handling repository list request")
		w.Header().Set("Content-Type", "application/json")

		repos := []*github.Repository{
			{
				Name:    github.String(testRepo),
				Private: github.Bool(true),
			},
		}
		if err := json.NewEncoder(w).Encode(repos); err != nil {
			t.Fatal(err)
		}
	})
}

// register404Handler sets up a catch-all handler for unmatched routes.
func register404Handler(t *testing.T, mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("WARNING - Unhandled request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
}
