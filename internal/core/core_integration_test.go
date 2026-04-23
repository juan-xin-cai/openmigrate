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

func TestExportIncludesDesktopDataAndInspect(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	passphrase := "pass-1"

	setupDesktopSource(t, sourceHome, "acct-1")
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

	root, meta, err := pack.UnpackPackage(exportResult.PackagePath, passphrase)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(root))

	configData, err := ioutil.ReadFile(filepath.Join(root, "Library", "Application Support", "Claude", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(configData), "oauth:token") || strings.Contains(string(configData), "dxt:allowlist") {
		t.Fatalf("desktop config still contains stripped fields: %s", configData)
	}
	preferencesData, err := ioutil.ReadFile(filepath.Join(root, "Library", "Application Support", "Claude", "Preferences"))
	if err != nil {
		t.Fatalf("read preferences: %v", err)
	}
	if strings.Contains(string(preferencesData), "device_id_salt") || strings.Contains(string(preferencesData), "per_host_zoom_levels") {
		t.Fatalf("preferences still contains stripped fields: %s", preferencesData)
	}
	sessionData, err := ioutil.ReadFile(filepath.Join(root, "Library", "Application Support", "Claude", "local-agent-mode-sessions", "s1", ".claude", ".claude.json"))
	if err != nil {
		t.Fatalf("read local session config: %v", err)
	}
	if strings.Contains(string(sessionData), "oauthAccount") {
		t.Fatalf("local agent session still contains oauthAccount: %s", sessionData)
	}
	if _, err := os.Stat(filepath.Join(root, "Library", "Application Support", "Claude", "local-agent-mode-sessions", "s1", ".audit-key")); !os.IsNotExist(err) {
		t.Fatalf(".audit-key should be excluded, err=%v", err)
	}
	if meta.OwnerAccountID != "acct-1" {
		t.Fatalf("owner account id = %q", meta.OwnerAccountID)
	}
	if len(meta.AgentTypes) != 1 || meta.AgentTypes[0] != "claude-desktop" {
		t.Fatalf("agent types = %#v", meta.AgentTypes)
	}

	inspected, err := Inspect(context.Background(), types.InspectParams{PkgPath: exportResult.PackagePath})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if inspected.OwnerAccountID != "acct-1" {
		t.Fatalf("inspect owner account id = %q", inspected.OwnerAccountID)
	}
}

func TestImportRejectsDesktopAccountMismatch(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	targetHome := t.TempDir()
	passphrase := "pass-1"

	setupDesktopSource(t, sourceHome, "acct-1")
	writeDesktopAccount(t, targetHome, "acct-2")

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
	_, err = Import(context.Background(), ImportParams{
		PackagePath: exportResult.PackagePath,
		Passphrase:  passphrase,
		TargetHome:  targetHome,
	})
	if !errors.Is(err, types.ErrAccountMismatch) {
		t.Fatalf("err = %v", err)
	}
}

func TestExportWithoutDesktopSessionsLeavesOwnerAccountEmptyAndSkipsAccountCheck(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	targetHome := t.TempDir()
	passphrase := "pass-1"

	setupDesktopSource(t, sourceHome, "acct-1")
	writeDesktopAccount(t, targetHome, "acct-2")

	exportResult, err := Export(context.Background(), ExportParams{
		SourceHome:    sourceHome,
		Agent:         "claude-code",
		Version:       "v2",
		OutputDir:     outputDir,
		Passphrase:    passphrase,
		ExcludeScopes: []string{"sessions"},
	})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	meta, err := Inspect(context.Background(), types.InspectParams{PkgPath: exportResult.PackagePath})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if meta.OwnerAccountID != "" {
		t.Fatalf("owner account id = %q", meta.OwnerAccountID)
	}

	_, err = Import(context.Background(), ImportParams{
		PackagePath: exportResult.PackagePath,
		Passphrase:  passphrase,
		TargetHome:  targetHome,
	})
	if err != nil {
		t.Fatalf("import should skip account check, got %v", err)
	}
}

func TestExportFailsWhenDesktopSessionsIncludedButAccountFileMissing(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()

	mustWriteFileCore(t, filepath.Join(sourceHome, "Library", "Application Support", "Claude", "config.json"), `{"keep":"ok"}`)
	mustWriteFileCore(t, filepath.Join(sourceHome, "Library", "Application Support", "Claude", "local-agent-mode-sessions", "s1", ".claude", ".claude.json"), `{"oauthAccount":"acct-1","keep":"ok"}`)

	_, err := Export(context.Background(), ExportParams{
		SourceHome: sourceHome,
		Agent:      "claude-code",
		Version:    "v2",
		OutputDir:  outputDir,
		Passphrase: "pass-1",
	})
	if err == nil {
		t.Fatalf("expected export to fail when owner account file is missing")
	}
	if !strings.Contains(err.Error(), "ownerAccountId") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyImportAllowsDesktopAccountMismatchWithSkipAndRewritesGitWorktrees(t *testing.T) {
	sourceHome := t.TempDir()
	outputDir := t.TempDir()
	targetHome := t.TempDir()
	passphrase := "pass-1"

	setupDesktopSource(t, sourceHome, "acct-1")
	mustWriteFileCore(
		t,
		filepath.Join(sourceHome, "Library", "Application Support", "Claude", "git-worktrees.json"),
		`{"worktrees":{"main":"/Users/roy/projects/foo"}}`,
	)
	writeDesktopAccount(t, targetHome, "acct-2")

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

	mappedDir := filepath.Join(targetHome, "workspace", "foo")
	result, err := ApplyImport(context.Background(), ImportApplyParams{
		PackagePath:             exportResult.PackagePath,
		Passphrase:              passphrase,
		SkipDesktopSessionCheck: true,
		Mapping: types.PathMapping{
			SourceHome: "/Users/roy",
			TargetHome: targetHome,
			ProjectMappings: []types.PathPair{
				{From: "/Users/roy/projects/foo", To: mappedDir},
			},
		},
		Decisions: types.ConflictDecision{Actions: map[string]types.DecisionAction{
			"Library":                            types.DecisionOverwrite,
			"Library/Application Support":        types.DecisionOverwrite,
			"Library/Application Support/Claude": types.DecisionOverwrite,
			"Library/Application Support/Claude/cowork-enabled-cli-ops.json": types.DecisionOverwrite,
		}},
	})
	if err != nil {
		t.Fatalf("apply import: %v", err)
	}
	if len(result.Snapshot.Targets) == 0 {
		t.Fatalf("snapshot missing targets")
	}

	worktreesData, err := ioutil.ReadFile(filepath.Join(targetHome, "Library", "Application Support", "Claude", "git-worktrees.json"))
	if err != nil {
		t.Fatalf("read git-worktrees: %v", err)
	}
	if !strings.Contains(string(worktreesData), mappedDir) {
		t.Fatalf("git-worktrees not rewritten: %s", worktreesData)
	}
	if strings.Contains(string(worktreesData), "/Users/roy/projects/foo") {
		t.Fatalf("git-worktrees still contains source path: %s", worktreesData)
	}
}

func TestInspectReturnsMetaNotFound(t *testing.T) {
	_, err := Inspect(context.Background(), types.InspectParams{PkgPath: filepath.Join(t.TempDir(), "missing.ommigrate")})
	if !errors.Is(err, types.ErrMetaNotFound) {
		t.Fatalf("err = %v", err)
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

func setupDesktopSource(t *testing.T, home, accountID string) {
	t.Helper()
	writeDesktopAccount(t, home, accountID)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "config.json"), `{"oauth:token":"abc","dxt:allowlist-dev":true,"keep":"ok"}`)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "Preferences"), `{"electron":{"media":{"device_id_salt":"salt","other":"keep"}},"partition":{"one":{"per_host_zoom_levels":{"a":1},"keep":true}}}`)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), `{"mcp":"ok"}`)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "extensions-blocklist.json"), `{"blocked":[]}`)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "git-worktrees.json"), `{"worktrees":{}}`)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "local-agent-mode-sessions", "s1", ".claude", ".claude.json"), `{"oauthAccount":"acct-1","keep":"ok"}`)
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "local-agent-mode-sessions", "s1", ".audit-key"), "secret")
}

func writeDesktopAccount(t *testing.T, home, accountID string) {
	t.Helper()
	mustWriteFileCore(t, filepath.Join(home, "Library", "Application Support", "Claude", "cowork-enabled-cli-ops.json"), `{"ownerAccountId":"`+accountID+`"}`)
}
