package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

type HTMLPublisher struct {
	outputPath    string
	templatePath  string
	organization  string
	githubBaseURL string
}

func NewHTMLPublisher(outputPath, templatePath, organization, githubBaseURL string) *HTMLPublisher {
	if templatePath == "" {
		// 기본 템플릿 사용
		templatePath = "templates/report.html"
	}

	return &HTMLPublisher{
		outputPath:    outputPath,
		templatePath:  templatePath,
		organization:  organization,
		githubBaseURL: githubBaseURL,
	}
}

func (p *HTMLPublisher) PublishScanResult(result *models.ScanResult) error {
	if p.outputPath == "" {
		return fmt.Errorf("invalid configuration: missing output path")
	}

	// 출력 디렉토리 생성
	err := os.MkdirAll(filepath.Dir(p.outputPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 템플릿 로드
	tmpl, err := template.ParseFiles(p.templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// 출력 파일 생성
	file, err := os.Create(p.outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// 템플릿 실행
	data := map[string]interface{}{
		"Result":        result,
		"Organization":  p.organization,
		"GitHubBaseURL": p.githubBaseURL,
		"GeneratedAt":   time.Now().Format(time.RFC1123),
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (p *HTMLPublisher) GetName() string {
	return "html"
}
