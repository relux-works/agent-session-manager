package merkleinventory

import (
	"errors"
	"fmt"
)

var (
	ErrUnsupportedVersion = errors.New("inventory version is outside this implementation")
	ErrInvalidRequest     = errors.New("invalid inventory request")
	ErrInvalidObject      = errors.New("invalid inventory object")
	ErrNotFound           = errors.New("inventory node or object not found")
	ErrIntegrity          = errors.New("inventory integrity failure")
	ErrResponseTooLarge   = errors.New("objects.get response exceeds negotiated line limit")
	ErrExcluded           = errors.New("object class is excluded from the replicated union")
)

// Refusal carries the literal Section 11 failure token when one is specified.
// The transport owner decides how to frame that operation-specific failure.
type Refusal struct {
	Code  string
	Cause error
}

func (err *Refusal) Error() string {
	if err.Cause == nil {
		return err.Code
	}
	return fmt.Sprintf("%s: %v", err.Code, err.Cause)
}

func (err *Refusal) Unwrap() error { return err.Cause }

func refusal(code string, cause error) error {
	return &Refusal{Code: code, Cause: cause}
}
