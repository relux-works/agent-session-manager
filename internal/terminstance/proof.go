package terminstance

// ProviderProofKind is the closed §4.C ProviderProofKind enum.
type ProviderProofKind string

// Provider proof kinds.
const (
	ProofProviderQuiescence   ProviderProofKind = "provider_quiescence"
	ProofProviderProcessExit  ProviderProofKind = "provider_process_exit"
	ProofAXCheckpointBoundary ProviderProofKind = "ax_checkpoint_boundary"
)

// ParseProviderProofKind admits exactly the three §4.C proof kinds.
func ParseProviderProofKind(value string) (ProviderProofKind, error) {
	switch ProviderProofKind(value) {
	case ProofProviderQuiescence, ProofProviderProcessExit, ProofAXCheckpointBoundary:
		return ProviderProofKind(value), nil
	default:
		return "", refuse(CodeProtocolError, "provider proof vocabulary")
	}
}

// RequiresProviderObservation reports whether the proof kind is a provider
// kind: provider_quiescence and provider_process_exit observe the provider
// process and additionally require the provider_process_observation
// capability, while ax_checkpoint_boundary is an AX-side boundary and
// requires only safe_boundary_observation.
func RequiresProviderObservation(kind ProviderProofKind) bool {
	switch kind {
	case ProofProviderQuiescence, ProofProviderProcessExit:
		return true
	default:
		return false
	}
}
