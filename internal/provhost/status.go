package provhost

import (
	"encoding/base64"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// StatusState is the reconciled durable state of a materialization
// transaction. Unknown is fail-closed: the host must quarantine the
// transaction root rather than infer prepare or commit.
type StatusState string

// ProviderTransactionStatus states Section 7.5 defines.
const (
	StatusUnknown    StatusState = "unknown"
	StatusPrepared   StatusState = "prepared"
	StatusCommitted  StatusState = "committed"
	StatusRolledBack StatusState = "rolled_back"
)

// statusStates is the exact Section 7.5 ProviderTransactionStatus state
// vocabulary: unknown, prepared, committed, and rolled_back, nothing
// else. It is package-level so the closed vocabulary census derives
// it: admitting a fifth state here must redden
// TestStatusStatesAreDerivedFromSpec, not decode as durable state.
var statusStates = map[string]bool{
	string(StatusUnknown):    true,
	string(StatusPrepared):   true,
	string(StatusCommitted):  true,
	string(StatusRolledBack): true,
}

// validStatusState reports whether the state is a registry member.
func validStatusState(state string) bool {
	return statusStates[state]
}

// StatusIDs identifies the transaction a status read must describe: the
// caller-supplied materialization and transaction IDs and the exact
// transaction authority the request located it through. A status naming
// any other transaction is not this transaction's durable state.
type StatusIDs struct {
	MaterializationID      string
	TransactionID          string
	TransactionAuthorityID string
}

// statusBodyMembers is the exact member set DecodeStatusOutcome accepts:
// the ProviderTransactionStatus row of Section 7.5, where operation_id is
// the mutation key the status request omits. A well-formed operation_id is
// ignored — it is not this read's identity — but a malformed one makes
// the observation integrity-invalid, as does any unknown member.
var statusBodyMembers = map[string]bool{
	"operation_id":             true,
	"materialization_id":       true,
	"transaction_id":           true,
	"transaction_authority_id": true,
	"plan_id":                  true,
	"state":                    true,
	"rollback_token":           true,
	"native_discovery":         true,
}

// DecodeStatusOutcome reconciles one materialize-status success body to
// the current durable state. It enforces the Section 7.5 recovery rules:
// unknown requires null plan, token, and discovery; prepared requires all
// three non-null; committed and rolled_back require a non-null plan and
// discovery with a null token. Identity members must equal the requested
// transaction.
//
// Every uninterpretable observation fails with integrity_failure,
// including a state of unknown: the caller must quarantine the
// transaction root and must not infer success. Status reads are evolving
// observations, so no byte-identical replay rule applies: the same
// transaction may report prepared under one envelope request_id and
// committed under the next, and both readings are returned as-is.
//
// Only the charset and length of a rollback token are checked here (at
// least 256 bits of base64url); its entropy is not observable. The shapes
// of native_discovery and plan contents belong to the operation layer.
func DecodeStatusOutcome(body []byte, want StatusIDs) (StatusState, error) {
	integrity := func(detail string, state string) (StatusState, error) {
		failure, err := failIntegrity(detail, state, want.MaterializationID, want.TransactionID)
		if err != nil {
			return "", err
		}
		return "", failure
	}
	members, fault := decodeStrictObject(body)
	if fault != nil {
		return integrity("status body is "+fault.detail, "")
	}
	for name := range members {
		if !statusBodyMembers[name] {
			return integrity("status body carries unknown member", "")
		}
	}
	for _, name := range []string{"materialization_id", "transaction_id", "transaction_authority_id", "plan_id", "state", "rollback_token", "native_discovery"} {
		if _, present := members[name]; !present {
			return integrity("status body misses a required member", "")
		}
	}
	stringMember := func(name string) (string, bool) {
		var value string
		if err := json.Unmarshal(members[name], &value); err != nil {
			return "", false
		}
		return value, true
	}
	materializationID, ok := stringMember("materialization_id")
	if !ok || materializationID != want.MaterializationID {
		return integrity("status names another materialization", "")
	}
	transactionID, ok := stringMember("transaction_id")
	if !ok || transactionID != want.TransactionID {
		return integrity("status names another transaction", "")
	}
	authorityID, ok := stringMember("transaction_authority_id")
	if !ok || authorityID != want.TransactionAuthorityID {
		return integrity("status names another transaction authority", "")
	}
	if raw, present := members["operation_id"]; present {
		var operationID string
		if err := json.Unmarshal(raw, &operationID); err != nil {
			return integrity("status operation_id is not a string", "")
		}
		if _, err := scalar.ParseUUIDv7(operationID); err != nil {
			return integrity("status operation_id is not a UUIDv7", "")
		}
	}
	state, ok := stringMember("state")
	if !ok || !validStatusState(state) {
		return integrity("status state is not a registry member", "")
	}
	planNull := isNull(members["plan_id"])
	tokenNull := isNull(members["rollback_token"])
	discoveryNull := isNull(members["native_discovery"])
	if !planNull {
		var digest string
		if err := json.Unmarshal(members["plan_id"], &digest); err != nil {
			return integrity("status plan_id is not a string", state)
		}
		if _, err := scalar.ParseDigest(digest); err != nil {
			return integrity("status plan_id is not a digest", state)
		}
	}
	if !tokenNull {
		var token string
		if err := json.Unmarshal(members["rollback_token"], &token); err != nil {
			return integrity("status rollback_token is not a string", state)
		}
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(raw) < 32 {
			return integrity("status rollback_token is shorter than 256 bits", state)
		}
	}
	if !discoveryNull && !isJSONObject(members["native_discovery"]) {
		return integrity("status native_discovery is not an object", state)
	}
	switch StatusState(state) {
	case StatusUnknown:
		// Unknown is not a usable state: whether or not the
		// nullability shape holds, the host must fail closed and
		// quarantine the transaction root rather than infer prepare
		// or commit. The shape violation names its own rule; a
		// well-shaped unknown still refuses as unknown.
		if !planNull || !tokenNull || !discoveryNull {
			return integrity("unknown status carries plan, token, or discovery", state)
		}
		return integrity("transaction state is unknown", state)
	case StatusPrepared:
		if planNull || tokenNull || discoveryNull {
			return integrity("prepared status misses plan, token, or discovery", state)
		}
	case StatusCommitted, StatusRolledBack:
		if planNull || discoveryNull || !tokenNull {
			return integrity("terminal status misses plan or discovery or keeps a token", state)
		}
	}
	return StatusState(state), nil
}

// isNull reports whether the raw member is a JSON null literal.
func isNull(raw json.RawMessage) bool {
	return string(raw) == "null"
}
