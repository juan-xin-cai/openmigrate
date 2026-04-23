package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestConfirmSkipDesktopAccountCheckYes(t *testing.T) {
	var out bytes.Buffer
	skip, err := confirmSkipDesktopAccountCheck(bytes.NewBufferString("y\n"), &out, false, types.ErrAccountMismatch)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !skip {
		t.Fatalf("expected skip=true")
	}
}

func TestConfirmSkipDesktopAccountCheckNo(t *testing.T) {
	var out bytes.Buffer
	skip, err := confirmSkipDesktopAccountCheck(bytes.NewBufferString("n\n"), &out, false, types.ErrAccountMismatch)
	if skip {
		t.Fatalf("expected skip=false")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("err = %#v", err)
	}
}

func TestConfirmSkipDesktopAccountCheckNonInteractive(t *testing.T) {
	var out bytes.Buffer
	skip, err := confirmSkipDesktopAccountCheck(bytes.NewBuffer(nil), &out, true, types.ErrAccountMismatch)
	if skip {
		t.Fatalf("expected skip=false")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("err = %#v", err)
	}
}
