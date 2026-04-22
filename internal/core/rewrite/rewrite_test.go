package rewrite

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestRewriteTreeRewritesJSONLDirAndSkipsBinary(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, ".claude", "projects", "-Users-roy-projects-foo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	jsonlPath := filepath.Join(projectDir, "history.jsonl")
	line := "{\"cwd\":\"/Users/roy/projects/foo\",\"tool\":\"/opt/tools/rg\"}\n"
	if err := ioutil.WriteFile(jsonlPath, []byte(line), 0o644); err != nil {
		t.Fatalf("write jsonl: %v", err)
	}
	binPath := filepath.Join(projectDir, "db.sqlite")
	if err := ioutil.WriteFile(binPath, []byte{0x00, 0x01}, 0o644); err != nil {
		t.Fatalf("write bin: %v", err)
	}

	logger := omlog.MustLogger(nil)
	defer logger.Close()
	report, err := RewriteTree(root, types.PathMapping{
		SourceHome: "/Users/roy",
		TargetHome: "/Users/alice",
		ProjectMappings: []types.PathPair{
			{From: "/Users/roy/projects/foo", To: "/Users/alice/work/foo"},
		},
	}, types.PathScanResult{
		ProjectRoots:  []string{"/Users/roy/projects/foo"},
		ExternalPaths: []string{"/opt/tools/rg"},
	}, logger)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	renamed := filepath.Join(root, ".claude", "projects", "-Users-alice-work-foo", "history.jsonl")
	data, err := ioutil.ReadFile(renamed)
	if err != nil {
		t.Fatalf("read rewritten: %v", err)
	}
	if got := string(data); got != "{\"cwd\":\"/Users/alice/work/foo\",\"tool\":\"/opt/tools/rg\"}\n" {
		t.Fatalf("rewritten jsonl = %q", got)
	}
	if report.RewrittenFiles != 1 {
		t.Fatalf("rewritten files = %d", report.RewrittenFiles)
	}
	if len(report.SkippedBinary) != 1 || report.SkippedBinary[0] != ".claude/projects/-Users-roy-projects-foo/db.sqlite" {
		t.Fatalf("skipped binary = %#v", report.SkippedBinary)
	}
	if len(report.ProjectRoots) != 1 || report.ProjectRoots[0] != "/Users/alice/work/foo" {
		t.Fatalf("project roots = %#v", report.ProjectRoots)
	}
}
