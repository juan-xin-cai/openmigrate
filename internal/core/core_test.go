package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestBuildSuggestedMappingPrefillsExistingTargets(t *testing.T) {
	targetHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetHome, "projects", "claudecode"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	scan := types.PathScanResult{
		HomePrefix: "/Users/feifei",
		ProjectRoots: []string{
			"/Users/feifei/projects/claudecode",
			"/Users/feifei/projects/missing",
		},
	}

	got := buildSuggestedMapping(scan, targetHome)
	if len(got.ProjectMappings) != 2 {
		t.Fatalf("project mappings = %#v", got.ProjectMappings)
	}
	wantTo := filepath.Join(targetHome, "projects", "claudecode")
	if got.ProjectMappings[0].To != wantTo {
		t.Errorf("existing target: To = %q, want %q", got.ProjectMappings[0].To, wantTo)
	}
	if got.ProjectMappings[1].To != "" {
		t.Errorf("missing target: To = %q, want empty", got.ProjectMappings[1].To)
	}
}

func TestBuildSuggestedMappingSkipsWhenSourceHomeMissing(t *testing.T) {
	targetHome := t.TempDir()
	scan := types.PathScanResult{
		ProjectRoots: []string{"/Users/feifei/projects/claudecode"},
	}
	got := buildSuggestedMapping(scan, targetHome)
	if len(got.ProjectMappings) != 1 || got.ProjectMappings[0].To != "" {
		t.Fatalf("project mappings = %#v", got.ProjectMappings)
	}
}
