package clonereconcile

import (
	"errors"
	"fmt"
)

// ErrInvalid is the refusal for every irreconcilable input: a
// missing, duplicated, or double-counted link at any tier, a row
// whose source evidence is not its own candidate's tier-1 raw,
// target evidence that does not resolve into the sealed reads, a
// synthesized row claiming source evidence, a tuple/native/pairing
// mismatch, or an owner-delegated shape refusal. Callers match with
// errors.Is; the message names the tier and the offending key.
var ErrInvalid = errors.New("clonereconcile: irreconcilable clone evidence")

func refuse(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
