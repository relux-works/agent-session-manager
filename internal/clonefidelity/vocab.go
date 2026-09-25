package clonefidelity

import (
	"github.com/relux-works/agent-session-manager/internal/environ"
)

// This file owns the four Section 13.14.2 closed vocabularies: the
// seven dispositions, the five profiles, the five strategies, and
// the nineteen core reason codes plus reverse-DNS extensions. Every
// table is quoted from the pinned specification text; the oracle
// tests retype each list from the spec and sweep admission and
// refusal through the production entries.

// dispositions is the exact seven-member disposition vocabulary
// Section 13.14.2 states: exact|semantic|summarized|
// opaque_preserved|synthesized|omitted|unrecoverable.
var dispositions = []string{
	"exact",
	"semantic",
	"summarized",
	"opaque_preserved",
	"synthesized",
	"omitted",
	"unrecoverable",
}

// ValidDisposition reports whether the name is a fidelity
// disposition.
func ValidDisposition(disposition string) bool {
	for _, allowed := range dispositions {
		if disposition == allowed {
			return true
		}
	}
	return false
}

// Dispositions returns the exact seven-member disposition
// vocabulary in specification order. Callers iterate the returned
// copy; the table above stays the single owner.
func Dispositions() []string {
	out := make([]string, len(dispositions))
	copy(out, dispositions)
	return out
}

// profiles is the exact five-member profile vocabulary Section
// 13.14.2 states: strict_exact|maximal_safe|compact|messages_only|
// archive_only. Maximal-safe is default.
var profiles = []string{
	"strict_exact",
	"maximal_safe",
	"compact",
	"messages_only",
	"archive_only",
}

// ValidProfile reports whether the name is a fidelity profile.
func ValidProfile(profile string) bool {
	for _, allowed := range profiles {
		if profile == allowed {
			return true
		}
	}
	return false
}

// Profiles returns the exact five-member profile vocabulary in
// specification order. Callers iterate the returned copy; the table
// above stays the single owner.
func Profiles() []string {
	out := make([]string, len(profiles))
	copy(out, profiles)
	return out
}

// strategies is the exact five-member strategy vocabulary Section
// 13.14.2 states: same_environment_native_rewrite|
// target_native_writer|target_official_import|continuation_context|
// archive_only. Continuation context is explicitly non-native
// historical fidelity. (The Projection Plan sibling narrows this to
// the four non-archive strategies; this registry owns the full
// Section 13.14.2 list.)
var strategies = []string{
	"same_environment_native_rewrite",
	"target_native_writer",
	"target_official_import",
	"continuation_context",
	"archive_only",
}

// ValidStrategy reports whether the name is a fidelity strategy.
func ValidStrategy(strategy string) bool {
	for _, allowed := range strategies {
		if strategy == allowed {
			return true
		}
	}
	return false
}

// Strategies returns the exact five-member strategy vocabulary in
// specification order. Callers iterate the returned copy; the table
// above stays the single owner.
func Strategies() []string {
	out := make([]string, len(strategies))
	copy(out, strategies)
	return out
}

// coreReasons is the exact nineteen-member closed core reason list
// Section 13.14.2 states: target_no_equivalent|
// source_not_persisted|source_truncated|source_corrupt|
// foreign_encrypted_payload|foreign_signature_unverifiable|
// target_schema_constraint|target_context_limit|target_size_limit|
// target_version_gate|official_importer_loss|graph_flattened|
// unsafe_pending_action|credential_excluded|secret_policy|
// operator_policy|unsupported_media_type|unknown_native_event|
// derived_index_rebuilt.
var coreReasons = []string{
	"target_no_equivalent",
	"source_not_persisted",
	"source_truncated",
	"source_corrupt",
	"foreign_encrypted_payload",
	"foreign_signature_unverifiable",
	"target_schema_constraint",
	"target_context_limit",
	"target_size_limit",
	"target_version_gate",
	"official_importer_loss",
	"graph_flattened",
	"unsafe_pending_action",
	"credential_excluded",
	"secret_policy",
	"operator_policy",
	"unsupported_media_type",
	"unknown_native_event",
	"derived_index_rebuilt",
}

// IsCoreReasonCode reports whether the code is one of the nineteen
// closed core reason codes.
func IsCoreReasonCode(code string) bool {
	for _, core := range coreReasons {
		if code == core {
			return true
		}
	}
	return false
}

// CoreReasonCodes returns the exact nineteen-member closed core
// reason list in specification order. Callers iterate the returned
// copy; the table above stays the single owner.
func CoreReasonCodes() []string {
	out := make([]string, len(coreReasons))
	copy(out, coreReasons)
	return out
}

// ValidReasonCode reports whether the code is an admissible reason:
// one of the nineteen closed core codes or a reverse-DNS extension
// reason. An extension can never redefine a core code: core codes
// carry no dot and the reverse-DNS grammar requires one, so the
// two arms are disjoint by construction. Length bounds live at the
// JSON gates (reason_codes items are string[1..128]); this
// predicate is pure vocabulary.
func ValidReasonCode(code string) bool {
	if IsCoreReasonCode(code) {
		return true
	}
	return isExtensionReason(code)
}

// isExtensionReason reports whether the code is a reverse-DNS
// extension reason. It delegates to the single environ grammar so
// extension reasons and extension keys can never drift.
func isExtensionReason(code string) bool {
	return environ.CheckReverseDNS(code)
}
