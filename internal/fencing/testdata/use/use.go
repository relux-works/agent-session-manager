// Package use exercises only the public fencing surface: authorize
// through a production entry, then bind the minted token. It must
// compile: the positive control for the forgery fixture.
package use

import (
	"github.com/relux-works/agent-session-manager/internal/fencing"
)

// BindQuiesce authorizes a caller-supplied observation and binds the
// minted token to quiesce. It is the compile control, not a gate.
func BindQuiesce(presented fencing.PresentedToken, observation fencing.Observation) (fencing.ProviderLease, error) {
	token, err := fencing.AuthorizeMutation(presented, observation)
	if err != nil {
		return fencing.ProviderLease{}, err
	}
	return token.Bind(fencing.ProviderQuiesce)
}
