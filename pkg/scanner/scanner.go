package scanner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

// Scanner handles GitHub Enterprise Server workflow scanning operations
type Scanner struct {
	client          *github.Client
	concurrentScans int
}

// NewScanner creates a new Scanner instance with the given token, base URL and concurrent scan limit
func NewScanner(token string, baseURL string, concurrentScans int) (*Scanner, error) {
	client := initializeGitHubClient(token, baseURL)
	scanner := &Scanner{
		client:          client,
		concurrentScans: concurrentScans,
	}

	// Validate token permissions
	if err := scanner.ValidateToken(context.Background()); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return scanner, nil
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
		repos, resp, err := s.client.Repositories.ListByOrg(ctx, org, opts)
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
		return nil, fmt.Errorf("encountered %d errors during scan", len(scanErrors))
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
	workflows, _, err := s.client.Actions.ListWorkflows(ctx, org, repo.GetName(), nil)
	if err != nil {
		return fmt.Errorf("failed to list workflows for %s: %w", repo.GetName(), err)
	}

	for _, workflow := range workflows.Workflows {
		fileContent, _, _, err := s.client.Repositories.GetContents(ctx, org, repo.GetName(), workflow.GetPath(), nil)
		if err != nil {
			log.Printf("Failed to get contents for %s: %v", workflow.GetPath(), err)
			continue
		}

		content, err := fileContent.GetContent()
		if err != nil {
			log.Printf("Failed to decode content for %s: %v", workflow.GetPath(), err)
			continue
		}

		schedules, err := extractSchedulesWithValidation([]byte(content))
		if err != nil || len(schedules) == 0 {
			log.Printf("No schedules found in %s: %v", workflow.GetPath(), err)
			continue
		}

		runs, _, err := s.client.Actions.ListWorkflowRunsByID(ctx, org, repo.GetName(), workflow.GetID(), &github.ListWorkflowRunsOptions{
			ListOptions: github.ListOptions{PerPage: 1},
		})
		if err != nil {
			return fmt.Errorf("failed to get workflow runs for %s: %w", workflow.GetName(), err)
		}

		lastStatus := "Unknown"
		if len(runs.WorkflowRuns) > 0 {
			lastStatus = runs.WorkflowRuns[0].GetStatus()
		}

		resultMutex.Lock()
		*results = append(*results, models.WorkflowInfo{
			RepoName:      repo.GetName(),
			WorkflowName:  workflow.GetName(),
			WorkflowID:    workflow.GetID(),
			CronSchedules: schedules,
			LastStatus:    lastStatus,
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

// initializeGitHubClient creates a new GitHub Enterprise client with given token and base URL
// Configures OAuth2 authentication and returns configured client
// Exits program if client creation fails
func initializeGitHubClient(token, baseURL string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	client, err := github.NewEnterpriseClient(baseURL, baseURL, tc)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}
	return client
}

// ValidateToken checks if the provided GitHub PAT is valid and has necessary permissions
func (s *Scanner) ValidateToken(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Step 1: Verify token is valid
	user, resp, err := s.client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			if ghErr, ok := err.(*github.ErrorResponse); ok {
				if strings.Contains(ghErr.Message, "Bad credentials") {
					return fmt.Errorf("token has expired or is invalid: %w", err)
				}
			}
			return fmt.Errorf("authentication failed: %w", err)
		}
		return fmt.Errorf("failed to validate token: %w", err)
	}

	// Step 2: Verify repo:* scope by attempting to list private repositories
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
		Type:      "private",
		Sort:      "updated",
		Direction: "desc",
	}

	_, resp, err = s.client.Repositories.List(ctx, user.GetLogin(), opts)
	if err != nil {
		if resp != nil && resp.StatusCode == 403 {
			return fmt.Errorf("token lacks required repository access permissions (repo:*): %w", err)
		}
		if resp != nil && resp.StatusCode == 404 {
			return fmt.Errorf("user or organization not found: %w", err)
		}
		return fmt.Errorf("failed to validate repository permissions: %w", err)
	}

	return nil
}
