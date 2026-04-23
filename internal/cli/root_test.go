package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/openmigrate/openmigrate/internal/buildinfo"
)

func TestRootHelpShowsSubcommands(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "--help"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, errOut.String())
	}
	text := out.String()
	for _, sub := range []string{"doctor", "export", "inspect", "import", "rollback"} {
		if !strings.Contains(text, sub) {
			t.Fatalf("help missing %s: %q", sub, text)
		}
	}
}

func TestUnknownCommandReturnsExitCode2(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"openmigrate", "unknown-cmd"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 2 {
		t.Fatalf("exit code = %d stderr=%q", code, errOut.String())
	}
}

func TestRootVersionShowsBuildInfo(t *testing.T) {
	oldArgs := os.Args
	oldVersion, oldCommit, oldBuildDate := buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate
	defer func() {
		os.Args = oldArgs
		buildinfo.Version = oldVersion
		buildinfo.Commit = oldCommit
		buildinfo.BuildDate = oldBuildDate
	}()

	buildinfo.Version = "v1.2.3"
	buildinfo.Commit = "abc1234"
	buildinfo.BuildDate = "2026-04-23T10:11:12Z"
	os.Args = []string{"openmigrate", "--version"}

	var out, errOut bytes.Buffer
	code := Execute(Streams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, errOut.String())
	}
	text := out.String()
	for _, want := range []string{"openmigrate v1.2.3", "commit: abc1234", "built: 2026-04-23T10:11:12Z"} {
		if !strings.Contains(text, want) {
			t.Fatalf("version output missing %q: %q", want, text)
		}
	}
}
