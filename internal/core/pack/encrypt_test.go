package pack

import (
	"bytes"
	"errors"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plain := []byte("secret payload")
	var encrypted bytes.Buffer
	if err := Encrypt(bytes.NewReader(plain), &encrypted, "pass-1"); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	var decrypted bytes.Buffer
	if err := Decrypt(bytes.NewReader(encrypted.Bytes()), &decrypted, "pass-1"); err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got := decrypted.String(); got != string(plain) {
		t.Fatalf("decrypt mismatch: %q", got)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	var encrypted bytes.Buffer
	if err := Encrypt(bytes.NewBufferString("payload"), &encrypted, "pass-1"); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	err := Decrypt(bytes.NewReader(encrypted.Bytes()), &bytes.Buffer{}, "pass-2")
	if !errors.Is(err, types.ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestEncryptEmptyPassphrase(t *testing.T) {
	err := Encrypt(bytes.NewBufferString("payload"), &bytes.Buffer{}, "")
	if !errors.Is(err, types.ErrNoPassphrase) {
		t.Fatalf("expected ErrNoPassphrase, got %v", err)
	}
}
