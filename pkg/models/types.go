package models

type WorkflowInfo struct {
	RepoName      string
	WorkflowName  string
	WorkflowID    int64
	CronSchedules []string
	LastStatus    string
	LastCommitter string
}

type ScanResult struct {
	Workflows  []WorkflowInfo
	TotalRepos int
}
