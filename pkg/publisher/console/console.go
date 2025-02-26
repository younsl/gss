package console

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
	"github.com/younsl/ghes-schedule-scanner/pkg/reporter"
)

// ConsolePublisher는 스캔 결과를 콘솔에 출력하는 Publisher입니다
type ConsolePublisher struct {
	formatter reporter.ReportFormatter
}

// NewConsolePublisher는 새로운 ConsolePublisher 인스턴스를 생성합니다
func NewConsolePublisher() *ConsolePublisher {
	return &ConsolePublisher{
		formatter: &reporter.ConsoleFormatter{},
	}
}

// PublishScanResult는 스캔 결과를 콘솔에 출력합니다
func (c *ConsolePublisher) PublishScanResult(result *models.ScanResult) error {
	logrus.Info("Publishing scan results to console")

	if result == nil {
		return fmt.Errorf("cannot publish nil scan result")
	}

	formatted := c.formatter.FormatReport(result)
	fmt.Print(formatted)

	logrus.Info("Successfully published scan results to console")
	return nil
}

// GetName은 Publisher의 이름을 반환합니다
func (c *ConsolePublisher) GetName() string {
	return "console"
}
