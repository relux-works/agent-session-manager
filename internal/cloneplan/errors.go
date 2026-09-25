package cloneplan

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every malformed projection input:
// unknown or missing members, out-of-vocabulary values, broken
// bounds, unsorted or duplicate order, branch violations, DAG
// violations, constant disagreement, and digest disagreement.
// Callers match with errors.Is; the message names the offending
// member.
var ErrInvalid = errors.New("cloneplan: invalid projection object")

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
