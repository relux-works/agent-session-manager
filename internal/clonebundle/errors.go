package clonebundle

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every malformed clone-bundle input:
// unknown or missing members, out-of-vocabulary values, broken
// bounds, unsorted or duplicate order, digest disagreement, and
// excluded-path admission. Callers match with errors.Is; the message
// names the offending member.
var ErrInvalid = errors.New("clonebundle: invalid clone bundle object")

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
