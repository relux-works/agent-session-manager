package hosttrust

import "errors"

var (
	// ErrTrustDecode reports JSON syntax or shape failures without
	// echoing machine-local bytes. Rendered messages stay static.
	ErrTrustDecode = errors.New("host trust document decode failed")
	// ErrTrustValidation reports a well-formed document that violates the
	// closed Host Trust Store 1.0.0 contract.
	ErrTrustValidation = errors.New("host trust document validation failed")
	// ErrTrustStoreMissing reports a missing store. It is never an empty
	// store: callers must run explicit setup instead of inferring one.
	ErrTrustStoreMissing = errors.New("host trust store is missing")
	// ErrTrustStoreUnreadable reports permission, ownership or I/O failures.
	ErrTrustStoreUnreadable = errors.New("host trust store is unreadable")
	// ErrTrustUnsafeCustody reports owner-only violations: group/world
	// access, foreign ownership, or symlink/reparse escapes.
	ErrTrustUnsafeCustody = errors.New("host trust custody is not owner-only")
	// ErrTrustDurability reports a failed atomic commit. A failed commit is
	// a failure, never absence or success.
	ErrTrustDurability = errors.New("host trust durable commit failed")
	// ErrGenerationExhausted reports a mutation refused because the uint53
	// generation counter cannot increment again.
	ErrGenerationExhausted = errors.New("host trust generation exhausted")
	// ErrStaleGeneration reports a dispatch or mutation attempted under a
	// superseded authorization generation.
	ErrStaleGeneration = errors.New("host trust authorization generation is stale")
	// ErrEnrollmentRefused reports an enrollment tuple the operator
	// authorization or the uniqueness rules reject.
	ErrEnrollmentRefused = errors.New("host trust enrollment refused")
	// ErrRotationRefused reports a rotation outside the fresh-key bounded
	// window: same-key renewal, a second rotation over a retiring entry, or
	// more than two non-revoked credentials for one host.
	ErrRotationRefused = errors.New("host credential rotation refused")
	// ErrRevocationRefused reports a revocation with nothing to revoke.
	ErrRevocationRefused = errors.New("host credential revocation refused")
	// ErrCredentialProfile reports issued or imported bytes outside Host
	// Credential Profile 1.
	ErrCredentialProfile = errors.New("host credential profile check failed")
	// ErrCredentialCustody reports private/public mismatch, wrong host
	// binding, or unreadable custody files at use time.
	ErrCredentialCustody = errors.New("host credential custody check failed")
	// ErrAuthorizationRefused reports a dispatch or mutation boundary that
	// fails current-generation, validity, allowlist or hello checks.
	ErrAuthorizationRefused = errors.New("host channel authorization refused")
	// ErrNotAllowlisted reports a TLS-authenticated peer that is not on the
	// local allowlist. The string is the spec refusal identity shared with
	// peeridentity.ErrNotAllowlisted; hosttrust cannot import peeridentity
	// (import cycle), so a test pins the two strings equal.
	ErrNotAllowlisted = errors.New("peer_not_allowlisted")
	// ErrHostIdentityMismatch reports a hello host ID that differs from the
	// uniquely verified enrolled host UUID. The string is the spec refusal
	// identity shared with peeridentity.ErrHostIdentityMismatch; pinned
	// equal by test for the same reason as ErrNotAllowlisted.
	ErrHostIdentityMismatch = errors.New("host_identity_mismatch")
	// ErrInvalidStoreContext reports a store rooted outside an absolute
	// state directory or otherwise unusable as machine-local authority.
	ErrInvalidStoreContext = errors.New("invalid host trust store context")
	// ErrJointReplacementDurable reports a joint commit whose external
	// replacement was durable but whose generation bump failed. The source
	// bytes are restored and the intent marker aborts under the same
	// exclusive hold, so the operation fails cleanly with no generation
	// change; a compensation that cannot complete instead joins
	// ErrJointCompensationFailed with the marker kept.
	ErrJointReplacementDurable = errors.New("host trust joint replacement durable without generation")
	// ErrJointCompensationFailed reports error compensation that could not
	// restore the source and abort the intent marker: the restore failed,
	// the marker or the configuration moved outside the confirmed pins, or
	// the marker is unreadable. The marker is kept for operator
	// resolution and the configuration stays at the replacement, which
	// restart recovery completes forward once the fault heals.
	ErrJointCompensationFailed = errors.New("host trust joint error compensation failed")
	// ErrJointIntervened reports a joint config-plus-trust commit that
	// cannot converge: the configuration or the generation moved outside
	// the confirmed pins while the intent marker was outstanding. The
	// marker is kept for operator resolution; the restart refuses.
	ErrJointIntervened = errors.New("host trust joint commit was superseded")
	// ErrExclusiveHoldRequired reports a config-plus-trust pair mutation
	// attempted without a live exclusive authorization hold: a forged or
	// absent token, or a token used after its hold released. The mutation
	// refuses with nothing written.
	ErrExclusiveHoldRequired = errors.New("host trust exclusive authorization hold required")
	// ErrExclusiveHoldResourceMismatch reports a live hold minted for a
	// different trust store or without the exact configuration path binding
	// required by a pair writer. Liveness alone never authorizes a mutation
	// against another resource.
	ErrExclusiveHoldResourceMismatch = errors.New("host trust exclusive authorization resource mismatch")
)

// TrustError renders a static clause without machine-local values, paths,
// fingerprints, UUIDs or key material. Wrapped details stay available to
// errors.Is/errors.As through Unwrap.
type TrustError struct {
	Operation string
	Err       error
}

func (err *TrustError) Error() string { return "host trust " + err.Operation + " failed" }
func (err *TrustError) Unwrap() error { return err.Err }

// trustError is the only construction site for *TrustError.
var trustError = func(value TrustError) error { return &value }
