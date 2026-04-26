package logging_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beastixq/marketplace/internal/config"
	"github.com/beastixq/marketplace/internal/logging"
)

func TestNew_FileSink_AppendsJSON(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nested", "app.log")

	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		File:   logPath,
	}

	logger, closer, err := logging.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	logger.Info("hello", "user_id", int64(42))
	if err := closer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	var rec map[string]any
	if err := json.Unmarshal(raw[:len(raw)-1], &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["msg"] != "hello" {
		t.Errorf("msg: got %v", rec["msg"])
	}
	if rec["user_id"].(float64) != 42 {
		t.Errorf("user_id: got %v", rec["user_id"])
	}
	if rec["level"] != "INFO" {
		t.Errorf("level: got %v", rec["level"])
	}
}

func TestNew_LevelFilter(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	cfg := config.LoggingConfig{Level: "warn", Format: "json", File: logPath}

	logger, closer, err := logging.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	logger.Debug("hidden")
	logger.Info("hidden")
	logger.Warn("visible")
	closer.Close()

	raw, _ := os.ReadFile(logPath)
	if strings.Contains(string(raw), "hidden") {
		t.Errorf("level filter failed: %s", raw)
	}
	if !strings.Contains(string(raw), "visible") {
		t.Errorf("warn record missing: %s", raw)
	}
}

func TestNew_AppendsAcrossOpens(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	cfg := config.LoggingConfig{Level: "info", Format: "json", File: logPath}

	for i := 0; i < 2; i++ {
		logger, closer, err := logging.New(cfg)
		if err != nil {
			t.Fatalf("New[%d]: %v", i, err)
		}
		logger.Info("entry")
		closer.Close()
	}

	raw, _ := os.ReadFile(logPath)
	if got := strings.Count(string(raw), "\"entry\""); got != 2 {
		t.Errorf("expected 2 entries (append), got %d. content:\n%s", got, raw)
	}
}

func TestNew_RejectsBadLevel(t *testing.T) {
	_, _, err := logging.New(config.LoggingConfig{Level: "trace", Format: "json", File: filepath.Join(t.TempDir(), "x.log")})
	if err == nil {
		t.Fatal("expected error on bad level")
	}
}

func TestNew_RejectsBadFormat(t *testing.T) {
	_, _, err := logging.New(config.LoggingConfig{Level: "info", Format: "xml", File: filepath.Join(t.TempDir(), "x.log")})
	if err == nil {
		t.Fatal("expected error on bad format")
	}
}

func TestNew_NoSinks(t *testing.T) {
	_, _, err := logging.New(config.LoggingConfig{Level: "info", Format: "json"})
	if err == nil {
		t.Fatal("expected error when neither file nor console set")
	}
}
