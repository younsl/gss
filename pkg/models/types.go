package models

import "time"

type WorkflowInfo struct {
	RepoName         string
	WorkflowName     string
	WorkflowID       int64
	WorkflowFileName string
	CronSchedules    []string
	LastStatus       string
	LastCommitter    string
	IsActiveUser     bool
}

type ScanResult struct {
	Workflows          []WorkflowInfo
	TotalRepos         int
	ExcludedReposCount int
	ScanDuration       time.Duration
	MaxConcurrentScans int32
}
