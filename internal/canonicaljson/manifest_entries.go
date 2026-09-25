package canonicaljson

// CheckManifestEntries validates an encoded Section 10.4 entries array using
// the same owner as immutable Transfer Manifest identity validation. Capture
// uses this before root/child manifests have been assembled; this checks only
// the entry set, not blob availability, filesystem containment, or closure.
func CheckManifestEntries(input []byte) error {
	value, err := decodeStrict(input)
	if err != nil {
		return err
	}
	values, err := requireArray(map[string]any{"entries": value}, "entries", 65536)
	if err != nil {
		return err
	}
	return validateManifestEntries(values)
}
