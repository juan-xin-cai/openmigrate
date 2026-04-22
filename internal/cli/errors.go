package cli

import (
	"errors"
	"fmt"
)

var (
	ErrNonInteractiveNoPassphrase = errors.New("non-interactive terminal requires OPENMIGRATE_PASSPHRASE")
	ErrUserCanceled               = errors.New("user canceled")
)

type ExitError struct {
	Code    int
	Message string
	Err     error
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func exitf(code int, err error, format string, args ...interface{}) error {
	return &ExitError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

func exitCode(err error) int {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return 2
}
