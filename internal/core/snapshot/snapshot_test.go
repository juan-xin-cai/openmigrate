package snapshot

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestCreateSnapshotAndRollback(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer os.Setenv("HOME", oldHome)

	target := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ioutil.WriteFile(target, []byte(`{"theme":"dark"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	logger := omlog.MustLogger(nil)
	defer logger.Close()

	meta, err := CreateSnapshot([]string{target}, "pass-1", logger)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if _, err := os.Stat(meta.ArchivePath); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove target: %v", err)
	}
	if err := Rollback(meta, "pass-1", logger); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	data, err := ioutil.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if string(data) != `{"theme":"dark"}` {
		t.Fatalf("restored = %q", string(data))
	}
}

func TestCreateSnapshotReturnsDiskFull(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	defer os.Setenv("HOME", oldHome)

	huge := filepath.Join(home, "huge.bin")
	file, err := os.Create(huge)
	if err != nil {
		t.Fatalf("create huge: %v", err)
	}
	if err := file.Truncate(1 << 50); err != nil {
		t.Fatalf("truncate huge: %v", err)
	}
	_ = file.Close()

	_, err = CreateSnapshot([]string{huge}, "pass-1", nil)
	if !errors.Is(err, types.ErrDiskFull) {
		t.Fatalf("expected ErrDiskFull, got %v", err)
	}
}
