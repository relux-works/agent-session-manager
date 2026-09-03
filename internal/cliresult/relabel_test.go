package cliresult

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// relabelledFailureDocument builds a Structured Error document naming code but
// declaring exitStatus, which is the whole attack: the code and the number that
// decides its class are two separate members, and only one guard requires them
// to agree.
func relabelledFailureDocument(t *testing.T, version axerror.Version, code axerror.Code, exitStatus int, retryable bool) []byte {
	t.Helper()

	encoded, err := json.Marshal(map[string]any{
		"schema":         axerror.Schema,
		"schema_version": string(version),
		"code":           string(code),
		"message":        "relabelled by the exit-class bypass measurement",
		"exit_code":      exitStatus,
		"retryable":      retryable,
		"details":        map[string]any{},
	})
	if err != nil {
		t.Fatalf("marshal %s %q at exit %d: %v", version, code, exitStatus, err)
	}
	return encoded
}

// TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead drives
// the bypass the round-5 review measured, through the production entry point a
// machine client actually calls.
//
// The shape. This package publishes a retryability refusal that keys on the
// exit status for three whole Section 15.2 classes - 7 authorization, 16 policy
// refusal, 130 operator interrupt - because the meaning of those classes is that
// the identical request cannot succeed without new authority or confirmation.
// The code and the exit status are two separate members of one document. Move
// the exit_code of an authorization failure to a class that permits retry, and
// the exit-keyed refusal is asked about the wrong class and stays silent.
//
// The only thing standing in the way is the code-to-exit-status agreement in
// axerror.decodeBody. With that guard narrowed to the single code its previous
// coverage drove, the reviewer's document
//
//	{"code":"authentication_failed", "exit_code":9, "retryable":true, ...}
//
// was ADMITTED by Read at exit status 9, and the caller received a Reading whose
// Code() names an authorization failure and whose Retryable() is true. The whole
// repository suite stayed green.
//
// This test sweeps the shape rather than pinning the one document. For every
// 1.0.0 code whose own class carries an exit-keyed retryability refusal, it
// relabels the document to every failure status that carries none, and requires
// Read to refuse each one with the agreement guard's own sentence. The control
// in the other direction is required too: the same code at its own exit status
// with retryable: true must be refused by the retryability gate, which is what
// makes the relabelling a bypass of something rather than a rewrite of nothing.
func TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead(t *testing.T) {
	t.Parallel()

	// CommandList binds CLI Result 1.0.0 and, through the static Section 15.1
	// table, Structured Error 1.0.0. Read resolves that itself; the version here
	// is only used to enumerate the registry the same way Read will read it.
	const invoked = CommandList
	const version = axerror.Version100

	groups, err := axerror.CodesByExitStatus(version)
	if err != nil {
		t.Fatalf("CodesByExitStatus(%s): %v", version, err)
	}

	// The classes whose refusal the relabelling disarms, and the destinations
	// that carry no exit-keyed refusal to disarm. Both are derived by asking
	// the production predicate rather than by restating the three numbers.
	var forbiddenClasses, permittedDestinations []int
	for status := range groups {
		if _, forbidden := axerror.RetryabilityRefusal("", status); forbidden {
			forbiddenClasses = append(forbiddenClasses, status)
			continue
		}
		permittedDestinations = append(permittedDestinations, status)
	}
	sort.Ints(forbiddenClasses)
	sort.Ints(permittedDestinations)
	if len(forbiddenClasses) != 3 {
		t.Fatalf("the exit-keyed retryability refusal covers %d classes %v of the %d %s registers, want 3",
			len(forbiddenClasses), forbiddenClasses, len(groups), version)
	}
	if len(permittedDestinations) == 0 {
		t.Fatal("no failure status permits a retry claim, so the relabelling has no destination and " +
			"this test would measure nothing")
	}

	refusedByTheAgreement, refusedByTheRetryGate, skippedCodeKeyed := 0, 0, 0
	for _, class := range forbiddenClasses {
		for _, code := range groups[class] {
			// A code the pinned document disqualifies by name keeps its refusal
			// wherever its exit_code is moved to, so it is not a witness for
			// this bypass. It is counted rather than silently dropped.
			if _, byName := axerror.RetryabilityRefusal(code, 0); byName {
				skippedCodeKeyed++
				continue
			}

			// Control: at its own class the retryability gate is the refusal.
			honest := relabelledFailureDocument(t, version, code, class, true)
			_, err := Read(invoked, InvocationOutput{Stdout: honest, ExitStatus: class})
			if !errors.Is(err, axerror.ErrInvalidStructuredError) {
				t.Fatalf("Read(%q at its own exit %d, retryable true) error = %v, want ErrInvalidStructuredError",
					code, class, err)
			}
			if !strings.Contains(err.Error(), "may not claim retryable") {
				t.Fatalf("Read(%q at its own exit %d, retryable true) was refused by something other "+
					"than the retryability gate: %v", code, class, err)
			}
			refusedByTheRetryGate++

			// The bypass: the same claim, relabelled into a class that permits
			// it. The process really does exit with the relabelled status, so
			// readFailure's exit_code equality corroborates the forgery instead
			// of catching it.
			for _, destination := range permittedDestinations {
				forged := relabelledFailureDocument(t, version, code, destination, true)
				reading, err := Read(invoked, InvocationOutput{Stdout: forged, ExitStatus: destination})
				if err == nil {
					retryable := reading.Retryable()
					readCode, _ := reading.Code()
					t.Fatalf("Read admitted %q relabelled from exit %d to exit %d: Code() = %q, "+
						"Retryable() = %v", code, class, destination, readCode, retryable)
				}
				if !errors.Is(err, axerror.ErrInvalidStructuredError) {
					t.Fatalf("Read(%q relabelled %d -> %d) error = %v, want ErrInvalidStructuredError",
						code, class, destination, err)
				}
				sentence := fmt.Sprintf("maps to exit %d, document carries %d", class, destination)
				if !strings.Contains(err.Error(), sentence) {
					t.Fatalf("Read(%q relabelled %d -> %d) was refused by something other than the "+
						"code-to-exit-status agreement: %v", code, class, destination, err)
				}
				refusedByTheAgreement++
			}
		}
	}

	// The measured ratio, re-derived from the projection rather than restated,
	// so a code leaving or joining one of the three classes moves the figure
	// instead of leaving a smaller sweep passing quietly.
	witnesses := 0
	for _, class := range forbiddenClasses {
		witnesses += len(groups[class])
	}
	witnesses -= skippedCodeKeyed
	if witnesses == 0 {
		t.Fatalf("%s registers no code in the %d exit-keyed refusal classes %v, so this test drove nothing",
			version, len(forbiddenClasses), forbiddenClasses)
	}
	if refusedByTheRetryGate != witnesses {
		t.Fatalf("the retryability gate refused %d of %d witness codes at their own class",
			refusedByTheRetryGate, witnesses)
	}
	if refusedByTheAgreement != witnesses*len(permittedDestinations) {
		t.Fatalf("the agreement guard refused %d relabellings, want %d witnesses x %d permitted destinations = %d",
			refusedByTheAgreement, witnesses, len(permittedDestinations),
			witnesses*len(permittedDestinations))
	}
	t.Logf("%s: %d witness codes in the %d exit-keyed refusal classes %v (%d skipped as code-keyed), "+
		"relabelled into %d permitted destinations %v: %d refusals by the agreement guard",
		version, witnesses, len(forbiddenClasses), forbiddenClasses, skippedCodeKeyed,
		len(permittedDestinations), permittedDestinations, refusedByTheAgreement)
}
