package writer

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestWriteSuccessPreservesMode(t *testing.T) {
	stage := t.TempDir()
	targetHome := t.TempDir()
	source := filepath.Join(stage, ".claude", "commands", "tool.sh")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ioutil.WriteFile(source, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}

	written, updated, skipped, err := Write([]types.FileEntry{
		{SourcePath: source, RelativePath: ".claude/commands/tool.sh", Mode: 0o755, GroupKey: ".claude/commands/tool.sh"},
	}, targetHome, types.ConflictDecision{Actions: map[string]types.DecisionAction{}}, types.ConflictReport{}, nil)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(updated) != 0 || len(skipped) != 0 || len(written) != 1 || written[0] != ".claude/commands/tool.sh" {
		t.Fatalf("written=%#v updated=%#v skipped=%#v", written, updated, skipped)
	}
	info, err := os.Stat(filepath.Join(targetHome, ".claude", "commands", "tool.sh"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestWriteFailureKeepsTargetUnchanged(t *testing.T) {
	stage := t.TempDir()
	targetHome := t.TempDir()
	existing := filepath.Join(targetHome, ".claude.json")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := ioutil.WriteFile(existing, []byte("before"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if err := os.Chmod(targetHome, 0o555); err != nil {
		t.Fatalf("chmod target home: %v", err)
	}
	defer os.Chmod(targetHome, 0o755)

	if err := os.MkdirAll(filepath.Join(stage, ".claude"), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	source := filepath.Join(stage, ".claude.json")
	if err := ioutil.WriteFile(source, []byte("after"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, _, _, err := Write([]types.FileEntry{
		{SourcePath: source, RelativePath: ".claude.json", Mode: 0o644, GroupKey: ".claude.json"},
	}, targetHome, types.ConflictDecision{Actions: map[string]types.DecisionAction{}}, types.ConflictReport{}, nil)
	if err == nil {
		t.Fatalf("expected write error")
	}
	data, readErr := ioutil.ReadFile(existing)
	if readErr != nil {
		t.Fatalf("read existing: %v", readErr)
	}
	if string(data) != "before" {
		t.Fatalf("target changed to %q", string(data))
	}
}

func TestWriteRejectsMissingConflictDecision(t *testing.T) {
	stage := t.TempDir()
	targetHome := t.TempDir()
	source := filepath.Join(stage, ".claude", "skills", "same", "skill.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := ioutil.WriteFile(source, []byte("package"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	existing := filepath.Join(targetHome, ".claude", "skills", "same", "skill.md")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := ioutil.WriteFile(existing, []byte("target"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	_, _, _, err := Write([]types.FileEntry{
		{SourcePath: source, RelativePath: ".claude/skills/same/skill.md", Mode: 0o644, GroupKey: ".claude/skills/same"},
	}, targetHome, types.ConflictDecision{Actions: map[string]types.DecisionAction{}}, types.ConflictReport{
		Buckets: map[string]types.ConflictBucket{
			"skills": {Conflicts: []types.ConflictItem{{Key: ".claude/skills/same", Type: "skills"}}},
		},
	}, nil)
	if !errors.Is(err, types.ErrConflictDecisionRequired) {
		t.Fatalf("expected ErrConflictDecisionRequired, got %v", err)
	}
}
