package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/younsl/ghes-schedule-scanner/pkg/models"
	"github.com/younsl/ghes-schedule-scanner/pkg/version"
)

type ReportFormatter interface {
	FormatReport(*models.ScanResult) string
}

type ConsoleFormatter struct{}

type Reporter struct {
	formatter ReportFormatter
}

func NewReporter(formatter ReportFormatter) *Reporter {
	logrus.Debug("Initializing new reporter")
	return &Reporter{formatter: formatter}
}

func (r *Reporter) GenerateReport(result *models.ScanResult) error {
	output, err := r.FormatResults(result)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate report")
		return fmt.Errorf("formatting error: %w", err)
	}
	fmt.Print(output)
	return nil
}

func (r *Reporter) FormatResults(result *models.ScanResult) (string, error) {
	logrus.WithFields(logrus.Fields{
		"workflowCount": len(result.Workflows),
		"scanDuration":  result.ScanDuration,
	}).Debug("Starting to format scan results")

	if result == nil {
		logrus.Error("Received nil scan result")
		return "", fmt.Errorf("cannot format nil result")
	}

	formatted := r.formatter.FormatReport(result)
	logrus.WithField("formattedLength", len(formatted)).Debug("Successfully formatted results")
	return formatted, nil
}

func (f *ConsoleFormatter) FormatReport(result *models.ScanResult) string {
	logrus.Debug("Formatting results for console output")

	if result == nil {
		logrus.Error("Received nil result in console formatter")
		return "No workflows found\n"
	}

	buildInfo := version.Get()

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("GSS Build: Version=%s, Commit=%s, Built=%s, Go=%s\n\n",
		buildInfo.Version, buildInfo.GitCommit, buildInfo.BuildDate, buildInfo.GoVersion))

	sb.WriteString("Scheduled Workflows Summary:\n")
	sb.WriteString(fmt.Sprintf("%-3s %-35s %-35s %-13s %-13s %-15s %s\n",
		"NO", "REPOSITORY", "WORKFLOW", "UTC SCHEDULE", "KST SCHEDULE", "LAST COMMITTER", "LAST STATUS"))

	for i, wf := range result.Workflows {
		for _, schedule := range wf.CronSchedules {
			sb.WriteString(fmt.Sprintf("%-3d %-35s %-35s %-13s %-13s %-15s %s\n",
				i+1,
				truncateString(wf.RepoName, 35),
				truncateString(wf.WorkflowName, 35),
				schedule,
				convertToKST(schedule),
				truncateString(wf.LastCommitter, 15),
				wf.LastStatus,
			))
		}
	}

	sb.WriteString(fmt.Sprintf("\nTotal Repositories: %d | Excluded Repositories: %d | Scheduled Workflows Found: %d\n",
		result.TotalRepos,
		result.ExcludedReposCount,
		len(result.Workflows)))
	return sb.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s + strings.Repeat(" ", maxLen-len(s))
	}
	return s[:maxLen-2] + ".."
}

func convertToKST(utcCron string) string {
	parts := strings.Split(utcCron, " ")
	if len(parts) != 5 {
		return utcCron
	}

	if !isValidCronExpression(parts) {
		return utcCron
	}

	minute := parts[0]
	hour := atoi(parts[1])
	dom := parts[2]
	month := parts[3]
	dow := parts[4]

	newHour := (hour + 9) % 24
	dayShift := (hour + 9) / 24

	if dayShift > 0 && dow != "*" {
		dowNum := atoi(dow)
		if dowNum >= 0 && dowNum <= 6 {
			newDow := (dowNum + 1) % 7
			dow = fmt.Sprintf("%d", newDow)
		}
	}

	return fmt.Sprintf("%s %d %s %s %s", minute, newHour, dom, month, dow)
}

func isValidCronExpression(parts []string) bool {
	if parts[0] != "*" && (atoi(parts[0]) < 0 || atoi(parts[0]) > 59) {
		return false
	}
	if parts[1] != "*" && (atoi(parts[1]) < 0 || atoi(parts[1]) > 23) {
		return false
	}
	if parts[4] != "*" && (atoi(parts[4]) < 0 || atoi(parts[4]) > 6) {
		return false
	}
	return true
}

func atoi(s string) int {
	if s == "*" {
		return 0
	}
	i := 0
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0
	}
	return i
}

func formatDuration(d time.Duration) string {
	logrus.WithField("duration", d).Debug("Formatting duration")
	return d.Round(time.Millisecond).String()
}
