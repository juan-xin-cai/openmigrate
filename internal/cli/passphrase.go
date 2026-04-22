package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

var (
	isTerminal   = term.IsTerminal
	readPassword = term.ReadPassword
)

func ReadPassphrase(prompt string, streams Streams) (string, error) {
	file, ok := streams.In.(*os.File)
	isTTY := ok && isTerminal(int(file.Fd()))
	if !isTTY {
		if env := strings.TrimSpace(os.Getenv("OPENMIGRATE_PASSPHRASE")); env != "" {
			return env, nil
		}
		return "", ErrNonInteractiveNoPassphrase
	}
	if _, err := fmt.Fprint(streams.ErrOut, prompt); err != nil {
		return "", err
	}
	defer fmt.Fprintln(streams.ErrOut)
	value, err := readPassword(int(file.Fd()))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(value)), nil
}
