package accountcheck

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestCheckMatchesTargetAccount(t *testing.T) {
	targetHome := t.TempDir()
	writeOwnerAccount(t, targetHome, "acct-1")
	err := Check(types.PackageMeta{OwnerAccountID: "acct-1"}, targetHome, false, nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
}

func TestCheckRejectsMismatch(t *testing.T) {
	targetHome := t.TempDir()
	writeOwnerAccount(t, targetHome, "acct-2")
	err := Check(types.PackageMeta{OwnerAccountID: "acct-1"}, targetHome, false, nil)
	if !errors.Is(err, types.ErrAccountMismatch) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckAllowsSkip(t *testing.T) {
	targetHome := t.TempDir()
	writeOwnerAccount(t, targetHome, "acct-2")
	logHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", logHome); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer os.Setenv("HOME", oldHome)

	logger, err := omlog.New(nil)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer logger.Close()

	if err := Check(types.PackageMeta{OwnerAccountID: "acct-1"}, targetHome, true, logger); err != nil {
		t.Fatalf("check: %v", err)
	}

	logData, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logData), "orphan") {
		t.Fatalf("expected orphan warning in log, got %s", logData)
	}
}

func TestCheckRejectsMissingTargetAccountFile(t *testing.T) {
	targetHome := t.TempDir()
	err := Check(types.PackageMeta{OwnerAccountID: "acct-1"}, targetHome, false, nil)
	if !errors.Is(err, types.ErrAccountMismatch) {
		t.Fatalf("err = %v", err)
	}
}

func TestExtractSourceAccount(t *testing.T) {
	sourceHome := t.TempDir()
	writeOwnerAccount(t, sourceHome, "acct-1")
	accountID, err := ExtractSourceAccount(sourceHome)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if accountID != "acct-1" {
		t.Fatalf("account id = %q", accountID)
	}
}

func writeOwnerAccount(t *testing.T, home, accountID string) {
	t.Helper()
	path := filepath.Join(home, "Library", "Application Support", "Claude", "cowork-enabled-cli-ops.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"ownerAccountId":"`+accountID+`"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
