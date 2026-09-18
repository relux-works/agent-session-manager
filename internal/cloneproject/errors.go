package cloneproject

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every malformed projection input and
// every fidelity violation: undecodable manifests, unstable or
// mismatched capture closure, missing or disagreeing blobs,
// malformed native framing, contradictory protection claims, live
// authority claimed by captured history, and ambiguous tool
// pairings. Callers match with errors.Is; the message names the
// offending member, record, or blob.
var ErrInvalid = errors.New("cloneproject: invalid projection")

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
