package conflict

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectClassifiesAddConflictAndTargetOnly(t *testing.T) {
	pkg := t.TempDir()
	target := t.TempDir()
	mustMkdirAll(t, filepath.Join(pkg, ".claude", "skills", "same"))
	mustMkdirAll(t, filepath.Join(pkg, ".claude", "skills", "new"))
	mustWriteFile(t, filepath.Join(pkg, ".claude", "skills", "same", "skill.md"), "pkg")
	mustWriteFile(t, filepath.Join(pkg, ".claude", "skills", "new", "skill.md"), "new")
	mustWriteFile(t, filepath.Join(pkg, ".claude", "settings.json"), `{"theme":"dark","hooks":"pkg"}`)

	mustMkdirAll(t, filepath.Join(target, ".claude", "skills", "same"))
	mustMkdirAll(t, filepath.Join(target, ".claude", "skills", "target-only"))
	mustWriteFile(t, filepath.Join(target, ".claude", "skills", "same", "skill.md"), "target")
	mustWriteFile(t, filepath.Join(target, ".claude", "skills", "target-only", "skill.md"), "old")
	mustWriteFile(t, filepath.Join(target, ".claude", "settings.json"), `{"theme":"light","keep":"me"}`)

	report, err := Detect(pkg, target)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	skills := report.Buckets["skills"]
	if len(skills.Additions) != 1 || skills.Additions[0].Key != ".claude/skills/new" {
		t.Fatalf("skill additions = %#v", skills.Additions)
	}
	if len(skills.Conflicts) != 1 || skills.Conflicts[0].Key != ".claude/skills/same" {
		t.Fatalf("skill conflicts = %#v", skills.Conflicts)
	}
	if len(skills.TargetOnly) != 1 || skills.TargetOnly[0].Key != ".claude/skills/target-only" {
		t.Fatalf("skill target-only = %#v", skills.TargetOnly)
	}

	settings := report.Buckets["settings"]
	if len(settings.Conflicts) != 1 || settings.Conflicts[0].Key != "settings:theme" {
		t.Fatalf("settings conflicts = %#v", settings.Conflicts)
	}
	if len(settings.Additions) != 1 || settings.Additions[0].Key != "settings:hooks" {
		t.Fatalf("settings additions = %#v", settings.Additions)
	}
	if len(settings.TargetOnly) != 1 || settings.TargetOnly[0].Key != "settings:keep" {
		t.Fatalf("settings target-only = %#v", settings.TargetOnly)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := ioutil.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
