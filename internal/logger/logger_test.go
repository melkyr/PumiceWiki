//go:build unit

package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"go-wiki-app/internal/config"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("console format", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := config.LogConfig{Level: "info", Format: "console"}
		log := New(cfg, &buf)

		log.Info("hello world")

		output := buf.String()
		if !strings.Contains(output, "hello world") {
			t.Errorf("expected log output to contain 'hello world', but got '%s'", output)
		}
		if strings.Contains(output, "{") {
			t.Errorf("expected console format, but got json-like output: %s", output)
		}
	})

	t.Run("json format", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := config.LogConfig{Level: "error", Format: "json"}
		log := New(cfg, &buf)

		testErr := errors.New("test error")
		log.Error(testErr, "an error occurred")

		output := buf.String()
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Fatalf("failed to unmarshal log output as json: %v\noutput: %s", err, output)
		}

		if logEntry["level"] != "error" {
			t.Errorf("expected log level 'error', got '%v'", logEntry["level"])
		}
		if logEntry["message"] != "an error occurred" {
			t.Errorf("expected message 'an error occurred', got '%v'", logEntry["message"])
		}
		if logEntry["error"] != "test error" {
			t.Errorf("expected error 'test error', got '%v'", logEntry["error"])
		}
	})

	t.Run("log level filtering", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := config.LogConfig{Level: "warn", Format: "console"}
		log := New(cfg, &buf)

		log.Info("this should be ignored")
		log.Warn("this should appear")

		output := buf.String()
		if strings.Contains(output, "this should be ignored") {
			t.Error("info level log should have been ignored")
		}
		if !strings.Contains(output, "this should appear") {
			t.Error("warn level log should have appeared")
		}
	})
}
