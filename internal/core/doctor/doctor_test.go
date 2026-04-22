package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestRunReportsBlocksAndWarns(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer os.Setenv("HOME", oldHome)

	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	claude := filepath.Join(binDir, "claude")
	script := "#!/bin/sh\necho 2.0.0\n"
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude: %v", err)
	}
	if err := os.Setenv("PATH", binDir+":"+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath)

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
	if statusByName["full-disk-access"] != types.DoctorBlock {
		t.Fatalf("full-disk-access status = %v", statusByName["full-disk-access"])
	}
}
