package dirnode

import (
	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// This file owns the Section 7.9 major bootstrap decision: the
// caller enumerates its locally supported majors in strictly
// descending order with one manifest request each, each attempt on a
// fresh process, and only the exact downgrade tuple authorizes the
// next lower attempt.
//
// The decision is a pure function over the attempt outcome, so it is
// testable without spawning processes: DecideBootstrapStep takes the
// request the attempt was issued under, the single response frame if
// the process produced one, and the process exit status, and returns
// exactly one of success (with the validated manifest), authorized
// downgrade (with the next lower major to attempt on a fresh
// process the caller claims in its guard), or a terminal refusal. Process lifecycle —
// launching, terminating, reaping — belongs to the caller; the
// ProcessGuard below is the host check that a returned or failed
// process is never reused for a lower major.

// BootstrapDecision is one DecideBootstrapStep outcome. Exactly one
// of Manifest, Downgrade, or Terminal carries the verdict: Done
// with a validated manifest whose supported versions contain the
// selected one, Downgrade with the next lower major to attempt on a
// fresh process, or a terminal refusal that must not cause a
// lower-major attempt.
type BootstrapDecision struct {
	Done      bool
	Manifest  Manifest
	Downgrade bool
	NextMajor int
	Terminal  error
}

// NextLowerMajor reports the next lower locally supported major to
// attempt. The second result is false when no lower major remains:
// the caller then terminates with its own incompatible_protocol,
// never with a relabeled envelope.
func NextLowerMajor(major int) (int, bool) {
	for index, supported := range supportedMajors {
		if supported != major {
			continue
		}
		if index+1 < len(supportedMajors) {
			return supportedMajors[index+1], true
		}
		return 0, false
	}
	return 0, false
}

// isExactDowngradeError reports whether the decoded failure is the
// exact downgrade trigger: Structured Error 1.2
// incompatible_protocol with exit 6 and retryable=false. The
// envelope half of the tuple — one well-framed failure echoing
// protocol, protocol_version, request_id, and operation=manifest
// with response schema_version 1.0.0 — is already established by
// CheckFailureEnvelope before this runs; the process-exit half is
// checked by DecideBootstrapStep against the reaped status. Every
// field is compared: a tuple that names another code, another exit,
// or claims retry permission is an ordinary failure, never
// negotiation evidence.
func isExactDowngradeError(failure *axerror.Error) bool {
	if failure == nil {
		return false
	}
	if failure.Code() != "incompatible_protocol" {
		return false
	}
	if failure.ExitCode() != 6 {
		return false
	}
	if failure.Retryable() {
		return false
	}
	return true
}

// ProcessGuard is the host check that a process which returned or
// failed an attempt is never reused for a lower major. The caller
// claims one opaque token per launched process before issuing the
// attempt; claiming the same token twice is a caller error caught
// before any node surface is touched. The zero value is unusable: a
// guard must come from NewProcessGuard so the seen set exists.
type ProcessGuard struct {
	seen map[string]bool
}

// NewProcessGuard returns an empty attempt-process guard.
func NewProcessGuard() ProcessGuard {
	return ProcessGuard{seen: map[string]bool{}}
}

// Claim records one attempt process. A token claimed before — a
// returned or failed process offered for a lower major — is
// refused with invalid_config.
func (guard ProcessGuard) Claim(token string) error {
	if token == "" {
		failure, err := failInvalid("bootstrap process token is empty", "process")
		if err != nil {
			return err
		}
		return failure
	}
	if guard.seen[token] {
		failure, err := failInvalid("bootstrap process was already used for an attempt", "process")
		if err != nil {
			return err
		}
		return failure
	}
	guard.seen[token] = true
	return nil
}

// DecideBootstrapStep decides one manifest attempt. frame carries
// the process's single response line, or is empty when the attempt
// produced no valid response (timeout, signal, exit without one
// valid frame, authentication/allowlist failure at the transport).
// exitStatus is the reaped process status.
//
// Success requires a well-framed success response with every echo
// exact, a complete schema-valid manifest whose
// supported_protocol_versions contains the exact attempted version,
// and process exit 0. The exact downgrade tuple — one well-framed
// failure with incompatible_protocol, exit 6, retryable=false, and
// process exit 6 — authorizes the next lower major when one
// remains, and terminates with the caller's own
// incompatible_protocol when none remains. A wrong or missing echo,
// malformed/partial/extra frame, invalid manifest, timeout, signal,
// nonmatching exit status, or any error other than the exact tuple
// is terminal and never causes a lower-major attempt.
func DecideBootstrapStep(want Request, frame []byte, exitStatus int) BootstrapDecision {
	if want.Operation != OpManifest {
		failure, err := failInvalid("bootstrap attempt does not carry operation manifest", "operation")
		if err != nil {
			return BootstrapDecision{Terminal: err}
		}
		return BootstrapDecision{Terminal: failure}
	}
	if len(frame) == 0 {
		failure, err := failTransport("bootstrap attempt produced no response frame", "response")
		if err != nil {
			return BootstrapDecision{Terminal: err}
		}
		return BootstrapDecision{Terminal: failure}
	}
	if body, err := CheckSuccessEnvelope(frame, want); err == nil {
		manifest, err := DecodeManifest(body)
		if err != nil {
			return BootstrapDecision{Terminal: err}
		}
		if !manifest.Supports(requestVersion(want)) {
			failure, faultErr := failViolation("bootstrap manifest does not contain the attempted version", "supported_protocol_versions")
			if faultErr != nil {
				return BootstrapDecision{Terminal: faultErr}
			}
			return BootstrapDecision{Terminal: failure}
		}
		if exitStatus != 0 {
			failure, faultErr := failViolation("bootstrap success carries a nonmatching exit status", "exit_status")
			if faultErr != nil {
				return BootstrapDecision{Terminal: faultErr}
			}
			return BootstrapDecision{Terminal: failure}
		}
		return BootstrapDecision{Done: true, Manifest: manifest}
	}
	failure, err := CheckFailureEnvelope(frame, want)
	if err != nil {
		return BootstrapDecision{Terminal: err}
	}
	if !isExactDowngradeError(failure) {
		terminal, terminalErr := failViolation("bootstrap failure is not the exact downgrade tuple", "error")
		if terminalErr != nil {
			return BootstrapDecision{Terminal: terminalErr}
		}
		return BootstrapDecision{Terminal: terminal}
	}
	if exitStatus != 6 {
		terminal, terminalErr := failViolation("bootstrap downgrade response carries a nonmatching exit status", "exit_status")
		if terminalErr != nil {
			return BootstrapDecision{Terminal: terminalErr}
		}
		return BootstrapDecision{Terminal: terminal}
	}
	next, ok := NextLowerMajor(want.Major)
	if !ok {
		terminal, terminalErr := failDowngrade("no lower locally supported major remains", "protocol_version")
		if terminalErr != nil {
			return BootstrapDecision{Terminal: terminalErr}
		}
		return BootstrapDecision{Terminal: terminal}
	}
	return BootstrapDecision{Downgrade: true, NextMajor: next}
}
