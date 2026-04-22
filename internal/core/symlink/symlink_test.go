package symlink

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestResolveHandlesInternalAndExternalSymlink(t *testing.T) {
	home := t.TempDir()
	internalTarget := filepath.Join(home, "real-skill")
	if err := os.MkdirAll(internalTarget, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(internalTarget, "skill.md"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	internalLink := filepath.Join(home, ".claude", "skills", "linked")
	if err := os.MkdirAll(filepath.Dir(internalLink), 0o755); err != nil {
		t.Fatalf("mkdir link dir: %v", err)
	}
	if err := os.Symlink(internalTarget, internalLink); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	externalRoot := t.TempDir()
	externalTarget := filepath.Join(externalRoot, "ext-skill")
	if err := os.MkdirAll(externalTarget, 0o755); err != nil {
		t.Fatalf("mkdir ext: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(externalTarget, "ext.md"), []byte("ext"), 0o644); err != nil {
		t.Fatalf("write ext: %v", err)
	}
	externalLink := filepath.Join(home, ".claude", "skills", "external")
	if err := os.Symlink(externalTarget, externalLink); err != nil {
		t.Fatalf("symlink ext: %v", err)
	}

	manifest := types.Manifest{
		SourceHome: home,
		Entries: []types.FileEntry{
			{SourcePath: internalLink, RelativePath: ".claude/skills/linked", IsSymlink: true, SymlinkTarget: internalTarget},
			{SourcePath: externalLink, RelativePath: ".claude/skills/external", IsSymlink: true, SymlinkTarget: externalTarget},
		},
	}
	logger := omlog.MustLogger(nil)
	defer logger.Close()
	got, err := Resolve(manifest, home, logger)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Links) != 2 {
		t.Fatalf("links = %#v", got.Links)
	}
	if got.Links[0].External {
		t.Fatalf("internal link marked external: %#v", got.Links[0])
	}
	if !got.Links[1].External {
		t.Fatalf("external link not marked external: %#v", got.Links[1])
	}
	if got.Links[1].Warning == "" {
		t.Fatalf("expected external warning")
	}
}

func TestResolveStopsDeepSymlinkChain(t *testing.T) {
	home := t.TempDir()
	finalDir := filepath.Join(home, "final")
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		t.Fatalf("mkdir final: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(finalDir, "data.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write final: %v", err)
	}
	link3 := filepath.Join(home, "link3")
	link2 := filepath.Join(home, "link2")
	link1 := filepath.Join(home, ".claude", "skills", "chain")
	if err := os.MkdirAll(filepath.Dir(link1), 0o755); err != nil {
		t.Fatalf("mkdir chain: %v", err)
	}
	if err := os.Symlink(finalDir, link3); err != nil {
		t.Fatalf("symlink3: %v", err)
	}
	if err := os.Symlink(link3, link2); err != nil {
		t.Fatalf("symlink2: %v", err)
	}
	if err := os.Symlink(link2, link1); err != nil {
		t.Fatalf("symlink1: %v", err)
	}

	manifest := types.Manifest{
		SourceHome: home,
		Entries:    []types.FileEntry{{SourcePath: link1, RelativePath: ".claude/skills/chain", IsSymlink: true, SymlinkTarget: link2}},
	}
	logger := omlog.MustLogger(nil)
	defer logger.Close()
	got, err := Resolve(manifest, home, logger)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Links) != 1 || got.Links[0].Warning == "" {
		t.Fatalf("expected deep-chain warning, got %#v", got.Links)
	}
}

func TestResolveCycleDoesNotLoopForever(t *testing.T) {
	home := t.TempDir()
	linkA := filepath.Join(home, ".claude", "skills", "cycle")
	linkB := filepath.Join(home, "cycle-b")
	if err := os.MkdirAll(filepath.Dir(linkA), 0o755); err != nil {
		t.Fatalf("mkdir cycle: %v", err)
	}
	if err := os.Symlink(linkB, linkA); err != nil {
		t.Fatalf("symlink A: %v", err)
	}
	if err := os.Symlink(linkA, linkB); err != nil {
		t.Fatalf("symlink B: %v", err)
	}

	manifest := types.Manifest{
		SourceHome: home,
		Entries:    []types.FileEntry{{SourcePath: linkA, RelativePath: ".claude/skills/cycle", IsSymlink: true, SymlinkTarget: linkB}},
	}
	logger := omlog.MustLogger(nil)
	defer logger.Close()
	got, err := Resolve(manifest, home, logger)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got.Links) != 1 || got.Links[0].Warning == "" {
		t.Fatalf("expected cycle warning, got %#v", got.Links)
	}
}

func TestRestoreKeepsEntityWhenSymlinkCreationFails(t *testing.T) {
	targetHome := t.TempDir()
	entityPath := filepath.Join(targetHome, ".claude", "skills", "linked")
	targetPath := filepath.Join(targetHome, "skills-src", "linked")
	if err := os.MkdirAll(entityPath, 0o755); err != nil {
		t.Fatalf("mkdir entity: %v", err)
	}
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(entityPath, "skill.md"), []byte("entity"), 0o644); err != nil {
		t.Fatalf("write entity: %v", err)
	}
	parent := filepath.Dir(entityPath)
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}
	defer os.Chmod(parent, 0o755)

	logger := omlog.MustLogger(nil)
	defer logger.Close()
	if err := Restore(targetHome, []types.LinkRelation{{
		LinkRelativePath:   ".claude/skills/linked",
		TargetRelativePath: "skills-src/linked",
	}}, logger); err != nil {
		t.Fatalf("restore: %v", err)
	}
	info, err := os.Lstat(entityPath)
	if err != nil {
		t.Fatalf("stat entity: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("entity should remain, got symlink")
	}
	if _, err := os.Stat(filepath.Join(entityPath, "skill.md")); err != nil {
		t.Fatalf("entity file missing: %v", err)
	}
}
