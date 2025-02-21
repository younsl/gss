package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLoggerInitialization(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantLevel logrus.Level
		wantErr   bool
	}{
		{
			name:      "debug level",
			level:     "DEBUG",
			wantLevel: logrus.DebugLevel,
			wantErr:   false,
		},
		{
			name:      "info level",
			level:     "INFO",
			wantLevel: logrus.InfoLevel,
			wantErr:   false,
		},
		{
			name:      "warn level",
			level:     "WARN",
			wantLevel: logrus.WarnLevel,
			wantErr:   false,
		},
		{
			name:      "error level",
			level:     "ERROR",
			wantLevel: logrus.ErrorLevel,
			wantErr:   false,
		},
		{
			name:      "invalid level",
			level:     "INVALID",
			wantLevel: logrus.InfoLevel,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitLogger(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitLogger() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && log.GetLevel() != tt.wantLevel {
				t.Errorf("InitLogger() level = %v, want %v", log.GetLevel(), tt.wantLevel)
			}
		})
	}
}

func TestLogOutput(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{})

	testMessage := "test log message"
	Info(testMessage)

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if msg, ok := output["msg"].(string); !ok || msg != testMessage {
		t.Errorf("Log message = %v, want %v", msg, testMessage)
	}

	if level, ok := output["level"].(string); !ok || level != "info" {
		t.Errorf("Log level = %v, want info", level)
	}
}

func TestLogWithFields(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.JSONFormatter{})

	testFields := logrus.Fields{
		"key1": "value1",
		"key2": 123,
	}

	WithFields(testFields).Info("test message")

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if value, ok := output["key1"].(string); !ok || value != "value1" {
		t.Errorf("Field key1 = %v, want value1", value)
	}

	if value, ok := output["key2"].(float64); !ok || int(value) != 123 {
		t.Errorf("Field key2 = %v, want 123", value)
	}
}
