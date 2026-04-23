package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
