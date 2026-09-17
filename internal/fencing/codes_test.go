package fencing

import (
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// TestGateRefusalsUseRegisteredCodes proves every fencing refusal
// carries a Section 15 registered code and nothing else: each
// sentinel message constructs a Structured Error with its exact
// pinned exit. A gate that invented a code outside the taxonomy
// fails here.
func TestGateRefusalsUseRegisteredCodes(t *testing.T) {
	vectors := map[string]struct {
		sentinel error
		exit     int
	}{
		"lease_conflict":            {ErrLeaseConflict, 10},
		"not_owner":                 {ErrNotOwner, 10},
		"stale_owner":               {ErrStaleOwner, 10},
		"invalid_arguments":         {ErrInvalidArguments, 2},
		"local_precondition_failed": {ErrLocalPreconditionFailed, 3},
	}
	for name, vector := range vectors {
		t.Run(name, func(t *testing.T) {
			failure, err := axerror.New(axerror.Spec{
				Version: axerror.Version100,
				Code:    axerror.Code(vector.sentinel.Error()),
				Message: "fencing refusal registration probe",
				IDs:     axerror.NoIDs(),
				Details: axerror.Details{},
			})
			if err != nil {
				t.Fatalf("axerror.New(%s) error = %v, want a registered code", name, err)
			}
			if failure.ExitCode() != vector.exit {
				t.Fatalf("axerror.New(%s) exit = %d, want %d", name, failure.ExitCode(), vector.exit)
			}
		})
	}
}

// TestParkDecisionCarriesSection15Cause proves a park decision is not
// a new error class: every parked refusal unwraps to its underlying
// registered cause alongside the parked marker.
func TestParkDecisionCarriesSection15Cause(t *testing.T) {
	observation := fenceObservation()
	observation.Winner.HolderHostID = fenceHostC
	_, err := AuthorizeActivation(fencePresented(), observation)
	if !IsParked(err) {
		t.Fatalf("remote activation error = %v, want a park decision", err)
	}
	mustRefuseCause(t, err, ErrNotOwner)
}

// mustRefuseCause fails unless err carries the sentinel through
// errors.Is (park decisions included).
func mustRefuseCause(t *testing.T, err error, sentinel error) {
	t.Helper()
	if err == nil {
		t.Fatalf("succeeded, want cause %v", sentinel)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is %v", err, sentinel)
	}
}
