package scanner

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/sirupsen/logrus"
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
	GetUser(ctx context.Context, username string) (*github.User, *github.Response, error)
}

type Scanner struct {
	client          GitHubClient
	concurrentScans int
	excludedRepos   map[string]struct{}
}

// NewScanner는 새로운 Scanner 인스턴스를 생성합니다
func NewScanner(client GitHubClient, concurrentScans int) *Scanner {
	excludedMap := make(map[string]struct{})
	configFilePath := "/etc/gss/exclude-repos.txt"

	if _, err := os.Stat(configFilePath); err == nil {
		content, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to read exclude config file: %s", configFilePath)
		} else {
			repoNames := strings.Split(string(content), "\n")
			for _, name := range repoNames {
				trimmedName := strings.TrimSpace(name)
				if trimmedName != "" {
					excludedMap[trimmedName] = struct{}{}
					logrus.WithFields(logrus.Fields{
						"repository": trimmedName,
						"source":     "file",
					}).Info("Will exclude repository from scan")
				}
			}
		}
	} else if !os.IsNotExist(err) {
		logrus.WithError(err).Warnf("Error checking exclude config file: %s", configFilePath)
	}

	return &Scanner{
		client:          client,
		concurrentScans: concurrentScans,
		excludedRepos:   excludedMap,
	}
}

// ScanScheduledWorkflows scans all repositories in an organization for scheduled workflows
// Returns scan results containing workflow information and total repository count
// May return error if repository listing or scanning fails
func (s *Scanner) ScanScheduledWorkflows(org string) (*models.ScanResult, error) {
	start := time.Now()
	maxRoutines := int32(0)
	activeRoutines := int32(0)
	totalRepos := 0

	logrus.WithFields(logrus.Fields{
		"organization":  org,
		"maxConcurrent": s.concurrentScans,
	}).Info("Starting workflow scan")

	ctx := context.Background()
	var results []models.WorkflowInfo
	var resultMutex sync.Mutex
	var scanErrors []error
	var errorMutex sync.Mutex

	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "sources",
		Sort:        "full_name",
		Direction:   "asc",
	}

	for {
		repos, resp, err := s.client.ListByOrg(ctx, org, opts)
		if err != nil {
			logrus.WithError(err).Error("Failed to list repositories")
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		totalRepos += len(repos)

		logrus.WithFields(logrus.Fields{
			"repoCount": len(repos),
			"page":      opts.Page,
		}).Debug("Processing repositories batch")

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, s.concurrentScans)

		for _, repo := range repos {
			if _, ok := s.excludedRepos[*repo.Name]; ok {
				logrus.WithField("repository", *repo.Name).Info("Skipping excluded repository")
				continue
			}

			wg.Add(1)
			semaphore <- struct{}{}

			go func(repo *github.Repository) {
				defer wg.Done()
				defer func() { <-semaphore }()

				routines := atomic.AddInt32(&activeRoutines, 1)
				atomic.StoreInt32(&maxRoutines, max(atomic.LoadInt32(&maxRoutines), routines))

				logrus.WithFields(logrus.Fields{
					"repository":     *repo.Name,
					"activeRoutines": routines,
				}).Debug("Scanning repository")

				workflows, err := s.scanRepository(ctx, repo)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"repository": *repo.Name,
						"error":      err,
					}).Error("Failed to scan repository")

					errorMutex.Lock()
					scanErrors = append(scanErrors, fmt.Errorf("failed to scan repository %s: %w", *repo.Name, err))
					errorMutex.Unlock()
					return
				}

				if len(workflows) > 0 {
					logrus.WithFields(logrus.Fields{
						"repository":    *repo.Name,
						"workflowCount": len(workflows),
					}).Info("Found scheduled workflows")

					resultMutex.Lock()
					results = append(results, workflows...)
					resultMutex.Unlock()
				}

				atomic.AddInt32(&activeRoutines, -1)
			}(repo)
		}

		wg.Wait()

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	duration := time.Since(start)
	logrus.WithFields(logrus.Fields{
		"duration":       duration,
		"maxConcurrent":  maxRoutines,
		"totalWorkflows": len(results),
		"errorCount":     len(scanErrors),
	}).Info("Scan completed")

	if len(scanErrors) > 0 {
		logrus.WithField("errors", scanErrors).Warn("Some repositories failed to scan")
	}

	return &models.ScanResult{
		Workflows:          results,
		TotalRepos:         totalRepos,
		ScanDuration:       duration,
		MaxConcurrentScans: atomic.LoadInt32(&maxRoutines),
	}, nil
}

// scanRepository scans a single repository for workflow files and their schedules
// Extracts cron schedules and last run status for each workflow
// Updates results slice through mutex for thread safety
func (s *Scanner) scanRepository(ctx context.Context, repo *github.Repository) ([]models.WorkflowInfo, error) {
	logrus.WithField("repository", *repo.Name).Debug("Starting repository scan")

	workflows, _, err := s.client.ListWorkflows(ctx, *repo.Owner.Login, *repo.Name, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	var scheduledWorkflows []models.WorkflowInfo

	for _, workflow := range workflows.Workflows {
		logrus.WithFields(logrus.Fields{
			"repository": *repo.Name,
			"workflow":   *workflow.Name,
		}).Debug("Checking workflow")

		if isScheduled, schedule := s.checkWorkflowSchedule(ctx, repo, workflow); isScheduled {
			logrus.WithFields(logrus.Fields{
				"repository": *repo.Name,
				"workflow":   *workflow.Name,
				"schedule":   schedule,
			}).Info("Found scheduled workflow")

			// 워크플로우 실행 이력 가져오기
			runs, _, err := s.client.ListWorkflowRuns(ctx, *repo.Owner.Login, *repo.Name, *workflow.ID, &github.ListWorkflowRunsOptions{
				ListOptions: github.ListOptions{
					PerPage: 1, // 최신 실행 하나만 가져오기
				},
			})
			if err != nil {
				logrus.WithError(err).Warn("Failed to get workflow runs")
				// 에러가 나도 계속 진행
			}

			var lastStatus string
			if runs != nil && len(runs.WorkflowRuns) > 0 {
				lastStatus = *runs.WorkflowRuns[0].Status
			}

			// 마지막 커미터 정보 가져오기
			commits, _, err := s.client.ListCommits(ctx, *repo.Owner.Login, *repo.Name, &github.CommitsListOptions{
				Path: workflow.GetPath(),
				ListOptions: github.ListOptions{
					PerPage: 1, // 최신 커밋 하나만 가져오기
				},
			})
			if err != nil {
				logrus.WithError(err).Warn("Failed to get commits")
				// 에러가 나도 계속 진행
			}

			var lastCommitter string
			var isActiveUser bool
			if len(commits) > 0 && commits[0].Author != nil {
				if commits[0].Author.Login != nil {
					user, _, err := s.client.GetUser(ctx, *commits[0].Author.Login)
					if err != nil {
						// API 에러가 발생한 경우, 커밋의 author name은 표시하되 Inactive로 표시
						lastCommitter = commits[0].Commit.GetAuthor().GetName()
						isActiveUser = false
					} else if user == nil {
						// 사용자가 존재하지 않는 경우 (탈퇴 등)
						lastCommitter = commits[0].Commit.GetAuthor().GetName()
						isActiveUser = false
					} else {
						lastCommitter = commits[0].Commit.GetAuthor().GetName()
						isActiveUser = true
					}
				} else if commits[0].Commit.GetAuthor() != nil {
					// Author.Login은 없지만 커밋의 author 정보는 있는 경우
					lastCommitter = commits[0].Commit.GetAuthor().GetName()
					isActiveUser = false
				} else {
					lastCommitter = "Unknown"
					isActiveUser = false
				}
			} else {
				lastCommitter = "Unknown"
				isActiveUser = false
			}

			workflowInfo := models.WorkflowInfo{
				RepoName:         *repo.Name,
				WorkflowName:     *workflow.Name,
				WorkflowID:       *workflow.ID,
				WorkflowFileName: *workflow.Path,
				CronSchedules:    []string{schedule},
				LastStatus:       lastStatus,
				LastCommitter:    lastCommitter,
				IsActiveUser:     isActiveUser,
			}

			scheduledWorkflows = append(scheduledWorkflows, workflowInfo)
		}
	}

	return scheduledWorkflows, nil
}

// checkWorkflowSchedule checks if a workflow is scheduled based on its cron schedule
// Returns true if the workflow is scheduled and its cron schedule
func (s *Scanner) checkWorkflowSchedule(ctx context.Context, repo *github.Repository, workflow *github.Workflow) (bool, string) {
	content, _, _, err := s.client.GetContents(ctx, *repo.Owner.Login, *repo.Name, workflow.GetPath(), nil)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get workflow content: %s", workflow.GetPath())
		return false, ""
	}

	// GetContent()로 콘텐츠 가져오기
	contentStr, err := content.GetContent()
	if err != nil {
		logrus.WithError(err).Error("Failed to get content string")
		return false, ""
	}

	// base64 디코딩 시도 제거
	// YAML 직접 파싱
	var workflowData map[string]interface{}
	if err := yaml.Unmarshal([]byte(contentStr), &workflowData); err != nil {
		logrus.WithError(err).Errorf("Failed to parse workflow %s", workflow.GetPath())
		return false, ""
	}

	schedules, err := extractSchedulesWithValidation([]byte(contentStr))
	if err != nil || len(schedules) == 0 {
		logrus.WithField("path", workflow.GetPath()).Debug("No schedules found")
		return false, ""
	}

	return true, schedules[0]
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

func (a *GitHubClientAdapter) GetUser(ctx context.Context, username string) (*github.User, *github.Response, error) {
	return a.client.Users.Get(ctx, username)
}
