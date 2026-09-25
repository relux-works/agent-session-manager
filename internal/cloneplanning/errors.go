package cloneplanning

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every malformed planning input:
// unknown strategies, profiles, item classes, or fact values,
// archive-branch target planning, strict-exact non-exact items,
// and malformed visible-projection inputs. Callers match with
// errors.Is; the message names the offending member.
var ErrInvalid = errors.New("cloneplanning: invalid projection planning input")

func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
