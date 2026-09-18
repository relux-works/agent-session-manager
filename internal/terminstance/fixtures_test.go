package terminstance

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

const (
	fixtureSessionA   = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	fixtureSessionB   = "0198f4c8-8e50-7f66-8f70-1234567890ac"
	fixtureInstance   = "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb1"
	fixtureOperation  = "0198f4c8-8e50-7f66-8f70-ccccccccccc1"
	fixtureOperationB = "0198f4c8-8e50-7f66-8f70-ccccccccccc2"
	fixtureHost       = "0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa1"
	fixtureQuiescence = "0198f4c8-8e50-7f66-8f70-ddddddddddd1"
	fixtureBootstrap  = "0198f4c8-8e50-7f66-8f70-eeeeeeeeeee1"
	fixtureLease      = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	fixtureLeaseB     = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	fixtureDigestA    = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fixtureDigestB    = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	fixtureDigestC    = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	fixtureIssued     = "2026-09-01T00:00:00.000Z"
	fixtureExpires    = "2026-09-02T00:00:00.000Z"
	fixtureDeadline   = "2026-09-01T18:00:00.000Z"
	// fixtureLateDeadline lies AFTER the fixture authorization
	// expiry, so an expired authorization is observable independently
	// of the deadline arm: at an instant past the expiry but before
	// this deadline, the auth gate fires while the deadline passes.
	fixtureLateDeadline = "2026-09-03T00:00:00.000Z"
	fixtureBackend      = "example.backend"
	fixtureImpl         = "1.2.3"
	fixtureProto        = "1.0.0"
	fixtureGen          = "generation-one"
)

func fixtureNow() time.Time {
	now, err := time.Parse(time.RFC3339Nano, "2026-09-01T12:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return now
}

// fixtureLater returns an instant past the fixture deadline and the
// fixture authorization expiry alike.
func fixtureLater() time.Time {
	later, err := time.Parse(time.RFC3339Nano, "2026-09-04T00:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return later
}

// fixturePastDeadline returns an instant past the fixture deadline but
// before the fixture authorization expiry, so the deadline arm fires
// while auth still binds.
func fixturePastDeadline() time.Time {
	past, err := time.Parse(time.RFC3339Nano, "2026-09-01T19:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return past
}

// fixturePastExpiry returns an instant past the fixture authorization
// expiry but before the late deadline, so the expiry arm fires while
// the deadline passes.
func fixturePastExpiry() time.Time {
	past, err := time.Parse(time.RFC3339Nano, "2026-09-02T01:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return past
}

func fixtureTimestamp(t *testing.T, value string) scalar.Timestamp {
	t.Helper()
	parsed, err := scalar.ParseTimestamp(value)
	if err != nil {
		t.Fatalf("ParseTimestamp(%q) error = %v", value, err)
	}
	return parsed
}

// authDoc renders an AXAuthorization document, applying member overrides.
// A nil override value drops the member; overrides are raw JSON, so
// callers spell numbers, booleans and nulls without quotes.
func authDoc(epoch string, overrides map[string]*string) string {
	members := map[string]string{
		"lease_id":                  `"` + fixtureLease + `"`,
		"lease_epoch":               epoch,
		"holder_host_id":            `"` + fixtureHost + `"`,
		"authorization_kind":        `"control"`,
		"issued_at":                 `"` + fixtureIssued + `"`,
		"expires_at":                `"` + fixtureExpires + `"`,
		"authorization_evidence_id": `"` + fixtureDigestA + `"`,
	}
	for name, override := range overrides {
		if override == nil {
			delete(members, name)
		} else {
			members[name] = *override
		}
	}
	names := []string{"lease_id", "lease_epoch", "holder_host_id", "authorization_kind", "issued_at", "expires_at", "authorization_evidence_id"}
	var body strings.Builder
	body.WriteString("{")
	first := true
	for _, name := range names {
		raw, known := members[name]
		if !known {
			continue
		}
		if !first {
			body.WriteString(",")
		}
		first = false
		body.WriteString(`"` + name + `":` + raw)
	}
	for name, raw := range members {
		known := false
		for _, listed := range names {
			if listed == name {
				known = true
				break
			}
		}
		if known {
			continue
		}
		if !first {
			body.WriteString(",")
		}
		first = false
		body.WriteString(`"` + name + `":` + raw)
	}
	body.WriteString("}")
	return body.String()
}

func strptr(value string) *string { return &value }

// testAuth parses the default control authorization for the fixture
// lease at epoch 1.
func testAuth(t *testing.T) AXAuthorization {
	t.Helper()
	auth, err := ParseAXAuthorization([]byte(authDoc("1", nil)))
	if err != nil {
		t.Fatalf("ParseAXAuthorization() error = %v", err)
	}
	return auth
}

// contextDoc renders a MutationContext document embedding authJSON.
func contextDoc(operationID, session, instance, backend, impl, proto, generation, key, deadline, authJSON string) string {
	return `{"operation_id":"` + operationID + `",` +
		`"session_id":"` + session + `",` +
		`"terminal_instance_id":"` + instance + `",` +
		`"terminal_backend_id":"` + backend + `",` +
		`"implementation_version":"` + impl + `",` +
		`"protocol_version":"` + proto + `",` +
		`"backend_generation":"` + generation + `",` +
		`"idempotency_key":"` + key + `",` +
		`"deadline_at":"` + deadline + `",` +
		`"authorization":` + authJSON + `}`
}

// testContext builds the default parsed quiesce context through the
// production ParseMutationContext entry.
func testContext(t *testing.T) MutationContext {
	t.Helper()
	return testContextWithDeadline(t, fixtureDeadline)
}

// testContextWithDeadline builds the parsed quiesce context with an
// explicit deadline through the production ParseMutationContext
// entry.
func testContextWithDeadline(t *testing.T, deadline string) MutationContext {
	t.Helper()
	key := fixtureInstance + "/quiesce/" + fixtureQuiescence
	raw := contextDoc(fixtureOperation, fixtureSessionA, fixtureInstance, fixtureBackend, fixtureImpl, fixtureProto, fixtureGen, key, deadline, authDoc("1", nil))
	context, err := ParseMutationContext([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMutationContext() error = %v", err)
	}
	return context
}

func testParams() Params {
	return Params{QuiescenceGeneration: fixtureQuiescence}
}

// mockBackend is the modeled terminal backend: scripted effects and a
// scripted status observation with a full call record.
type mockBackend struct {
	mutex         sync.Mutex
	effects       []terminalbackend.SideEffect
	evidence      map[terminalbackend.SideEffect]string
	effectErrors  map[terminalbackend.SideEffect]error
	observation   StatusObservation
	observeErr    error
	observeCalls  int
	effectHandler func(effect terminalbackend.SideEffect) (string, error)
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		evidence:     make(map[terminalbackend.SideEffect]string),
		effectErrors: make(map[terminalbackend.SideEffect]error),
	}
}

func (backend *mockBackend) PerformEffect(_ context.Context, effect terminalbackend.SideEffect) (string, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	backend.effects = append(backend.effects, effect)
	if backend.effectHandler != nil {
		return backend.effectHandler(effect)
	}
	if err, failing := backend.effectErrors[effect]; failing {
		return "", err
	}
	if evidence, known := backend.evidence[effect]; known {
		return evidence, nil
	}
	return defaultEvidence(len(backend.effects)), nil
}

// defaultEvidence mints a distinct valid digest per effect call so
// multi-effect results satisfy sorted-unique evidence without every
// test spelling per-effect evidence by hand.
func defaultEvidence(call int) string {
	return "sha256:" + strings.Repeat("0", 63) + string(rune('0'+call%10))
}

func (backend *mockBackend) ObserveStatus(_ context.Context, _ StatusBody) (StatusObservation, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	backend.observeCalls++
	if backend.observeErr != nil {
		return StatusObservation{}, backend.observeErr
	}
	return backend.observation, nil
}

func (backend *mockBackend) performed() []terminalbackend.SideEffect {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	return append([]terminalbackend.SideEffect(nil), backend.effects...)
}

// testEngine builds an engine over a fresh temp store, the given
// backend, a fixed winning lease and generation, and the fixture clock.
func testEngine(t *testing.T, backend Backend, lease LeaseView, generation string) *Engine {
	t.Helper()
	store, err := OpenReceiptStore(t.TempDir())
	if err != nil {
		t.Fatalf("OpenReceiptStore() error = %v", err)
	}
	now := fixtureNow()
	return &Engine{
		Store:             store,
		Backend:           backend,
		CurrentLease:      func() LeaseView { return lease },
		CurrentGeneration: func() string { return generation },
		Now:               func() time.Time { return now },
	}
}

func testLease() LeaseView {
	return LeaseView{LeaseID: fixtureLease, Epoch: 1}
}

func testAdmitted(capabilities ...string) terminalbackend.Admitted {
	return terminalbackend.Admitted{Capabilities: capabilities}
}

// requireRefusal asserts the exact literal wire code and static detail.
// Every expectation is a literal from the pinned specification text,
// never a production constant.
func requireRefusal(t *testing.T, err error, wantCode, wantDetail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q at %q", wantCode, wantDetail)
	}
	var refusal *Error
	if !errors.As(err, &refusal) {
		t.Fatalf("error = %v (%T), want *Error with code %q at %q", err, err, wantCode, wantDetail)
	}
	if refusal.Code != wantCode || refusal.Detail != wantDetail {
		t.Errorf("refusal = %q at %q, want %q at %q", refusal.Code, refusal.Detail, wantCode, wantDetail)
	}
}

func requireLiteral(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want literal %q", what, got, want)
	}
}

func mustParseState(t *testing.T, value string) terminalbackend.InstanceState {
	t.Helper()
	state, err := terminalbackend.ParseInstanceState(value)
	if err != nil {
		t.Fatalf("ParseInstanceState(%q) error = %v", value, err)
	}
	return state
}

func boolptr(value bool) *bool { return &value }
