package errors

import stderrors "errors"

var (
	ErrPermissionDenied = stderrors.New("permission denied")
	ErrToolNotFound     = stderrors.New("tool not found")
	ErrInvalidPath      = stderrors.New("invalid path")
	ErrTimeout          = stderrors.New("timeout")
	ErrInvalidInput     = stderrors.New("invalid input")
)
