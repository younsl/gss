package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

func TestConsolePublisher_PublishScanResult(t *testing.T) {
	publisher := NewConsolePublisher()

	// 테스트 데이터 생성
	result := &models.ScanResult{
		TotalRepos: 2,
		Workflows: []models.WorkflowInfo{
			{
				RepoName:      "test-repo-1",
				WorkflowName:  "workflow1.yml",
				CronSchedules: []string{"0 15 * * *"},
				LastStatus:    "success",
				LastCommitter: "user1",
			},
		},
	}

	// 출력 테스트
	err := publisher.PublishScanResult(result)
	assert.NoError(t, err)

	// nil 결과 테스트
	err = publisher.PublishScanResult(nil)
	assert.Error(t, err)

	// GetName 테스트
	assert.Equal(t, "console", publisher.GetName())
}
