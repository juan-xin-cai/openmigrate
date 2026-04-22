package pathscan

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestScanClassifiesL1L2L3(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "history.jsonl")
	content := "/Users/roy/projects/foo/app\n/Users/roy/projects/foo/README.md\n/Users/roy/projects/bar/main.go\n/opt/homebrew/bin/rg\n"
	if err := ioutil.WriteFile(textPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write text: %v", err)
	}
	binPath := filepath.Join(root, "data.db")
	if err := ioutil.WriteFile(binPath, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("write bin: %v", err)
	}

	manifest := types.Manifest{
		Entries: []types.FileEntry{
			{SourcePath: textPath, RelativePath: ".claude/history.jsonl"},
			{SourcePath: binPath, RelativePath: ".claude/cache.db"},
		},
	}
	got, err := Scan(manifest)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if got.HomePrefix != "/Users/roy" {
		t.Fatalf("home prefix = %q", got.HomePrefix)
	}
	if len(got.ProjectRoots) != 2 || got.ProjectRoots[0] != "/Users/roy/projects/foo" || got.ProjectRoots[1] != "/Users/roy/projects/bar" {
		t.Fatalf("project roots = %#v", got.ProjectRoots)
	}
	if len(got.ExternalPaths) != 1 || got.ExternalPaths[0] != "/opt/homebrew/bin/rg" {
		t.Fatalf("external paths = %#v", got.ExternalPaths)
	}
}

func TestScanEmptyManifest(t *testing.T) {
	got, err := Scan(types.Manifest{})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if got.HomePrefix != "" || len(got.ProjectRoots) != 0 || len(got.ExternalPaths) != 0 {
		t.Fatalf("unexpected result: %#v", got)
	}
}
