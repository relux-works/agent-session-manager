package clonefidelity

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every malformed fidelity input:
// unknown or missing members, out-of-vocabulary values, broken
// bounds, unsorted or duplicate order, branch violations, aggregate
// disagreement, and digest disagreement. Callers match with
// errors.Is; the message names the offending member.
var ErrInvalid = errors.New("clonefidelity: invalid fidelity object")

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
