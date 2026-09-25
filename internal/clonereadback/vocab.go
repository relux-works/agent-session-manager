package clonereadback

// This file owns the two closed vocabularies the read-back
// contracts state: the manifest mode (staged|live) and the evidence
// row kind (native_sample|parser_trace|marker_observation),
// SPEC v0.7.0 Section 13.14.2 lines 10737-10754. Every other
// vocabulary these shapes touch (platforms, severities, digests,
// extension keys) stays with its landed owner.

var readBackModes = map[string]bool{
	"staged": true,
	"live":   true,
}

// ValidReadBackMode reports whether the mode is one of the two
// closed read-back modes. The mode additionally binds the trusted
// read authority at both entries, so a relabeled mode is an
// authority mismatch, never a second reading of the same
// manifest.
func ValidReadBackMode(mode string) bool {
	return readBackModes[mode]
}

// ReadBackModes lists the closed read-back modes in a fixed order.
func ReadBackModes() []string {
	return []string{"staged", "live"}
}

var evidenceKinds = map[string]bool{
	"native_sample":      true,
	"parser_trace":       true,
	"marker_observation": true,
}

// ValidEvidenceKind reports whether the kind is one of the three
// closed evidence row kinds.
func ValidEvidenceKind(kind string) bool {
	return evidenceKinds[kind]
}

// EvidenceKinds lists the closed evidence row kinds in a fixed
// order.
func EvidenceKinds() []string {
	return []string{"native_sample", "parser_trace", "marker_observation"}
}
