package cli

import (
	"io"
	"os"

	"golang.org/x/term"
)

func IsNonInteractive(yesFlag bool, input io.Reader) bool {
	if yesFlag {
		return true
	}
	file, ok := input.(*os.File)
	if !ok {
		return true
	}
	return !term.IsTerminal(int(file.Fd()))
}
