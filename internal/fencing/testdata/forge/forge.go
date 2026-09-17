// Package forge attempts to assemble a fencing LeaseToken by field
// name from outside the package. It must NOT compile: the seal is
// unexported. The census test builds this fixture expecting failure
// and builds the sibling use fixture expecting success.
package forge

import (
	"github.com/relux-works/agent-session-manager/internal/fencing"
)

// Forge assembles a LeaseToken the only way an outsider can spell,
// which the compiler refuses.
func Forge() fencing.LeaseToken {
	return fencing.LeaseToken{seal: nil}
}
