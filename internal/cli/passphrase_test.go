package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestReadPassphraseUsesEnvInNonInteractiveMode(t *testing.T) {
	old := os.Getenv("OPENMIGRATE_PASSPHRASE")
	if err := os.Setenv("OPENMIGRATE_PASSPHRASE", "secret"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer os.Setenv("OPENMIGRATE_PASSPHRASE", old)

	value, err := ReadPassphrase("prompt: ", Streams{In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("read passphrase: %v", err)
	}
	if value != "secret" {
		t.Fatalf("value = %q", value)
	}
}

func TestReadPassphraseReturnsErrorWithoutTTYOrEnv(t *testing.T) {
	old := os.Getenv("OPENMIGRATE_PASSPHRASE")
	if err := os.Unsetenv("OPENMIGRATE_PASSPHRASE"); err != nil {
		t.Fatalf("unset env: %v", err)
	}
	defer os.Setenv("OPENMIGRATE_PASSPHRASE", old)

	_, err := ReadPassphrase("prompt: ", Streams{In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	if err != ErrNonInteractiveNoPassphrase {
		t.Fatalf("err = %v", err)
	}
}

func TestReadPassphraseIgnoresEnvOnTTY(t *testing.T) {
	oldEnv := os.Getenv("OPENMIGRATE_PASSPHRASE")
	if err := os.Setenv("OPENMIGRATE_PASSPHRASE", "env-secret"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer os.Setenv("OPENMIGRATE_PASSPHRASE", oldEnv)

	oldIsTerminal, oldReadPassword := isTerminal, readPassword
	isTerminal = func(int) bool { return true }
	readPassword = func(int) ([]byte, error) { return []byte("typed-secret"), nil }
	defer func() {
		isTerminal = oldIsTerminal
		readPassword = oldReadPassword
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()
	defer writer.Close()

	value, err := ReadPassphrase("prompt: ", Streams{In: reader, Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("read passphrase: %v", err)
	}
	if value != "typed-secret" {
		t.Fatalf("value = %q", value)
	}
}
