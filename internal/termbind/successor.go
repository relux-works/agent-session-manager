package termbind

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// MintSuccessorBinding mints the reboot-successor Terminal Instance
// Binding for a new backend generation over a parsed prior: the
// §4.C restore row requires the restore result to carry the new
// binding, and only this owner computes the Section 4.B identity, so
// the backend succeeds the prior through this entry instead of
// re-encoding or echoing it.
//
// The successor inherits every prior member except three: the backend
// generation moves to the new generation, supersedes_binding_id names
// the prior digest, and binding_id is the recomputed omit-self
// identity. The mint is deterministic and clock-free: the same prior
// and generation always mint the same document, so a retry after a
// crash between the restore effects and the successor persist
// converges instead of forking. The minted document is re-admitted
// before return: a mint this entry cannot parse is refused rather
// than persisted anywhere.
func MintSuccessorBinding(prior Binding, generation string) (Binding, []byte, error) {
	empty := Binding{}
	if _, err := terminalbackend.GenerationDigest(generation); err != nil {
		return empty, nil, err
	}
	if generation == prior.BackendGeneration {
		return empty, nil, protocolRefusal("binding successor generation")
	}
	object := map[string]any{
		"schema":                 BindingSchema,
		"schema_version":         BindingSchemaVersion,
		"binding_id":             "",
		"session_id":             prior.SessionID,
		"host_id":                prior.HostID,
		"host_incarnation_id":    prior.HostIncarnationID,
		"terminal_instance_id":   prior.TerminalInstanceID,
		"terminal_backend_id":    prior.TerminalBackendID,
		"implementation_version": prior.ImplementationVersion,
		"protocol_version":       prior.ProtocolVersion,
		"backend_generation":     generation,
		"native_reference":       prior.NativeReference,
		"created_at":             prior.CreatedAt,
		"supersedes_binding_id":  prior.BindingID,
		"extensions":             map[string]any{},
	}
	staged, err := json.Marshal(object)
	if err != nil {
		return empty, nil, protocolRefusal("binding successor frame")
	}
	identity, err := bindingIdentity(staged, "binding_id")
	if err != nil {
		return empty, nil, err
	}
	object["binding_id"] = identity
	raw, err := json.Marshal(object)
	if err != nil {
		return empty, nil, protocolRefusal("binding successor frame")
	}
	minted, err := ParseTerminalBinding(raw)
	if err != nil {
		return empty, nil, protocolRefusal("binding successor image")
	}
	return minted, raw, nil
}
