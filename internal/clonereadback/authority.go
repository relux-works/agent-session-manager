package clonereadback

// This file binds the read-back mode to the trusted read authority
// (SPEC v0.7.0 lines 3796-3797: ReadAuthority carries
// purpose:source_native|target_staged|target_live; Section 13.14.2
// lines 10753-10754: "Staged and live manifests are distinct and
// cannot be relabeled."; line 10580: "Modes cannot be relabeled.").
// The mode is DERIVED from the authority purpose the caller threads
// from its authority chain — never parsed from the manifest being
// validated. A resealed manifest carrying the opposite mode refuses
// even though its self-digest recomputes, because the digest alone
// does not carry the stage. source_native grants no target
// read-back and refuses at every entry. Authority decoding and
// freshness stay with the sessadapter owner and the caller chain;
// these entries consume the Purpose of an already-validated
// sessadapter.ReadAuthority only.

// modeForAuthorityPurpose derives the expected read-back mode from
// a trusted read authority purpose: target_staged seals staged,
// target_live seals live. source_native is a source purpose, never
// a target read-back, and any other purpose is outside the closed
// vocabulary. Both read-back entries share this gate. The purpose
// names repeat here because no exported owner predicate defines
// the purpose-to-mode binding: sessadapter owns authority decoding
// (DecodeReadAuthority, called where the chain mints the struct),
// and this package owns the binding this leaf's invariant needs.
func modeForAuthorityPurpose(purpose string) (string, error) {
	switch purpose {
	case "target_staged":
		return "staged", nil
	case "target_live":
		return "live", nil
	case "source_native":
		return "", invalid("read-back evidence manifest read authority purpose source_native grants no target read-back")
	default:
		return "", invalid("read-back evidence manifest read authority purpose %q is outside source_native|target_staged|target_live", purpose)
	}
}

// checkModeAuthority enforces that the claimed mode equals the mode
// derived from the trusted read authority: a claim the authority
// does not grant is a relabel, never a second reading. Both
// read-back entries share this gate.
func checkModeAuthority(owner, claimed, expected, purpose string) error {
	if claimed != expected {
		return invalid("%s mode %q disagrees with read authority purpose %q", owner, claimed, purpose)
	}
	return nil
}
