package axerror

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestMinimumRedactableCauseIsPinnedAtItsBoundary narrows the sensitivity bound
// of the causal-leak gate.
//
// minRedactableCause decides which causes are scanned at all: a cause shorter
// than the bound is skipped, because a cause rendering as "EOF" would otherwise
// refuse every message containing that substring. Before this test every cause
// in the corpus was at least 64 characters, so raising the constant to 16 or to
// 64 left the suite green and only 4096 reddened - the bound was proved by
// deleting the gate rather than by narrowing it, and an edit widening it to 63
// would have silently stopped scanning a whole class of short causes.
//
// The two rows below are one character apart on either side of the constant, so
// moving it in either direction reddens exactly one of them.
func TestMinimumRedactableCauseIsPinnedAtItsBoundary(t *testing.T) {
	if minRedactableCause != 8 {
		t.Fatalf("minRedactableCause = %d; this test pins the boundary at 8", minRedactableCause)
	}
	atBound := strings.Repeat("s", minRedactableCause)
	belowBound := strings.Repeat("s", minRedactableCause-1)
	if utf8.RuneCountInString(atBound) != minRedactableCause ||
		utf8.RuneCountInString(belowBound) != minRedactableCause-1 {
		t.Fatalf("the boundary fixtures are not %d and %d characters", minRedactableCause, minRedactableCause-1)
	}

	// A cause exactly at the bound is scanned, so a message reproducing it is
	// refused. Widening the constant by one admits this and reddens here.
	_, err := New(Spec{
		Version: Version100,
		Code:    "provider_process_failed",
		Message: "the provider exited: " + atBound,
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   errors.New(atBound),
	})
	if !errors.Is(err, ErrCausalLeak) {
		t.Fatalf("a cause of exactly %d characters was not scanned: %v", minRedactableCause, err)
	}

	// A cause one character below the bound is skipped, so a message
	// reproducing it is admitted. Narrowing the constant by one refuses this
	// and reddens here.
	failure, err := New(Spec{
		Version: Version100,
		Code:    "provider_process_failed",
		Message: "the provider exited: " + belowBound,
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   errors.New(belowBound),
	})
	if err != nil {
		t.Fatalf("a cause of %d characters was scanned: %v", minRedactableCause-1, err)
	}
	if failure == nil {
		t.Fatalf("New returned no failure and no error")
	}

	// The same boundary governs a diagnostic value, not only the human text.
	_, err = New(Spec{
		Version: Version100,
		Code:    "provider_process_failed",
		Message: "the provider exited",
		IDs:     NoIDs(),
		Details: Details{"provider_stderr": atBound},
		Cause:   errors.New(atBound),
	})
	if !errors.Is(err, ErrCausalLeak) {
		t.Fatalf("a detail reproducing a cause at the bound was admitted: %v", err)
	}
	if _, err := New(Spec{
		Version: Version100,
		Code:    "provider_process_failed",
		Message: "the provider exited",
		IDs:     NoIDs(),
		Details: Details{"provider_stderr": belowBound},
		Cause:   errors.New(belowBound),
	}); err != nil {
		t.Fatalf("a detail reproducing a cause below the bound was refused: %v", err)
	}

	// The bound is measured in UTF-8 characters, not bytes: a cause of
	// minRedactableCause multi-byte characters is scanned even though a
	// byte-counted bound would already have admitted it at half that length.
	multiByte := strings.Repeat("é", minRedactableCause)
	if _, err := New(Spec{
		Version: Version100,
		Code:    "provider_process_failed",
		Message: "the provider exited: " + multiByte,
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   errors.New(multiByte),
	}); !errors.Is(err, ErrCausalLeak) {
		t.Fatalf("a multi-byte cause at the character bound was not scanned: %v", err)
	}
}

// TestShortCausesInAWrappedChainAreSkippedIndependently proves the bound is
// applied per link rather than to the outermost rendering only: a long outer
// cause is scanned while its short inner link is not, so the "EOF" case the
// bound exists for is still skipped inside a chain.
func TestShortCausesInAWrappedChainAreSkippedIndependently(t *testing.T) {
	inner := errors.New("EOF")
	outer := errors.Join(inner, errors.New(strings.Repeat("o", 64)))

	// The short inner link is skipped: a message containing "EOF" is admitted.
	if _, err := New(Spec{
		Version: Version100,
		Code:    "transport_failure",
		Message: "the peer closed the connection at EOF",
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   outer,
	}); err != nil {
		t.Fatalf("a short wrapped link was scanned: %v", err)
	}

	// The long link in the same chain is scanned.
	if _, err := New(Spec{
		Version: Version100,
		Code:    "transport_failure",
		Message: "the peer closed the connection: " + strings.Repeat("o", 64),
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   outer,
	}); !errors.Is(err, ErrCausalLeak) {
		t.Fatalf("a long wrapped link was not scanned")
	}
}
