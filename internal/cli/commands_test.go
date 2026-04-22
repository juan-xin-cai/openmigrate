package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(out.String(), "账号检查: PASS") || !strings.Contains(out.String(), "导入完成") {
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
