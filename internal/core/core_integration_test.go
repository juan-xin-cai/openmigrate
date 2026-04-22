package core

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/pack"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestExportApplyImportAndRollbackNoConflict(t *testing.T) {
	sourceHome := t.TempDir()
	targetHome := t.TempDir()
	outputDir := t.TempDir()
	passphrase := "pass-1"

	setupClaudeSource(t, sourceHome)

	exportResult, err := Export(context.Background(), ExportParams{
		SourceHome: sourceHome,
		Agent:      "claude-code",
		Version:    "v2",
		OutputDir:  outputDir,
		Passphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if _, err := os.Stat(exportResult.PackagePath); err != nil {
		t.Fatalf("package missing: %v", err)
	}
	metaBytes, err := ioutil.ReadFile(exportResult.MetaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	var metaMap map[string]interface{}
	if err := json.Unmarshal(metaBytes, &metaMap); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if _, ok := metaMap["path_scan"]; !ok {
		t.Fatalf("meta missing path_scan")
	}
	if strings.Contains(string(metaBytes), "pass-1") {
		t.Fatalf("meta leaked passphrase")
	}

	result, err := ApplyImport(context.Background(), ImportApplyParams{
		PackagePath: exportResult.PackagePath,
		Passphrase:  passphrase,
		Mapping: types.PathMapping{
			SourceHome: "/Users/roy",
			TargetHome: targetHome,
			ProjectMappings: []types.PathPair{
				{From: "/Users/roy/projects/foo", To: filepath.Join(targetHome, "workspace", "foo")},
			},
		},
		Decisions: types.ConflictDecision{Actions: map[string]types.DecisionAction{}},
	})
	if err != nil {
		t.Fatalf("apply import: %v", err)
	}
	if len(result.Snapshot.Targets) == 0 {
		t.Fatalf("snapshot missing targets")
	}

	projectHistory := filepath.Join(targetHome, ".claude", "projects", "-Users-roy-projects-foo", "history.jsonl")
	renamedHistory := filepath.Join(targetHome, ".claude", "projects", strings.ReplaceAll(filepath.Join(targetHome, "workspace", "foo"), "/", "-"), "history.jsonl")
	if _, err := os.Stat(projectHistory); !os.IsNotExist(err) {
		t.Fatalf("old project path still exists: %v", err)
	}
	data, err := ioutil.ReadFile(renamedHistory)
	if err != nil {
		t.Fatalf("read renamed history: %v", err)
	}
	if !strings.Contains(string(data), filepath.Join(targetHome, "workspace", "foo")) {
		t.Fatalf("history not rewritten: %q", string(data))
	}
	info, err := os.Lstat(filepath.Join(targetHome, ".claude", "skills", "existing"))
	if err != nil {
		t.Fatalf("stat existing skill: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("existing skill not restored as symlink")
	}
	externalInfo, err := os.Lstat(filepath.Join(targetHome, ".claude", "skills", "external"))
	if err != nil {
		t.Fatalf("stat external skill: %v", err)
	}
	if externalInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("external skill should be实体")
	}

	if err := Rollback(context.Background(), RollbackParams{Snapshot: result.Snapshot, Passphrase: passphrase}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetHome, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("expected imported data to be removed after rollback, err=%v", err)
	}
}

func TestImportWrongPassphraseReturnsErrDecryptFailed(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	targetHome := t.TempDir()
	setupClaudeSource(t, sourceHome)

	exportResult, err := Export(context.Background(), ExportParams{
		SourceHome: sourceHome,
		Agent:      "claude-code",
		Version:    "v2",
		OutputDir:  outputDir,
		Passphrase: "pass-1",
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	_, err = Import(context.Background(), ImportParams{
		PackagePath: exportResult.PackagePath,
		Passphrase:  "wrong-pass",
		TargetHome:  targetHome,
	})
	if !errors.Is(err, types.ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
	entries, readErr := os.ReadDir(targetHome)
	if readErr != nil {
		t.Fatalf("read target home: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("target home should stay empty, got %d entries", len(entries))
	}
}

func TestImportDetectsConflictForExistingSkill(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	targetHome := t.TempDir()
	passphrase := "pass-1"

	mustWriteFileCore(t, filepath.Join(sourceHome, ".claude.json"), `{"user":"roy"}`)
	mustWriteFileCore(t, filepath.Join(sourceHome, ".claude", "skills", "same", "skill.md"), "from-package")
	mustWriteFileCore(t, filepath.Join(targetHome, ".claude", "skills", "same", "skill.md"), "from-target")

	exportResult, err := Export(context.Background(), ExportParams{
		SourceHome: sourceHome,
		Agent:      "claude-code",
		Version:    "v2",
		OutputDir:  outputDir,
		Passphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	conflicts, err := Import(context.Background(), ImportParams{
		PackagePath: exportResult.PackagePath,
		Passphrase:  passphrase,
		TargetHome:  targetHome,
		Mapping: types.PathMapping{
			SourceHome: "/Users/roy",
			TargetHome: targetHome,
		},
	})
	if err != nil {
		t.Fatalf("import preflight: %v", err)
	}
	if len(conflicts.Buckets["skills"].Conflicts) != 1 || conflicts.Buckets["skills"].Conflicts[0].Key != ".claude/skills/same" {
		t.Fatalf("skills conflict = %#v", conflicts.Buckets["skills"])
	}
}

func TestImportDetectsConflictForMappedProject(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	targetHome := t.TempDir()
	passphrase := "pass-1"

	setupClaudeSource(t, sourceHome)
	mappedDir := filepath.Join(targetHome, "workspace", "foo")
	encodedMappedDir := filepath.Join(targetHome, ".claude", "projects", strings.ReplaceAll(mappedDir, "/", "-"))
	mustWriteFileCore(t, filepath.Join(encodedMappedDir, "history.jsonl"), "target")

	exportResult, err := Export(context.Background(), ExportParams{
		SourceHome: sourceHome,
		Agent:      "claude-code",
		Version:    "v2",
		OutputDir:  outputDir,
		Passphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	conflicts, err := Import(context.Background(), ImportParams{
		PackagePath: exportResult.PackagePath,
		Passphrase:  passphrase,
		TargetHome:  targetHome,
		Mapping: types.PathMapping{
			SourceHome: "/Users/roy",
			TargetHome: targetHome,
			ProjectMappings: []types.PathPair{
				{From: "/Users/roy/projects/foo", To: mappedDir},
			},
		},
	})
	if err != nil {
		t.Fatalf("import preflight: %v", err)
	}
	if len(conflicts.Buckets["projects"].Conflicts) != 1 {
		t.Fatalf("project conflicts = %#v", conflicts.Buckets["projects"])
	}
}

func TestExportPackageDoesNotContainSecretsField(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	passphrase := "pass-1"

	mustWriteFileCore(t, filepath.Join(sourceHome, ".claude.json"), `{"user":"roy","secrets":{"token":"abc"}}`)
	exportResult, err := Export(context.Background(), ExportParams{
		SourceHome: sourceHome,
		Agent:      "claude-code",
		Version:    "v2",
		OutputDir:  outputDir,
		Passphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	root, _, err := pack.UnpackPackage(exportResult.PackagePath, passphrase)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(root))
	data, err := ioutil.ReadFile(filepath.Join(root, ".claude.json"))
	if err != nil {
		t.Fatalf("read package: %v", err)
	}
	if strings.Contains(string(data), `"secrets"`) {
		t.Fatalf("package still contains secrets field: %s", string(data))
	}
}

func setupClaudeSource(t *testing.T, home string) {
	t.Helper()
	mustWriteFileCore(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	mustWriteFileCore(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark","hooks":{"pre":{"command":"/bin/echo ok"}}}`)
	mustWriteFileCore(t, filepath.Join(home, ".claude", "projects", "-Users-roy-projects-foo", "history.jsonl"), "{\"cwd\":\"/Users/roy/projects/foo\",\"tool\":\"/usr/bin/git\"}\n")
	mustWriteFileCore(t, filepath.Join(home, ".claude", "sessions", "ignored.txt"), "skip")
	internalSkill := filepath.Join(home, "skills-src", "existing")
	mustWriteFileCore(t, filepath.Join(internalSkill, "skill.md"), "internal")
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.Symlink(internalSkill, filepath.Join(home, ".claude", "skills", "existing")); err != nil {
		t.Fatalf("symlink internal: %v", err)
	}

	externalRoot := t.TempDir()
	externalSkill := filepath.Join(externalRoot, "outside")
	mustWriteFileCore(t, filepath.Join(externalSkill, "skill.md"), "external")
	if err := os.Symlink(externalSkill, filepath.Join(home, ".claude", "skills", "external")); err != nil {
		t.Fatalf("symlink external: %v", err)
	}
}

func mustWriteFileCore(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := ioutil.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
