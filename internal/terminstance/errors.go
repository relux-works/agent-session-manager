package terminstance

import (
	"errors"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Wire error codes this package may report. Every code shared with the
// landed core is a const alias of the landed constant, so the literal is
// spelled once on trunk: retyping it here would fork the vocabulary. The
// four row codes the landed core spells only as string literals inside
// its allowed-error table are spelled literally here and pinned equal to
// the landed table by TestRowCodesAreAdmittedByCheckErrorAllowed, which
// drives each literal through the landed CheckErrorAllowed for its row.
const (
	CodeProtocolError       = terminalbackend.CodeProtocolError
	CodeUnauthorized        = terminalbackend.CodeUnauthorized
	CodeUnavailable         = terminalbackend.CodeUnavailable
	CodePreconditionFailed  = terminalbackend.CodePreconditionFailed
	CodeIdempotencyMismatch = terminalbackend.CodeIdempotencyMismatch
	CodeStaleGeneration     = terminalbackend.CodeStaleGeneration
	CodeCapabilityUnproven  = terminalbackend.CodeCapabilityUnproven
	CodeIntegrityFailure    = terminalbackend.CodeIntegrityFailure
	// CodeTimeout is the §4.C terminal_backend_timeout row code.
	CodeTimeout = "terminal_backend_timeout"
	// CodeProcessFailed is the §4.C terminal_backend_process_failed row code.
	CodeProcessFailed = "terminal_backend_process_failed"
	// CodeQuiesceTimeout is the §4.C quiesce_timeout row code, admitted
	// only on wait-safe-boundary.
	CodeQuiesceTimeout = "quiesce_timeout"
	// CodeStopTimeout is the §4.C stop_timeout row code, admitted only
	// on request-stop.
	CodeStopTimeout = "stop_timeout"
)

// Error is a lifecycle refusal. Code is always one of the Code* constants;
// Detail is a static clause: it never echoes identities, generations,
// keys, digests, or other local data.
type Error struct {
	Code   string
	Detail string
}

func (err *Error) Error() string {
	return "terminal instance refused: " + err.Code + " at " + err.Detail
}

// refuse builds the refusal for one arm.
func refuse(code, detail string) *Error {
	return &Error{Code: code, Detail: detail}
}

// wrapLanded converts a landed terminalbackend refusal into this
// package's uniform error type, preserving the wire code and the static
// landed detail clause. A non-refusal error is never wrapped: callers
// classify uncoded failures explicitly at their own arm.
func wrapLanded(err error) *Error {
	var refusal *terminalbackend.Error
	if errors.As(err, &refusal) {
		return &Error{Code: refusal.Code, Detail: refusal.Detail}
	}
	return nil
}

// ErrorCode reports the wire code of an error produced by this package
// or the landed core, or "" when the error carries no wire code. Engine
// callers use it to read backend-reported codes; tests assert the
// literal codes directly on *Error.
func ErrorCode(err error) string {
	var refusal *Error
	if errors.As(err, &refusal) {
		return refusal.Code
	}
	var landed *terminalbackend.Error
	if errors.As(err, &landed) {
		return landed.Code
	}
	return ""
}
