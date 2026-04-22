package whitelist

import (
	"os"
	"path/filepath"
	"testing"
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
