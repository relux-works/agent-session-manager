package tmuxserver

import (
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// RealmAdmission is the landed admission verdict over the exact tmux
// server: the Reconcile/ResolveEvidence Admitted set plus the AX-known
// raw generation the admission binds to. Production probes report this
// verdict; tests model probe reports. This leaf never re-decides what
// the admission already proves — signature verification, liveness,
// expiry, and the sentinel/smoke requirement coverage are the landed
// terminalbackend owner's verdict, consumed here through Has — and
// composes it only with the generation equality the wrapper owner
// (internal/axpane checkRealm) already enforces: the admission's raw
// generation must equal the request's typed generation, or the proof
// is stale and refuses.
//
// There is deliberately no member for a cached sentinel result, a
// managername observation, or any attested-server boolean: none
// authorizes, so none is an input. A cached observation replayed as a
// live one is exactly a stale-generation admission, and it refuses on
// the generation arm.
type RealmAdmission struct {
	// Admitted is the landed admission verdict.
	Admitted terminalbackend.Admitted
	// RawGeneration is the AX-known raw generation the admission
	// binds to.
	RawGeneration string
}

// CheckServerAttested admits exactly an attested dedicated server:
// the landed admission holds the credential_capable_execution_realm
// row AND the admission's raw generation equals the request's typed
// generation. Each arm refuses alone: a missing row refuses even with
// a bound generation, and a wrong generation refuses even with an
// admitted row, so weakening either arm admits a server the committed
// tests pin.
func CheckServerAttested(admission RealmAdmission, wantGeneration string) error {
	if !admission.Admitted.Has(realmCapability) {
		return &Error{Code: CodeReadinessNotAuthorizing, Detail: "server attestation"}
	}
	if wantGeneration == "" || admission.RawGeneration != wantGeneration {
		return &Error{Code: CodeReadinessNotAuthorizing, Detail: "server generation"}
	}
	return nil
}

// BrokerPrincipal is the authenticated Aqua broker identity the
// background path decides over. Per Section 4.4 the broker exposes
// only authenticated, same-user, generation-bound terminal readiness,
// so the principal carries the authenticated UID and the generation
// its readiness binds to as typed, bound facts — never a bare
// boolean. The UID is compared against the process identity, and the
// generation against the request's typed generation.
type BrokerPrincipal struct {
	// UID is the broker's authenticated user identity.
	UID uint32
	// Generation is the generation the broker readiness binds to.
	Generation string
}

// BrokerReport is the live Aqua broker report the background path
// decides over: the authenticated principal plus the attested AX tmux
// server admission. Either fact missing means neither exists for the
// purpose of the broker-or-refuse rule.
type BrokerReport struct {
	Principal BrokerPrincipal
	Server    RealmAdmission
}

// CheckBrokerContact admits exactly the background MAY case: a
// same-user authenticated broker principal AND a generation-bound
// principal AND the attested AX tmux server admission bound to the
// same typed generation. Every other report — a foreign-user broker,
// an unbound principal, an unattested server, a stale admission —
// refuses, and the caller maps the refusal to the typed
// capability_unavailable the spec names.
func CheckBrokerContact(report BrokerReport, wantGeneration string, currentUID uint32) error {
	if report.Principal.UID != currentUID {
		return &Error{Code: CodeReadinessNotAuthorizing, Detail: "broker authentication"}
	}
	if report.Principal.Generation == "" || report.Principal.Generation != wantGeneration {
		return &Error{Code: CodeReadinessNotAuthorizing, Detail: "broker generation"}
	}
	return CheckServerAttested(report.Server, wantGeneration)
}
