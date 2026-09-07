package provider

import (
	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// Lift carries one discovery or trust refusal across the seam into a
// wire-ready Structured Error. It is the only production path from a
// provider.Error to the wire, and the composition contract of package
// doc is enforced here rather than stated there:
//
//   - The wire message is rebuilt from the stable code alone. Detail,
//     which deliberately names provider IDs, canonical paths, and owner
//     identities for host-local diagnostics, never crosses: the three
//     messages below are static text with no interpolation point.
//   - The filesystem cause crosses only as the local Go cause, kept for
//     errors.Is and errors.As and structurally unreachable from the
//     encoded object. Because the message cannot reproduce the rendered
//     cause, axerror's causal-leak gate admits what Lift builds and
//     refuses the naive shape (message from Error(), cause attached).
//   - The code crosses verbatim: invalid_config,
//     local_precondition_failed, and integrity_failure are registered in
//     Structured Error 1.0.0, the version a pre-protocol host failure
//     carries. An Error carrying any other code is failed closed and
//     never produces a wire object, in two sub-arms: a code the registry
//     does not carry is refused by axerror and Lift returns that
//     refusal; a registered code with no lift arm (today: any registered
//     code outside the three above, e.g. not_found) is handed back as
//     the error itself, so the caller keeps the local diagnostic without
//     minting a wire object under a generic message it never reviewed.
//     A generic message would carry the code's exit status without the
//     code's meaning, which is fail-open, not fail-closed.
//
// Lift builds no refusal of its own: every error it returns is axerror's
// or the input failure itself, so the provider refusal inventory (four
// constructors, closed code set) is unchanged by this file — it holds no
// Error literal and no errors.New, fmt.Errorf, or panic site.
func Lift(failure Error) (*axerror.Error, error) {
	var message string
	switch failure.code {
	case codeInvalidConfig:
		message = "provider discovery refused the request: invalid configuration"
	case codeLocalPrecondition:
		message = "provider discovery could not establish its filesystem preconditions"
	case codeIntegrityFailure:
		message = "provider trust receipt no longer matches the filesystem"
	default:
		if _, err := axerror.ExitCodeFor(axerror.Version100, axerror.Code(failure.code)); err != nil {
			return nil, err
		}
		return nil, failure
	}
	return axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    axerror.Code(failure.code),
		Message: message,
		Details: axerror.Details{},
		Cause:   failure,
	})
}
