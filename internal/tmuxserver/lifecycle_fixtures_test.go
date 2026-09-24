package tmuxserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/gowebpki/jcs"
)

// Lifecycle fixtures: identities, digests, timestamps, and closed-body
// builders for the eight operations. UUIDv7/v4 values are fixed
// literals in canonical lowercase form; the mint helper below derives
// binding IDs through the test-only JCS identity (production never
// mints — the restore echo bound).

const (
	lxSession    = "0198f4c8-8e50-7f66-8f70-111111111111"
	lxSessionB   = "0198f4c8-8e50-7f66-8f70-111111111112"
	lxInstance   = "0198f4c8-8e50-7f66-8f70-222222222221"
	lxOperation  = "0198f4c8-8e50-7f66-8f70-333333333331"
	lxBootstrap  = "0198f4c8-8e50-7f66-8f70-444444444441"
	lxClient     = "0198f4c8-8e50-7f66-8f70-555555555551"
	lxClientB    = "0198f4c8-8e50-7f66-8f70-555555555552"
	lxQuiesce    = "0198f4c8-8e50-7f66-8f70-666666666661"
	lxHost       = "0198f4c8-8e50-7f66-8f70-777777777771"
	lxLease      = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	lxLeaseB     = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	lxDigestA    = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	lxDigestB    = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	lxDigestC    = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	lxIssued     = "2026-09-01T00:00:00.000Z"
	lxExpires    = "2026-09-02T00:00:00.000Z"
	lxDeadline   = "2026-09-01T18:00:00.000Z"
	lxImpl       = "1.2.3"
	lxProto      = "1.0.0"
	lxGeneration = "generation-one"
)

func lxNow() time.Time {
	now, err := time.Parse(time.RFC3339Nano, "2026-09-01T12:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return now
}

// lxAuth builds the closed AXAuthorization member map for one kind.
func lxAuth(kind string) map[string]any {
	return map[string]any{
		"lease_id":                  lxLease,
		"lease_epoch":               float64(7),
		"holder_host_id":            lxHost,
		"authorization_kind":        kind,
		"issued_at":                 lxIssued,
		"expires_at":                lxExpires,
		"authorization_evidence_id": lxDigestA,
	}
}

// lxContext builds the closed MutationContext member map. The caller
// supplies the operation's canonical idempotency key.
func lxContext(key string) map[string]any {
	return map[string]any{
		"operation_id":           lxOperation,
		"session_id":             lxSession,
		"terminal_instance_id":   lxInstance,
		"terminal_backend_id":    "ax.tmux",
		"implementation_version": lxImpl,
		"protocol_version":       lxProto,
		"backend_generation":     lxGeneration,
		"idempotency_key":        key,
		"deadline_at":            lxDeadline,
		"authorization":          lxAuth("create"),
	}
}

// lxContextJSON marshals a context map. mutate adjusts the map before
// marshal (nil for the valid shape).
func lxContextJSON(t *testing.T, key string, mutate func(map[string]any)) json.RawMessage {
	t.Helper()
	object := lxContext(key)
	if mutate != nil {
		mutate(object)
	}
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// lxBindingDoc builds a valid Terminal Instance Binding 1.0.0
// document minted through the test-only JCS identity, bound to the
// lifecycle session/instance/generation. mutate adjusts the object
// before the ID is minted (nil for the valid shape); a mutate that
// sets binding_id to a non-empty value keeps it (for mismatch tests).
func lxBindingDoc(t *testing.T, mutate func(map[string]any)) json.RawMessage {
	t.Helper()
	object := map[string]any{
		"schema":                 "urn:ax:schema:terminal-instance-binding",
		"schema_version":         "1.0.0",
		"binding_id":             "",
		"session_id":             lxSession,
		"host_id":                lxHost,
		"host_incarnation_id":    lxHost,
		"terminal_instance_id":   lxInstance,
		"terminal_backend_id":    "ax.tmux",
		"implementation_version": lxImpl,
		"protocol_version":       lxProto,
		"backend_generation":     lxGeneration,
		"native_reference":       "tmux-session-fixture-0",
		"created_at":             "2026-09-01T00:00:00.000Z",
		"supersedes_binding_id":  nil,
		"extensions":             map[string]any{},
	}
	if mutate != nil {
		mutate(object)
	}
	if id, ok := object["binding_id"].(string); ok && id == "" {
		object["binding_id"] = lxMintIdentity(t, object, "binding_id")
	}
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// lxMintIdentity derives the JCS omit-self digest for test fixtures.
// It mirrors the termbind test helper; production never mints.
func lxMintIdentity(t *testing.T, object map[string]any, selfField string) string {
	t.Helper()
	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != selfField {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// lxAttachAuth builds the closed AttachAuthorization member map.
func lxAttachAuth(transport string, input bool) map[string]any {
	return map[string]any{
		"policy_evidence_id":  lxDigestA,
		"authorizing_host_id": lxHost,
		"transport":           transport,
		"input_authorized":    input,
		"issued_at":           lxIssued,
		"expires_at":          lxExpires,
	}
}

// lxBody marshals a top-level operation body map.
func lxBody(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// lxCreateKey is the canonical create/restore key for the fixtures.
func lxCreateKey() string { return lxSession + "/" + lxBootstrap }

// lxQuiesceKey is the canonical quiesce-input key for the fixtures.
func lxQuiesceKey() string { return lxInstance + "/quiesce/" + lxQuiesce }

// lxBoundaryKey is the canonical wait-safe-boundary key.
func lxBoundaryKey(kind string) string {
	return lxInstance + "/boundary/" + lxQuiesce + "/" + kind
}

// lxStopKey is the canonical request-stop key.
func lxStopKey() string { return lxInstance + "/stop/" + lxDigestA }

// lxTerminateKey is the canonical terminate-stale key.
func lxTerminateKey() string { return lxInstance + "/terminate/" + lxLeaseB + "/9" }

// lxCreateBody builds the closed create body. mutate adjusts the
// top-level map (nil for valid).
func lxCreateBody(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	context := lxContext(lxCreateKey())
	context["authorization"] = lxAuth("create")
	object := map[string]any{
		"context":                context,
		"binding":                json.RawMessage(lxBindingDoc(t, nil)),
		"bootstrap_operation_id": lxBootstrap,
		"entrypoint":             map[string]any{"argv": []string{"ax", "pane", lxSession}},
		"presentation_transport": "local_only",
		"interactive":            true,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxAttachBody builds the closed attach body.
func lxAttachBody(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	object := map[string]any{
		"session_id":             lxSession,
		"terminal_instance_id":   lxInstance,
		"terminal_backend_id":    "ax.tmux",
		"implementation_version": lxImpl,
		"protocol_version":       lxProto,
		"backend_generation":     lxGeneration,
		"client_id":              lxClient,
		"transport":              "local_only",
		"input_authorized":       true,
		"deadline_at":            lxDeadline,
		"authorization":          lxAttachAuth("local_only", true),
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxTerminateBody builds the closed terminate-stale body.
func lxTerminateBody(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	context := lxContext(lxTerminateKey())
	context["authorization"] = lxAuth("force_stale")
	object := map[string]any{
		"context":                context,
		"stale_lease_id":         lxLeaseB,
		"stale_epoch":            float64(9),
		"diagnostic_evidence_id": lxDigestC,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxRestoreBody builds the closed restore body. prior names the
// carried prior_binding_id (lxDigestA unless the caller rebinds).
func lxRestoreBody(t *testing.T, prior string, mutate func(map[string]any)) []byte {
	t.Helper()
	context := lxContext(lxCreateKey())
	context["authorization"] = lxAuth("restore")
	object := map[string]any{
		"context":                context,
		"prior_binding_id":       prior,
		"checkpoint_id":          lxDigestC,
		"bootstrap_operation_id": lxBootstrap,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxStatusBody builds the closed status body. exact selects the
// exact-instance shape (both non-null) versus session-scoped (both null).
func lxStatusBody(t *testing.T, exact, provider bool, mutate func(map[string]any)) []byte {
	t.Helper()
	var instance, generation any
	if exact {
		instance, generation = lxInstance, lxGeneration
	}
	object := map[string]any{
		"session_id":                   lxSession,
		"terminal_instance_id":         instance,
		"terminal_backend_id":          "ax.tmux",
		"implementation_version":       lxImpl,
		"protocol_version":             lxProto,
		"backend_generation":           generation,
		"include_provider_observation": provider,
		"deadline_at":                  lxDeadline,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxQuiesceBody builds the closed quiesce-input body.
func lxQuiesceBody(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	context := lxContext(lxQuiesceKey())
	context["authorization"] = lxAuth("control")
	object := map[string]any{
		"context":               context,
		"quiescence_generation": lxQuiesce,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxBoundaryBody builds the closed wait-safe-boundary body.
func lxBoundaryBody(t *testing.T, kind string, timeout float64, mutate func(map[string]any)) []byte {
	t.Helper()
	context := lxContext(lxBoundaryKey(kind))
	context["authorization"] = lxAuth("control")
	object := map[string]any{
		"context":               context,
		"quiescence_generation": lxQuiesce,
		"provider_proof_kind":   kind,
		"timeout_ms":            timeout,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}

// lxStopBody builds the closed request-stop body.
func lxStopBody(t *testing.T, timeout float64, mutate func(map[string]any)) []byte {
	t.Helper()
	context := lxContext(lxStopKey())
	context["authorization"] = lxAuth("control")
	object := map[string]any{
		"context":                   context,
		"safe_boundary_evidence_id": lxDigestA,
		"graceful_timeout_ms":       timeout,
	}
	if mutate != nil {
		mutate(object)
	}
	return lxBody(t, object)
}
