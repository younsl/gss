package scanner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/stretchr/testify/assert"
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
	t.Run("단일 저장소 스캔", func(t *testing.T) {
		mockClient := newMockGitHubClient()
		scanner := NewScanner(mockClient, 1)

		ctx := context.Background()
		repo := &github.Repository{
			Name:          github.String("test-repo"),
			Owner:         &github.User{Login: github.String("owner")},
			Archived:      github.Bool(false),
			DefaultBranch: github.String("main"),
		}

		results, err := scanner.scanRepository(ctx, repo)
		assert.NoError(t, err)
		assert.NotEmpty(t, results, "Should have workflow results")
	})
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

// runScanTest executes the scan test with the given test server
func runScanTest(t *testing.T, server *httptest.Server) []models.WorkflowInfo {
	client := newMockGitHubClient()

	// maxConcurrency 인자 추가
	scanner := NewScanner(client, 1)

	// 테스트용 저장소 생성
	repo := &github.Repository{
		Name: github.String("testrepo"),
		Owner: &github.User{
			Login: github.String("testorg"),
		},
		DefaultBranch: github.String("main"),
	}

	// 스캔 실행
	results, err := scanner.scanRepository(context.Background(), repo)
	if err != nil {
		t.Fatalf("Failed to scan repository: %v", err)
	}

	return results
}

// validateTestResults verifies the scan results
func validateTestResults(t *testing.T, results []models.WorkflowInfo) {
	assert.NotEmpty(t, results, "Expected non-empty results")
	assert.Equal(t, 1, len(results), "Expected exactly one workflow")

	workflow := results[0]
	assert.Equal(t, "testrepo", workflow.RepoName)
	assert.Equal(t, "Test Workflow", workflow.WorkflowName)
	assert.NotEmpty(t, workflow.CronSchedules, "Expected non-empty cron schedules")
}

// GitHubClientInterface는 테스트에 필요한 GitHub API 메서드들을 정의합니다
type GitHubClientInterface interface {
	ListByOrg(ctx context.Context, org string, opts *github.RepositoryListByOrgOptions) ([]*github.Repository, *github.Response, error)
	ListWorkflows(ctx context.Context, owner, repo string, opts *github.ListOptions) (*github.Workflows, *github.Response, error)
	GetWorkflow(ctx context.Context, owner, repo string, workflowID int64) (*github.Workflow, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	ListWorkflowRuns(ctx context.Context, owner, repo string, workflowID int64, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error)
}

// 테스트용 워크플로우 YAML 정의
const testWorkflowYAML = `
name: Test Workflow
on:
  schedule:
    - cron: "0 0 * * *"
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
`

type mockGitHubClient struct {
	shouldErr      bool
	invalidContent bool
	repoCount      int // 추가: 반환할 레포지토리 수
}

func newMockGitHubClient(opts ...func(*mockGitHubClient)) *mockGitHubClient {
	client := &mockGitHubClient{
		repoCount: 1, // 기본값 설정
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (m *mockGitHubClient) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, nil, fmt.Errorf("mock error")
	}

	content := testWorkflowYAML
	if m.invalidContent {
		content = "invalid yaml"
	}

	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))
	return &github.RepositoryContent{
		Content: github.String(encodedContent),
		Path:    github.String(path),
	}, nil, nil, nil
}

func (m *mockGitHubClient) ListWorkflows(ctx context.Context, owner, repo string, opts *github.ListOptions) (*github.Workflows, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, fmt.Errorf("mock error")
	}

	return &github.Workflows{
		Workflows: []*github.Workflow{
			{
				ID:   github.Int64(1),
				Name: github.String("Test Workflow"),
				Path: github.String(".github/workflows/test.yml"),
			},
		},
	}, nil, nil
}

func (m *mockGitHubClient) ListByOrg(ctx context.Context, org string, opts *github.RepositoryListByOrgOptions) ([]*github.Repository, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, fmt.Errorf("mock error")
	}

	repos := make([]*github.Repository, m.repoCount)
	for i := 0; i < m.repoCount; i++ {
		repos[i] = &github.Repository{
			Name:          github.String(fmt.Sprintf("test-repo-%d", i+1)),
			Archived:      github.Bool(false),
			DefaultBranch: github.String("main"),
		}
	}
	return repos, &github.Response{NextPage: 0}, nil
}

func (m *mockGitHubClient) GetWorkflow(ctx context.Context, owner, repo string, workflowID int64) (*github.Workflow, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, fmt.Errorf("mock error")
	}

	return &github.Workflow{
		ID:   github.Int64(workflowID),
		Name: github.String("Test Workflow"),
		Path: github.String(".github/workflows/test.yml"),
	}, nil, nil
}

func (m *mockGitHubClient) ListWorkflowRuns(ctx context.Context, owner, repo string, workflowID int64, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, fmt.Errorf("mock error")
	}

	return &github.WorkflowRuns{
		TotalCount: github.Int(1),
		WorkflowRuns: []*github.WorkflowRun{
			{
				ID:         github.Int64(1),
				Name:       github.String("Test Run"),
				CreatedAt:  &github.Timestamp{Time: time.Now()},
				UpdatedAt:  &github.Timestamp{Time: time.Now()},
				Status:     github.String("completed"),
				Conclusion: github.String("success"),
			},
		},
	}, nil, nil
}

func (m *mockGitHubClient) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, fmt.Errorf("mock error")
	}

	return []*github.RepositoryCommit{
		{
			SHA: github.String("abc123"),
			Commit: &github.Commit{
				Message: github.String("Test commit"),
				Author: &github.CommitAuthor{
					Date: &github.Timestamp{Time: time.Now()},
				},
			},
		},
	}, nil, nil
}

func (m *mockGitHubClient) GetUser(ctx context.Context, username string) (*github.User, *github.Response, error) {
	if m.shouldErr {
		return nil, nil, fmt.Errorf("mock error")
	}

	return &github.User{
		Login: github.String(username),
		Name:  github.String("Test User"),
		Email: github.String("test@example.com"),
	}, nil, nil
}

func TestScanScheduledWorkflows(t *testing.T) {
	t.Run("전체 저장소 스캔", func(t *testing.T) {
		mockClient := newMockGitHubClient()
		scanner := NewScanner(mockClient, 1)

		result, err := scanner.ScanScheduledWorkflows("owner")
		assert.NoError(t, err)
		assert.NotNil(t, result, "Result should not be nil")
		assert.NotEmpty(t, result.Workflows, "Should have workflow results")
	})
}

func TestScanScheduledWorkflows_Errors(t *testing.T) {
	t.Run("API 호출 실패", func(t *testing.T) {
		mockClient := newMockGitHubClient(func(m *mockGitHubClient) {
			m.shouldErr = true
		})
		scanner := NewScanner(mockClient, 1)

		result, err := scanner.ScanScheduledWorkflows("owner")
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("잘못된 워크플로우 형식", func(t *testing.T) {
		mockClient := newMockGitHubClient(func(m *mockGitHubClient) {
			m.invalidContent = true
			m.repoCount = 1
		})
		scanner := NewScanner(mockClient, 1)

		result, err := scanner.ScanScheduledWorkflows("owner")
		assert.Error(t, err)
		if result != nil {
			assert.Empty(t, result.Workflows)
		}
	})
}

func TestConcurrentScanning(t *testing.T) {
	t.Run("동시 스캔 처리", func(t *testing.T) {
		mockClient := newMockGitHubClient(func(m *mockGitHubClient) {
			m.repoCount = 5
		})
		scanner := NewScanner(mockClient, 5)

		result, err := scanner.ScanScheduledWorkflows("owner")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 5, len(result.Workflows))
	})
}
