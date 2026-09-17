package hosttrust

// aceKind classifies one DACL entry for the owner-only audit. Only explicit
// allow entries grant access; deny entries grant nothing; any other entry
// type has unknown grant semantics and fails closed.
type aceKind int

const (
	aceAllow aceKind = iota
	aceDeny
	aceOther
)

// allowedACE is one DACL entry in platform-independent form: the grant kind
// plus the trustee SID in string form. Platform extractors translate native
// ACEs into this shape; ownerOnlyGrants decides them without native calls,
// so the audit logic itself runs on every platform under test.
type allowedACE struct {
	kind aceKind
	sid  string
}

// ownerOnlyGrants reports whether a DACL grants access exclusively to owner:
// every allow entry names the owner SID, and at least one such entry exists
// (an empty or deny-only DACL is unusable and refuses). Deny entries are
// skipped: they grant nothing. Any other entry type refuses.
func ownerOnlyGrants(owner string, aces []allowedACE) bool {
	if owner == "" {
		return false
	}
	granted := false
	for _, ace := range aces {
		switch ace.kind {
		case aceDeny:
			continue
		case aceAllow:
			if ace.sid != owner {
				return false
			}
			granted = true
		default:
			return false
		}
	}
	return granted
}
