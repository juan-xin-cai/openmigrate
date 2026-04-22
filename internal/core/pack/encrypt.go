package pack

import (
	"io"

	"filippo.io/age"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func Encrypt(src io.Reader, dst io.Writer, passphrase string) error {
	if passphrase == "" {
		return types.ErrNoPassphrase
	}
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return err
	}
	writer, err := age.Encrypt(dst, recipient)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, src); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func Decrypt(src io.Reader, dst io.Writer, passphrase string) error {
	if passphrase == "" {
		return types.ErrNoPassphrase
	}
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return types.ErrDecryptFailed
	}
	reader, err := age.Decrypt(src, identity)
	if err != nil {
		return types.ErrDecryptFailed
	}
	if _, err := io.Copy(dst, reader); err != nil {
		return types.ErrDecryptFailed
	}
	return nil
}
