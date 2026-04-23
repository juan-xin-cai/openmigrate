package whitelist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestLoadPrefersExternalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-code-v2.json")
	data := `{"agent":"claude-code","version":"v2","roots":["custom"],"entries":[{"path":"custom","strategy":"include"}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write external config: %v", err)
	}
	old := os.Getenv("OPENMIGRATE_WHITELIST_DIR")
	if err := os.Setenv("OPENMIGRATE_WHITELIST_DIR", dir); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer os.Setenv("OPENMIGRATE_WHITELIST_DIR", old)

	cfg, err := Load("claude-code", "v2")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Roots) != 1 || cfg.Roots[0] != "custom" {
		t.Fatalf("roots = %#v", cfg.Roots)
	}
}

func TestLoadFallsBackToEmbeddedConfig(t *testing.T) {
	old := os.Getenv("OPENMIGRATE_WHITELIST_DIR")
	if err := os.Setenv("OPENMIGRATE_WHITELIST_DIR", t.TempDir()); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer os.Setenv("OPENMIGRATE_WHITELIST_DIR", old)

	cfg, err := Load("claude-code", "v2")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Roots) == 0 || cfg.Roots[0] != ".claude.json" {
		t.Fatalf("unexpected roots = %#v", cfg.Roots)
	}
}

func TestLoadClaudeDesktopConfig(t *testing.T) {
	cfg, err := Load("claude-desktop", "v1")
	if err != nil {
		t.Fatalf("load desktop config: %v", err)
	}
	if len(cfg.Entries) != 12 {
		t.Fatalf("desktop entries = %d", len(cfg.Entries))
	}
	var configEntry, preferencesEntry types.WhitelistEntry
	var excludeCount int
	for _, entry := range cfg.Entries {
		switch entry.Path {
		case "Library/Application Support/Claude/config.json":
			configEntry = entry
		case "Library/Application Support/Claude/Preferences":
			preferencesEntry = entry
		}
		if entry.Strategy == types.StrategyExclude {
			excludeCount++
		}
	}
	if len(configEntry.FieldStripRules) != 2 {
		t.Fatalf("config field strip rules = %#v", configEntry.FieldStripRules)
	}
	if len(preferencesEntry.FieldStripRules) != 2 {
		t.Fatalf("preferences field strip rules = %#v", preferencesEntry.FieldStripRules)
	}
	if excludeCount != 4 {
		t.Fatalf("exclude count = %d", excludeCount)
	}
}

func TestMatchSupportsRecursivePatterns(t *testing.T) {
	if !Match("Library/Application Support/Claude/local-agent-mode-sessions/a/.claude/.claude.json", "Library/Application Support/Claude/local-agent-mode-sessions/**/.claude/.claude.json") {
		t.Fatalf("recursive match failed")
	}
	if !Match("Library/Application Support/Claude/claude-code-sessions/foo", "Library/Application Support/Claude/claude-code-sessions/") {
		t.Fatalf("directory prefix match failed")
	}
}
