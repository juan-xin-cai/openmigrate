package pack

import (
	"os"
	"strings"
	"testing"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestCopyStrippedJSONFallsBackForNonJSON(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	logHome := t.TempDir()

	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", logHome); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer os.Setenv("HOME", oldHome)

	sourcePath := source + "/config.json"
	original := []byte("not-json")
	if err := os.WriteFile(sourcePath, original, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	logger, err := omlog.New(nil)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer logger.Close()

	targetPath := target + "/config.json"
	err = copyStrippedJSON(types.FileEntry{
		SourcePath:   sourcePath,
		RelativePath: "config.json",
		Mode:         0o644,
		FieldStripRules: []types.FieldStripRule{
			{Type: types.FieldStripRulePrefix, Value: "oauth:"},
		},
	}, targetPath, logger)
	if err != nil {
		t.Fatalf("copy stripped json: %v", err)
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("target content = %q", got)
	}

	logData, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logData), "field strip skipped for non-json entry") {
		t.Fatalf("expected non-json warning, got %s", logData)
	}
}
