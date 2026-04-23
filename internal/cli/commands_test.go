package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/pack"
)

func TestDoctorBlockReturnsExitCode2(t *testing.T) {
	home := t.TempDir()
	oldHome, oldPath, oldArgs := os.Getenv("HOME"), os.Getenv("PATH"), os.Args
	defer func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.Setenv("PATH", oldPath)
		os.Args = oldArgs
	}()
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("PATH", "")
	os.Args = []string{"openmigrate", "doctor"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 2 {
		t.Fatalf("exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "建议") {
		t.Fatalf("doctor output missing suggestion: %q", out.String())
	}
}

func TestExportWithEnvPassphraseExitCode0(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark"}`)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "pkg.ommigrate")); err != nil {
		t.Fatalf("package missing: %v", err)
	}
	if !strings.Contains(out.String(), "导出完成") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestExportOnlyAndExcludeConflictReturnsExitCode2(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate"), "--only", "settings", "--exclude", "sessions"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 2 {
		t.Fatalf("exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String()+errOut.String(), "--only 与 --exclude 不能同时指定") {
		t.Fatalf("unexpected output: out=%q err=%q", out.String(), errOut.String())
	}
}

func TestExportOnlySettingsExcludesSkills(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark"}`)
	writeCLIFile(t, filepath.Join(home, ".claude", "skills", "demo", "skill.md"), "demo")

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate"), "--only", "settings"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	root, _, err := pack.UnpackPackage(filepath.Join(outDir, "pkg.ommigrate"), "pass-1")
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(root))
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "demo", "skill.md")); !os.IsNotExist(err) {
		t.Fatalf("skills entry should be excluded, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "settings.json")); err != nil {
		t.Fatalf("settings entry missing: %v", err)
	}
}

func TestExportNoHistoryExcludesProjectJSONL(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(home, ".claude", "projects", "-Users-roy-projects-foo", "history.jsonl"), "{}\n")
	writeCLIFile(t, filepath.Join(home, ".claude", "projects", "-Users-roy-projects-foo", "meta.json"), `{"cwd":"/Users/roy/projects/foo"}`)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate"), "--no-history"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	root, _, err := pack.UnpackPackage(filepath.Join(outDir, "pkg.ommigrate"), "pass-1")
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(root))
	if _, err := os.Stat(filepath.Join(root, ".claude", "projects", "-Users-roy-projects-foo", "history.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("history jsonl should be excluded, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "projects", "-Users-roy-projects-foo", "meta.json")); err != nil {
		t.Fatalf("project metadata should remain: %v", err)
	}
}

func TestImportWrongPassphraseReturnsExitCode2(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark"}`)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	var out, errOut bytes.Buffer
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}
	if code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); code != 0 {
		t.Fatalf("export exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}

	targetHome := t.TempDir()
	restoreTargetHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", targetHome)
	defer os.Setenv("HOME", restoreTargetHome)
	oldPass := os.Getenv("OPENMIGRATE_PASSPHRASE")
	_ = os.Setenv("OPENMIGRATE_PASSPHRASE", "wrong-pass")
	defer os.Setenv("OPENMIGRATE_PASSPHRASE", oldPass)

	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "import", "--yes", filepath.Join(outDir, "pkg.ommigrate")}
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 2 {
		t.Fatalf("import exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String()+errOut.String(), "密码错误") {
		t.Fatalf("expected wrong-passphrase message, out=%q err=%q", out.String(), errOut.String())
	}
	for _, path := range []string{
		filepath.Join(targetHome, ".claude"),
		filepath.Join(targetHome, ".claude.json"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("target data should stay absent, path=%s err=%v", path, err)
		}
	}
}

func TestImportYesAndRollbackExitCode0(t *testing.T) {
	sourceHome := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, sourceHome)
	defer restore()

	writeCLIFile(t, filepath.Join(sourceHome, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(sourceHome, ".claude", "settings.json"), `{"theme":"dark"}`)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	var out, errOut bytes.Buffer
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}
	if code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); code != 0 {
		t.Fatalf("export exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}

	targetHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", targetHome)
	defer os.Setenv("HOME", oldHome)

	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "import", "--yes", filepath.Join(outDir, "pkg.ommigrate")}
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("import exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "账号检查: 无 Desktop sessions，跳过") || !strings.Contains(out.String(), "导入完成") {
		t.Fatalf("unexpected import output: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(targetHome, ".claude", "settings.json")); err != nil {
		t.Fatalf("imported settings missing: %v", err)
	}

	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "rollback"}
	code = Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("rollback exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "回滚完成") {
		t.Fatalf("unexpected rollback output: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(targetHome, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("expected imported data removed after rollback, err=%v", err)
	}
}

func TestInspectCommandExitCode0(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark"}`)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	var out, errOut bytes.Buffer
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}
	if code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); code != 0 {
		t.Fatalf("export exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "inspect", filepath.Join(outDir, "pkg.ommigrate")}
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("inspect exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "主机名:") || !strings.Contains(out.String(), "文件数:") {
		t.Fatalf("unexpected inspect output: %q", out.String())
	}
}

func TestInspectCommandWorksWithoutPassphrase(t *testing.T) {
	home := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, home)
	defer restore()

	setupCLIDesktopSource(t, home, "acct-1")

	oldArgs := os.Args
	oldPass := os.Getenv("OPENMIGRATE_PASSPHRASE")
	defer func() {
		os.Args = oldArgs
		_ = os.Setenv("OPENMIGRATE_PASSPHRASE", oldPass)
	}()

	var out, errOut bytes.Buffer
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}
	if code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); code != 0 {
		t.Fatalf("export exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}

	_ = os.Unsetenv("OPENMIGRATE_PASSPHRASE")
	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "inspect", filepath.Join(outDir, "pkg.ommigrate")}
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("inspect exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "Desktop 账号 ID: acct-1") {
		t.Fatalf("inspect output missing owner account id: %q", text)
	}
	if strings.Contains(text, "请输入") || strings.Contains(errOut.String(), "请输入") {
		t.Fatalf("inspect should not request passphrase, out=%q err=%q", text, errOut.String())
	}
}

func TestInspectCommandReturnsExitCode2WhenMetaMissing(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "inspect", filepath.Join(t.TempDir(), "missing.ommigrate")}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 2 {
		t.Fatalf("exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String()+errOut.String(), "找不到元信息文件") {
		t.Fatalf("unexpected output: out=%q err=%q", out.String(), errOut.String())
	}
}

func TestImportAccountMismatchReturnsExitCode1InNonInteractiveMode(t *testing.T) {
	sourceHome := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, sourceHome)
	defer restore()

	setupCLIDesktopSource(t, sourceHome, "acct-1")

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	var out, errOut bytes.Buffer
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}
	if code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); code != 0 {
		t.Fatalf("export exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}

	targetHome := t.TempDir()
	writeCLIFile(t, filepath.Join(targetHome, "Library", "Application Support", "Claude", "cowork-enabled-cli-ops.json"), `{"ownerAccountId":"acct-2"}`)
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", targetHome)
	defer os.Setenv("HOME", oldHome)

	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "import", "--yes", filepath.Join(outDir, "pkg.ommigrate")}
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 1 {
		t.Fatalf("import exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "账号不一致") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestImportSkipDesktopSessionCheckReturnsExitCode0(t *testing.T) {
	sourceHome := t.TempDir()
	outDir := t.TempDir()
	restore := setupCLIEnv(t, sourceHome)
	defer restore()

	setupCLIDesktopSource(t, sourceHome, "acct-1")

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	var out, errOut bytes.Buffer
	os.Args = []string{"openmigrate", "export", "--out", filepath.Join(outDir, "pkg.ommigrate")}
	if code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); code != 0 {
		t.Fatalf("export exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}

	targetHome := t.TempDir()
	writeCLIFile(t, filepath.Join(targetHome, "Library", "Application Support", "Claude", "cowork-enabled-cli-ops.json"), `{"ownerAccountId":"acct-2"}`)
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", targetHome)
	defer os.Setenv("HOME", oldHome)

	out.Reset()
	errOut.Reset()
	os.Args = []string{"openmigrate", "import", "--yes", "--skip-desktop-session-check", filepath.Join(outDir, "pkg.ommigrate")}
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("import exit code = %d out=%q err=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "导入完成") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func setupCLIEnv(t *testing.T, home string) func() {
	t.Helper()
	oldHome, oldPath, oldPass := os.Getenv("HOME"), os.Getenv("PATH"), os.Getenv("OPENMIGRATE_PASSPHRASE")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	if err := os.Setenv("OPENMIGRATE_PASSPHRASE", "pass-1"); err != nil {
		t.Fatalf("set passphrase: %v", err)
	}
	claudeDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "claude"), []byte("#!/bin/sh\necho 1.0.0\n"), 0o755); err != nil {
		t.Fatalf("write claude: %v", err)
	}
	if err := os.Setenv("PATH", claudeDir); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, "Library", "Application Support", "Claude"), 0o755); err != nil {
		t.Fatalf("mkdir Claude dir: %v", err)
	}
	return func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.Setenv("PATH", oldPath)
		_ = os.Setenv("OPENMIGRATE_PASSPHRASE", oldPass)
	}
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setupCLIDesktopSource(t *testing.T, home, accountID string) {
	t.Helper()
	writeCLIFile(t, filepath.Join(home, ".claude.json"), `{"user":"roy"}`)
	writeCLIFile(t, filepath.Join(home, "Library", "Application Support", "Claude", "cowork-enabled-cli-ops.json"), `{"ownerAccountId":"`+accountID+`"}`)
	writeCLIFile(t, filepath.Join(home, "Library", "Application Support", "Claude", "config.json"), `{"keep":"ok"}`)
	writeCLIFile(t, filepath.Join(home, "Library", "Application Support", "Claude", "local-agent-mode-sessions", "s1", ".claude", ".claude.json"), `{"oauthAccount":"acct-1","keep":"ok"}`)
}
