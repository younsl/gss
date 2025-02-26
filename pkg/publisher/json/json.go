package json

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/younsl/ghes-schedule-scanner/pkg/models"
)

type JSONPublisher struct {
	outputPath string // 빈 문자열이면 stdout으로 출력
}

func NewJSONPublisher(outputPath string) *JSONPublisher {
	return &JSONPublisher{
		outputPath: outputPath,
	}
}

func (p *JSONPublisher) PublishScanResult(result *models.ScanResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan result to JSON: %w", err)
	}

	if p.outputPath == "" {
		// stdout으로 출력
		fmt.Println(string(data))
		return nil
	}

	// 파일에 저장
	err = os.WriteFile(p.outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON to file: %w", err)
	}

	return nil
}

func (p *JSONPublisher) GetName() string {
	return "json"
}
