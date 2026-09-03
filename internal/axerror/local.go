package axerror

import (
	"errors"
	"fmt"
	"sort"
)

// ErrUnknownSurface reports a surface for which the pinned document states no
// exact local failure mapping. It is returned instead of a plausible default,
// because inventing a mapping here would put a code on the wire that no clause
// assigns.
var ErrUnknownSurface = errors.New("no pinned local failure mapping for surface")

// Surface names a boundary at which ax reads output from a child process or a
// peer. Only the surfaces whose local mapping the pinned document states
// exactly are represented. Directory Node is deliberately absent: Section 15.3
// says the caller emits "incompatible_protocol or
// adapter_protocol_violation/transport_failure as applicable" without fixing
// which, so this package reports the mapping as unknown rather than choosing
// one and presenting the choice as the contract.
type Surface string

const (
	// SurfaceProviderStdio is the Provider JSON-over-stdio row of the Section
	// 15.1 bootstrap table.
	SurfaceProviderStdio Surface = "provider_stdio"
	// SurfaceTaskBoardBridge is the task-board bridge facade row.
	SurfaceTaskBoardBridge Surface = "task_board_bridge"
	// SurfaceMeshRPC is the Mesh RPC row.
	SurfaceMeshRPC Surface = "mesh_rpc"
	// SurfaceTerminalBackend is the Section 15.3 TerminalBackend paragraph.
	SurfaceTerminalBackend Surface = "terminal_backend"
)

// UntrustedOutcome classifies what ax observed at the boundary. It is a
// classification of the local read, not a value taken from the payload: the
// payload is never parsed far enough to yield one.
type UntrustedOutcome string

const (
	// OutcomeRecognizableMajorMismatch is the "recognizable major mismatch"
	// branch of every row.
	OutcomeRecognizableMajorMismatch UntrustedOutcome = "recognizable_major_mismatch"
	// OutcomeUnusableFrame is the otherwise branch: invalid JSON, an oversize
	// line, missing framing identity, or output that cannot be framed.
	OutcomeUnusableFrame UntrustedOutcome = "unusable_frame"
)

type localMapping struct {
	version       Version
	majorMismatch Code
	otherwise     Code
}

// localMappings is the Section 15.1 bootstrap table plus the Section 15.3
// TerminalBackend paragraph, each row quoted in the comment above its entry.
// The version is fixed per surface and is not a caller argument, so a local
// failure cannot be emitted under a version the surface does not bind.
var localMappings = map[Surface]localMapping{
	// "the host accepts no child error object ... and emits its own local Error
	// 1.0.0: incompatible_protocol for a recognizable major mismatch, otherwise
	// provider_protocol_error".
	SurfaceProviderStdio: {version: Version100, majorMismatch: "incompatible_protocol", otherwise: "provider_protocol_error"},
	// "ax emits its own local Error 1.0.0: incompatible_protocol for a
	// recognizable major mismatch, otherwise task_board_bridge_unavailable".
	SurfaceTaskBoardBridge: {version: Version100, majorMismatch: "incompatible_protocol", otherwise: "task_board_bridge_unavailable"},
	// "The initiator emits its own local Error 1.0.0: incompatible_protocol for
	// a recognizable major mismatch, otherwise transport_failure".
	SurfaceMeshRPC: {version: Version100, majorMismatch: "incompatible_protocol", otherwise: "transport_failure"},
	// "the AX caller emits a local Error 1.3 terminal_backend_protocol_incompatible
	// for a recognizable major mismatch and terminal_backend_protocol_error otherwise".
	SurfaceTerminalBackend: {version: Version130, majorMismatch: "terminal_backend_protocol_incompatible", otherwise: "terminal_backend_protocol_error"},
}

// LocalFromUntrusted builds the ax-local failure for an unsupported major or an
// unusable first frame at a child or peer boundary.
//
// It takes no part of the foreign payload. Its inputs are the surface, ax's own
// classification of what it observed, human text ax wrote, the identifiers ax
// itself knows, and a local Go cause. There is no parameter through which a
// foreign code, retryable bit, detail map, or authority field could be adopted,
// which is how Section 15.1's "Receivers MUST NOT parse a different major's
// payload far enough to trust its error code, retryable bit, details, or
// authority fields" is enforced rather than merely intended.
//
// The result is never retryable. Every code this table can emit either belongs
// to a class the caller must inspect before repeating, or describes a boundary
// that the identical request cannot get past; nothing observed at an unusable
// frame licenses a safe-retry claim, so the constructor does not offer one.
func LocalFromUntrusted(
	surface Surface,
	outcome UntrustedOutcome,
	message string,
	ids IDs,
	details Details,
	cause error,
) (*Error, error) {
	mapping, known := localMappings[surface]
	if !known {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSurface, surface)
	}
	var code Code
	switch outcome {
	case OutcomeRecognizableMajorMismatch:
		code = mapping.majorMismatch
	case OutcomeUnusableFrame:
		code = mapping.otherwise
	default:
		return nil, fmt.Errorf("%w: outcome %q is not a pinned classification", ErrInvalidStructuredError, outcome)
	}
	if details == nil {
		details = Details{}
	}
	return New(Spec{
		Version: mapping.version,
		Code:    code,
		Message: message,
		IDs:     ids,
		Details: details,
		Cause:   cause,
	})
}

// LocalSurfaces returns the surfaces with a pinned local failure mapping, in a
// stable order. It is the measured denominator of this package's bootstrap
// coverage: four of the five boundaries the pinned document names, with
// Directory Node excluded for the reason stated on Surface.
func LocalSurfaces() []Surface {
	result := make([]Surface, 0, len(localMappings))
	for surface := range localMappings {
		result = append(result, surface)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}
