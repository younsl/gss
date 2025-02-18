package scanner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"encoding/base64"

	"github.com/google/go-github/v50/github"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

// GitHubClient는 GitHub API 호출에 필요한 메서드들을 정의합니다
type GitHubClient interface {
	ListByOrg(ctx context.Context, org string, opts *github.RepositoryListByOrgOptions) ([]*github.Repository, *github.Response, error)
	ListWorkflows(ctx context.Context, owner, repo string, opts *github.ListOptions) (*github.Workflows, *github.Response, error)
	GetWorkflow(ctx context.Context, owner, repo string, workflowID int64) (*github.Workflow, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
	ListWorkflowRuns(ctx context.Context, owner, repo string, workflowID int64, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
}

// Scanner는 GitHub 워크플로우를 스캔하는 구조체입니다
type Scanner struct {
	client          GitHubClient // 인터페이스로 변경
	concurrentScans int
}

// NewScanner는 새로운 Scanner 인스턴스를 생성합니다
func NewScanner(client GitHubClient, concurrentScans int) *Scanner {
	return &Scanner{
		client:          client,
		concurrentScans: concurrentScans,
	}
}

// ScanScheduledWorkflows scans all repositories in an organization for scheduled workflows
// Returns scan results containing workflow information and total repository count
// May return error if repository listing or scanning fails
func (s *Scanner) ScanScheduledWorkflows(org string) (*models.ScanResult, error) {
	start := time.Now()
	maxRoutines := int32(0)
	activeRoutines := int32(0)

	ctx := context.Background()
	var results []models.WorkflowInfo
	var resultMutex sync.Mutex
	var scanErrors []error
	var errorMutex sync.Mutex

	// 아카이빙되지 않은 레포지터리만 조회
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "sources",
		Sort:        "full_name",
		Direction:   "asc",
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := s.client.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	sem := make(chan struct{}, s.concurrentScans)
	var wg sync.WaitGroup

	for i, repo := range allRepos {
		if repo.GetArchived() {
			continue
		}

		log.Printf("Scanning repository (%d/%d): %s", i+1, len(allRepos), repo.GetName())
		wg.Add(1)
		sem <- struct{}{}

		go func(repo *github.Repository) {
			atomic.AddInt32(&activeRoutines, 1)
			atomic.CompareAndSwapInt32(&maxRoutines, atomic.LoadInt32(&activeRoutines)-1, atomic.LoadInt32(&activeRoutines))

			defer func() {
				atomic.AddInt32(&activeRoutines, -1)
				wg.Done()
				<-sem
			}()

			if err := s.scanRepository(ctx, org, repo, &results, &resultMutex); err != nil {
				errorMutex.Lock()
				scanErrors = append(scanErrors, err)
				errorMutex.Unlock()
			}
		}(repo)
	}

	wg.Wait()
	log.Printf("Scan completed: %d repositories in %v with max %d concurrent goroutines",
		len(allRepos), time.Since(start), maxRoutines)

	if len(scanErrors) > 0 {
		return nil, fmt.Errorf("encountered %d errors during scan: %v", len(scanErrors), scanErrors)
	}

	return &models.ScanResult{
		Workflows:  results,
		TotalRepos: len(allRepos),
	}, nil
}

// scanRepository scans a single repository for workflow files and their schedules
// Extracts cron schedules and last run status for each workflow
// Updates results slice through mutex for thread safety
func (s *Scanner) scanRepository(ctx context.Context, org string, repo *github.Repository, results *[]models.WorkflowInfo, resultMutex *sync.Mutex) error {
	workflows, _, err := s.client.ListWorkflows(ctx, org, repo.GetName(), nil)
	if err != nil {
		return fmt.Errorf("failed to list workflows for %s: %w", repo.GetName(), err)
	}

	for _, workflow := range workflows.Workflows {
		content, _, _, err := s.client.GetContents(ctx, org, repo.GetName(), workflow.GetPath(), nil)
		if err != nil {
			return fmt.Errorf("failed to get workflow content: %w", err)
		}

		// GetContent()의 두 반환값을 모두 처리
		contentStr, err := content.GetContent()
		if err != nil {
			return fmt.Errorf("failed to get content string: %w", err)
		}

		decodedContent, err := base64.StdEncoding.DecodeString(contentStr)
		if err != nil {
			return fmt.Errorf("failed to decode workflow content: %w", err)
		}

		// 워크플로우 콘텐츠 파싱
		var workflowData map[string]interface{}
		if err := yaml.Unmarshal([]byte(decodedContent), &workflowData); err != nil {
			// YAML 파싱 에러를 상위로 전파
			return fmt.Errorf("failed to parse workflow %s: %w", workflow.GetPath(), err)
		}

		schedules, err := extractSchedulesWithValidation(decodedContent)
		if err != nil || len(schedules) == 0 {
			log.Printf("No schedules found in %s: %v", workflow.GetPath(), err)
			continue
		}

		runs, _, err := s.client.ListWorkflowRuns(ctx, org, repo.GetName(), workflow.GetID(), &github.ListWorkflowRunsOptions{
			ListOptions: github.ListOptions{PerPage: 1},
		})
		if err != nil {
			return fmt.Errorf("failed to get workflow runs for %s: %w", workflow.GetName(), err)
		}

		lastStatus := "Unknown"
		if runs != nil && len(runs.WorkflowRuns) > 0 {
			lastStatus = runs.WorkflowRuns[0].GetStatus()
		}

		commits, _, err := s.client.ListCommits(ctx, org, repo.GetName(), &github.CommitsListOptions{
			Path:        workflow.GetPath(),
			ListOptions: github.ListOptions{PerPage: 1},
		})
		if err != nil {
			log.Printf("Failed to get commits for %s: %v", workflow.GetPath(), err)
			continue
		}

		lastCommitter := "Unknown"
		if len(commits) > 0 {
			commit := commits[0]
			if author := commit.GetAuthor(); author != nil {
				lastCommitter = author.GetLogin()
			}
			if lastCommitter == "" && commit.Commit != nil && commit.Commit.Author != nil {
				lastCommitter = commit.Commit.Author.GetName()
			}
			if lastCommitter == "" {
				lastCommitter = "Unknown"
			}
		}

		resultMutex.Lock()
		*results = append(*results, models.WorkflowInfo{
			RepoName:      repo.GetName(),
			WorkflowName:  workflow.GetName(),
			WorkflowID:    workflow.GetID(),
			CronSchedules: schedules,
			LastStatus:    lastStatus,
			LastCommitter: lastCommitter,
		})
		resultMutex.Unlock()
	}
	return nil
}

// extractSchedulesWithValidation parses YAML content and extracts cron schedules
// Returns list of cron schedule strings and error if YAML parsing fails
func extractSchedulesWithValidation(content []byte) ([]string, error) {
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	schedules := extractSchedules(workflow)
	return schedules, nil
}

// extractSchedules extracts cron schedules from parsed workflow YAML
// Returns list of cron schedule strings found in workflow triggers
func extractSchedules(workflow map[string]interface{}) []string {
	var schedules []string
	if on, ok := workflow["on"].(map[string]interface{}); ok {
		if schedule, ok := on["schedule"].([]interface{}); ok {
			for _, s := range schedule {
				if cronMap, ok := s.(map[string]interface{}); ok {
					if cron, ok := cronMap["cron"].(string); ok {
						schedules = append(schedules, cron)
					}
				}
			}
		}
	}
	return schedules
}

// GitHubClientAdapter wraps the github.Client to implement GitHubClient interface
type GitHubClientAdapter struct {
	client *github.Client
}

// InitializeGitHubClient creates a new GitHub client with the given token and base URL
func InitializeGitHubClient(token, baseURL string) GitHubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	client, err := github.NewEnterpriseClient(baseURL, baseURL, tc)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}
	return &GitHubClientAdapter{client: client}
}

func (a *GitHubClientAdapter) ListByOrg(ctx context.Context, org string, opts *github.RepositoryListByOrgOptions) ([]*github.Repository, *github.Response, error) {
	return a.client.Repositories.ListByOrg(ctx, org, opts)
}

func (a *GitHubClientAdapter) ListWorkflows(ctx context.Context, owner, repo string, opts *github.ListOptions) (*github.Workflows, *github.Response, error) {
	return a.client.Actions.ListWorkflows(ctx, owner, repo, opts)
}

func (a *GitHubClientAdapter) GetWorkflow(ctx context.Context, owner, repo string, workflowID int64) (*github.Workflow, *github.Response, error) {
	return a.client.Actions.GetWorkflowByID(ctx, owner, repo, workflowID)
}

func (a *GitHubClientAdapter) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	return a.client.Repositories.GetContents(ctx, owner, repo, path, opts)
}

func (a *GitHubClientAdapter) ListWorkflowRuns(ctx context.Context, owner, repo string, workflowID int64, opts *github.ListWorkflowRunsOptions) (*github.WorkflowRuns, *github.Response, error) {
	return a.client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, opts)
}

func (a *GitHubClientAdapter) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	return a.client.Repositories.ListCommits(ctx, owner, repo, opts)
}
