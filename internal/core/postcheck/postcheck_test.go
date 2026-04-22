package postcheck

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestCheckReportsMissingCommandsAndProjectRoots(t *testing.T) {
	targetHome := t.TempDir()
	settingsPath := filepath.Join(targetHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	settings := `{
  "hooks": {"pre": {"command": "/missing/hook"}},
  "mcp": {"servers": [{"command": "definitely-missing-binary"}]}
}`
	if err := ioutil.WriteFile(settingsPath, []byte(settings), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	report, err := Check(targetHome, types.RewriteReport{
		ProjectRoots:  []string{filepath.Join(targetHome, "no-project")},
		ExternalPaths: []string{"/missing/tool"},
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(report.Items) != 4 {
		t.Fatalf("items = %#v", report.Items)
	}
}
