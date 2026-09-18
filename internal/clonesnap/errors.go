package clonesnap

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every malformed capture input and
// every detected source violation: unsanitized or unplanned members,
// containment escapes, excluded-class admission, digest disagreement,
// undecided races, and target-branch refusals. Callers match with
// errors.Is; the message names the offending member.
var ErrInvalid = errors.New("clonesnap: invalid capture")

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
