package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestIsNonInteractive(t *testing.T) {
	if !IsNonInteractive(true, os.Stdin) {
		t.Fatalf("--yes should force non-interactive")
	}
	if !IsNonInteractive(false, bytes.NewBuffer(nil)) {
		t.Fatalf("buffer input should be treated as non-interactive")
	}
}
