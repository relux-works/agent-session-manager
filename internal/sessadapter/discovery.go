package sessadapter

import (
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file owns Session Adapter discovery. The adapter is not a
// separately discovered plugin: it runs in the same trusted
// ax-provider-<id> executable and under the same host-observed
// executable digest as Provider Protocol 2.0.0. Discovering it
// therefore means binding the adapter manifest the executable
// returned to the independently observed provider candidate —
// same provider identifier, same executable, same digests — and
// sealing that binding so every later call and target mutation can
// be checked against freshly read facts and the Journal binding.

// Role names one side of a SessionAdapterExecutionBinding. The role
// is call-scoped: a source binding never authorizes a target call.
type Role string

// The closed role vocabulary.
const (
	RoleSource Role = "source"
	RoleTarget Role = "target"
)

// roleNames is the closed role vocabulary as a table so the census
// derives it like every other closed vocabulary: Discover compares
// through validRole, never through an inline comparison chain the
// census cannot see. The table is built from the constants above,
// so the two spellings cannot drift.
var roleNames = []string{
	string(RoleSource),
	string(RoleTarget),
}

// validRole reports whether the role is one of the two closed
// binding roles.
func validRole(role Role) bool {
	for _, allowed := range roleNames {
		if string(role) == allowed {
			return true
		}
	}
	return false
}

// TrustedCandidate is the independently observed provider
// candidate the adapter must share: the identifier, kind,
// executable path, owner identity, and digests the host observed
// through provider discovery and trust, never through the adapter.
type TrustedCandidate struct {
	ProviderID             string
	Kind                   CandidateKind
	ExecutablePath         string
	OwnerIdentity          string
	ExecutableSHA256       string
	ProviderManifestDigest string
}

// ExecutionBinding is one sealed
// SessionAdapterExecutionBinding: role, provider, candidate kind,
// canonical executable path, owner identity, executable digest,
// provider and adapter manifest digests, and the observation time.
// verified_at records when the host observed these facts, not what
// they are: equality compares the eight identity facts, never the
// clock reading, because two reads of one unchanged candidate
// legitimately carry different timestamps.
type ExecutionBinding struct {
	Role                   Role
	ProviderID             string
	CandidateKind          CandidateKind
	ExecutablePath         string
	OwnerIdentity          string
	ExecutableSHA256       string
	ProviderManifestDigest string
	AdapterManifestDigest  string
	VerifiedAt             string
}

// Discover binds one validated adapter manifest to one trusted
// candidate: the manifest's provider identifier must equal the
// candidate's, and the host-computed manifest digest is sealed into
// the binding alongside the candidate's executable facts. A
// well-formed manifest for a different provider is an integrity
// failure, not a usable adapter: agreement with the trusted
// candidate is what makes the manifest this adapter's, and a
// failed or partial observation never means the candidate is
// absent.
func Discover(role Role, manifest Manifest, manifestDigest string, candidate TrustedCandidate) (ExecutionBinding, error) {
	if !validRole(role) {
		failure, err := failInvalid("discovery role is outside source|target", "role")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	if manifest.ProviderID != candidate.ProviderID {
		failure, err := failIntegrity("adapter manifest provider identifier does not match the trusted candidate", "provider_id")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	if candidate.ExecutablePath == "" {
		failure, err := failInvalid("discovery candidate carries no executable path", "executable_path")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	if candidate.OwnerIdentity == "" {
		failure, err := failInvalid("discovery candidate carries no owner identity", "owner_identity")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	if _, ok := checkDigestString(candidate.ExecutableSHA256); !ok {
		failure, err := failInvalid("discovery candidate executable digest is not a digest", "executable_sha256")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	if _, ok := checkDigestString(candidate.ProviderManifestDigest); !ok {
		failure, err := failInvalid("discovery candidate provider manifest digest is not a digest", "provider_manifest_digest")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	if _, ok := checkDigestString(manifestDigest); !ok {
		failure, err := failInvalid("discovery adapter manifest digest is not a digest", "session_adapter_manifest_digest")
		if err != nil {
			return ExecutionBinding{}, err
		}
		return ExecutionBinding{}, failure
	}
	return ExecutionBinding{
		Role:                   role,
		ProviderID:             candidate.ProviderID,
		CandidateKind:          candidate.Kind,
		ExecutablePath:         candidate.ExecutablePath,
		OwnerIdentity:          candidate.OwnerIdentity,
		ExecutableSHA256:       candidate.ExecutableSHA256,
		ProviderManifestDigest: candidate.ProviderManifestDigest,
		AdapterManifestDigest:  manifestDigest,
		VerifiedAt:             "",
	}, nil
}

// checkDigestString reports whether the value parses as a digest of
// the pinned form.
func checkDigestString(value string) (string, bool) {
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return "", false
	}
	return digest.String(), true
}

// CheckBindingEquality requires the sealed binding to equal freshly
// read trusted-candidate facts and the Journal binding on all
// eight identity facts. Before every call and every target
// mutation these facts MUST agree; a failed or partial read is an
// integrity failure, and no self-claim or publisher claim
// establishes trust. verified_at is compared by presence only
// when both sides carry it: it timestamps the observation and two
// observations of one unchanged candidate legitimately differ.
func CheckBindingEquality(sealed, fresh ExecutionBinding) error {
	if sealed.Role != fresh.Role ||
		sealed.ProviderID != fresh.ProviderID ||
		sealed.CandidateKind != fresh.CandidateKind ||
		sealed.ExecutablePath != fresh.ExecutablePath ||
		sealed.OwnerIdentity != fresh.OwnerIdentity ||
		sealed.ExecutableSHA256 != fresh.ExecutableSHA256 ||
		sealed.ProviderManifestDigest != fresh.ProviderManifestDigest ||
		sealed.AdapterManifestDigest != fresh.AdapterManifestDigest {
		failure, err := failIntegrity("session adapter binding no longer equals the freshly read trusted facts", "binding")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// CheckCallBinding requires one state-bearing call's context to
// match its sealed binding and its admitted tuple: the provider
// identifier, both digests, and the environment must equal the
// binding facts and the tuple the registry admitted, and the role
// must equal the call's side. A context answering for another
// provider, another executable, another manifest, or an
// unadmitted tuple is refused before the adapter surface is
// touched.
func CheckCallBinding(binding ExecutionBinding, role Role, context CallContext, admitted Tuple) error {
	if role != binding.Role {
		failure, err := failIntegrity("adapter call role does not match the sealed binding role", "role")
		if err != nil {
			return err
		}
		return failure
	}
	if context.ProviderID != binding.ProviderID {
		failure, err := failIntegrity("adapter call provider identifier does not match the sealed binding", "provider_id")
		if err != nil {
			return err
		}
		return failure
	}
	if context.ManifestDigest != binding.AdapterManifestDigest {
		failure, err := failIntegrity("adapter call manifest digest does not match the sealed binding", "session_adapter_manifest_digest")
		if err != nil {
			return err
		}
		return failure
	}
	if context.ExecutableSHA256 != binding.ExecutableSHA256 {
		failure, err := failIntegrity("adapter call executable digest does not match the sealed binding", "executable_sha256")
		if err != nil {
			return err
		}
		return failure
	}
	if context.Environment != admitted {
		failure, err := failUnsupportedTuple("adapter call environment is not the admitted tuple", context.Environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
