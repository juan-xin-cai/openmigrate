package types

import (
	"errors"
	"fmt"
)

var (
	ErrNoPassphrase             = errors.New("passphrase is required")
	ErrDecryptFailed            = errors.New("decrypt failed")
	ErrDiskFull                 = errors.New("insufficient disk space")
	ErrPermissionDenied         = errors.New("permission denied")
	ErrSchemaVersion            = errors.New("unsupported package schema version")
	ErrContextCanceled          = errors.New("operation canceled")
	ErrPathMappingRequired      = errors.New("path mapping is required before conflict detection")
	ErrConflictDecisionRequired = errors.New("conflict decision is required")
	ErrSnapshotNotFound         = errors.New("snapshot not found")
	ErrNotJSON                  = errors.New("input is not valid json")
	ErrMetaNotFound             = errors.New("package meta not found")
	ErrConflictingScopeFilter   = errors.New("only-scopes and exclude-scopes cannot be combined")
	ErrAccountMismatch          = errors.New("claude desktop account mismatch")
)

type ErrorCode string

const (
	CodeInvalidInput ErrorCode = "invalid_input"
	CodeDecrypt      ErrorCode = "decrypt_failed"
	CodeFilesystem   ErrorCode = "filesystem"
	CodeConflict     ErrorCode = "conflict"
)

type OpError struct {
	Code ErrorCode
	Op   string
	Err  error
}

func (e *OpError) Error() string {
	if e == nil {
		return ""
	}
	if e.Op == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *OpError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Wrap(code ErrorCode, op string, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Code: code, Op: op, Err: err}
}
