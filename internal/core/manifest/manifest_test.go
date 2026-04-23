package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/openmigrate/openmigrate/internal/core/whitelist"
)

func TestBuildAppliesScopeFiltersAndNoHistory(t *testing.T) {
	sourceHome := t.TempDir()
	writeFile(t, filepath.Join(sourceHome, ".claude.json"), `{"user":"roy"}`)
	writeFile(t, filepath.Join(sourceHome, ".claude", "skills", "one", "skill.md"), "skill")
	writeFile(t, filepath.Join(sourceHome, ".claude", "projects", "-Users-roy-projects-foo", "history.jsonl"), "{}\n")
	writeFile(t, filepath.Join(sourceHome, "Library", "Application Support", "Claude", "config.json"), `{"oauth:token":"abc","keep":true}`)
	writeFile(t, filepath.Join(sourceHome, "Library", "Application Support", "Claude", "vm_bundles", "huge.bin"), "skip")

	codeCfg, err := whitelist.Load("claude-code", "v2")
	if err != nil {
		t.Fatalf("load code cfg: %v", err)
	}
	desktopCfg, err := whitelist.Load("claude-desktop", "v1")
	if err != nil {
		t.Fatalf("load desktop cfg: %v", err)
	}

	manifest, err := Build(types.ManifestParams{
		SourceHome: sourceHome,
		OnlyScopes: []string{"settings"},
		NoHistory:  true,
	}, codeCfg, desktopCfg)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	got := map[string]types.FileEntry{}
	for _, entry := range manifest.Entries {
		got[entry.RelativePath] = entry
	}
	if _, ok := got[".claude/skills/one/skill.md"]; ok {
		t.Fatalf("skills entry should be filtered out")
	}
	if _, ok := got[".claude/projects/-Users-roy-projects-foo/history.jsonl"]; ok {
		t.Fatalf("history entry should be filtered out")
	}
	configEntry, ok := got["Library/Application Support/Claude/config.json"]
	if !ok {
		t.Fatalf("desktop config missing")
	}
	if len(configEntry.FieldStripRules) != 2 {
		t.Fatalf("field strip rules = %#v", configEntry.FieldStripRules)
	}
	if _, ok := got["Library/Application Support/Claude/vm_bundles/huge.bin"]; ok {
		t.Fatalf("excluded vm bundle should not exist")
	}
}

func TestBuildRejectsConflictingScopeFilters(t *testing.T) {
	_, err := Build(types.ManifestParams{
		SourceHome:    t.TempDir(),
		OnlyScopes:    []string{"settings"},
		ExcludeScopes: []string{"sessions"},
	})
	if err != types.ErrConflictingScopeFilter {
		t.Fatalf("err = %v", err)
	}
}

func TestBuildAppliesExcludeScopes(t *testing.T) {
	sourceHome := t.TempDir()
	writeFile(t, filepath.Join(sourceHome, ".claude.json"), `{"user":"roy"}`)
	writeFile(t, filepath.Join(sourceHome, "Library", "Application Support", "Claude", "claude-code-sessions", "session-1", "entry.json"), `{"id":"s1"}`)
	writeFile(t, filepath.Join(sourceHome, "Library", "Application Support", "Claude", "config.json"), `{"keep":true}`)

	codeCfg, err := whitelist.Load("claude-code", "v2")
	if err != nil {
		t.Fatalf("load code cfg: %v", err)
	}
	desktopCfg, err := whitelist.Load("claude-desktop", "v1")
	if err != nil {
		t.Fatalf("load desktop cfg: %v", err)
	}

	manifest, err := Build(types.ManifestParams{
		SourceHome:    sourceHome,
		ExcludeScopes: []string{"sessions"},
	}, codeCfg, desktopCfg)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	for _, entry := range manifest.Entries {
		if entry.RelativePath == "Library/Application Support/Claude/claude-code-sessions/session-1/entry.json" {
			t.Fatalf("desktop session entry should be filtered out")
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
