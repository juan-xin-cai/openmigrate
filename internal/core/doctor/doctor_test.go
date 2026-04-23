package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestRunReportsDesktopChecks(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldPath := os.Getenv("PATH")
	oldDesktopAppPath := desktopAppPath
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer os.Setenv("HOME", oldHome)
	defer os.Setenv("PATH", oldPath)
	defer func() { desktopAppPath = oldDesktopAppPath }()

	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	claude := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\necho 2.0.0\n"), 0o755); err != nil {
		t.Fatalf("write claude: %v", err)
	}
	plutil := filepath.Join(binDir, "plutil")
	if err := os.WriteFile(plutil, []byte("#!/bin/sh\necho '<plist><string>1.2.3</string></plist>'\n"), 0o755); err != nil {
		t.Fatalf("write plutil: %v", err)
	}
	if err := os.Setenv("PATH", binDir+":"+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	desktopAppPath = filepath.Join(home, "Applications", "Claude.app")
	infoPlist := filepath.Join(desktopAppPath, "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(infoPlist), 0o755); err != nil {
		t.Fatalf("mkdir plist dir: %v", err)
	}
	if err := os.WriteFile(infoPlist, []byte("plist"), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	claudeDir := filepath.Join(home, "Library", "Application Support", "Claude")
	if err := os.MkdirAll(claudeDir, 0o700); err != nil {
		t.Fatalf("mkdir claude dir: %v", err)
	}
	if err := os.Chmod(claudeDir, 0o000); err != nil {
		t.Fatalf("chmod claude dir: %v", err)
	}
	defer os.Chmod(claudeDir, 0o700)

	report, err := Run(Params{
		Mode:                types.DoctorModeImport,
		PackageAgentVersion: "1.9.0",
		AbortOnSkew:         false,
		ExpectedPackageSize: 1,
	}, nil)
	if err != nil {
		t.Fatalf("doctor run: %v", err)
	}

	statusByName := map[string]types.DoctorStatus{}
	for _, item := range report.Items {
		statusByName[item.Name] = item.Status
	}
	if statusByName["claude"] != types.DoctorPass {
		t.Fatalf("claude status = %v", statusByName["claude"])
	}
	if statusByName["version-skew"] != types.DoctorWarn {
		t.Fatalf("version-skew status = %v", statusByName["version-skew"])
	}
	if statusByName["claude-desktop"] != types.DoctorPass {
		t.Fatalf("claude-desktop status = %v", statusByName["claude-desktop"])
	}
	if statusByName["claude-desktop-version"] != types.DoctorPass {
		t.Fatalf("claude-desktop-version status = %v", statusByName["claude-desktop-version"])
	}
	if statusByName["claude-desktop-full-disk-access"] != types.DoctorBlock {
		t.Fatalf("claude-desktop-full-disk-access status = %v", statusByName["claude-desktop-full-disk-access"])
	}
}

func TestRunWarnsWhenDesktopMissing(t *testing.T) {
	oldDesktopAppPath := desktopAppPath
	desktopAppPath = filepath.Join(t.TempDir(), "Claude.app")
	defer func() { desktopAppPath = oldDesktopAppPath }()

	report, err := Run(Params{Mode: types.DoctorModeExport}, nil)
	if err != nil {
		t.Fatalf("doctor run: %v", err)
	}
	for _, item := range report.Items {
		if item.Name == "claude-desktop" && item.Status != types.DoctorWarn {
			t.Fatalf("claude-desktop status = %v", item.Status)
		}
	}
}
