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

func TestNormalizeImportMappingAllowsEmptyProjectMappingsWhenHomeKnown(t *testing.T) {
	scan := types.PathScanResult{
		HomePrefix:   "/Users/feifei",
		ProjectRoots: []string{"/Users/feifei/projects/foo", "/Users/feifei/projects/bar"},
	}
	got, err := normalizeImportMapping(types.PathMapping{TargetHome: "/Users/roy"}, "", scan)
	if err != nil {
		t.Fatalf("expected no error when home is known and user skipped all rows, got %v", err)
	}
	if len(got.ProjectMappings) != 0 {
		t.Fatalf("project mappings = %#v", got.ProjectMappings)
	}
	if got.SourceHome != "/Users/feifei" || got.TargetHome != "/Users/roy" {
		t.Fatalf("home prefixes = (%q, %q)", got.SourceHome, got.TargetHome)
	}
}

func TestNormalizeImportMappingDropsEmptyPairs(t *testing.T) {
	scan := types.PathScanResult{
		HomePrefix:   "/Users/feifei",
		ProjectRoots: []string{"/Users/feifei/projects/foo", "/Users/feifei/projects/bar"},
	}
	in := types.PathMapping{
		TargetHome: "/Users/roy",
		ProjectMappings: []types.PathPair{
			{From: "/Users/feifei/projects/foo", To: "/Users/roy/work/foo"},
			{From: "/Users/feifei/projects/bar", To: ""},
		},
	}
	got, err := normalizeImportMapping(in, "", scan)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(got.ProjectMappings) != 1 || got.ProjectMappings[0].From != "/Users/feifei/projects/foo" {
		t.Fatalf("project mappings = %#v", got.ProjectMappings)
	}
}

func TestNormalizeImportMappingErrorsWhenHomeAndMappingsMissing(t *testing.T) {
	scan := types.PathScanResult{
		ProjectRoots: []string{"/Users/feifei/projects/foo"},
	}
	if _, err := normalizeImportMapping(types.PathMapping{TargetHome: "/Users/roy"}, "", scan); err == nil {
		t.Fatalf("expected ErrPathMappingRequired when no home prefix is known")
	}
}
