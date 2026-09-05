package terminalbackend_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	terminalbackend "github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

const (
	conformanceSessionA = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	conformanceSessionB = "0198f4c8-8e50-7f66-8f70-1234567890ac"
	conformanceHost     = "0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa1"
	conformanceIssued   = "2026-09-01T00:00:00.000Z"
	conformanceExpires  = "2026-09-02T00:00:00.000Z"
)

func conformanceNow() time.Time {
	now, err := time.Parse(time.RFC3339Nano, "2026-09-01T12:00:00.000Z")
	if err != nil {
		panic(err)
	}
	return now
}

func testAttachAuthorization(t *testing.T) terminalbackend.AttachAuthorization {
	t.Helper()

	raw := `{"policy_evidence_id":"` + testDigest + `",` +
		`"authorizing_host_id":"` + conformanceHost + `",` +
		`"transport":"local_only",` +
		`"input_authorized":true,` +
		`"issued_at":"` + conformanceIssued + `",` +
		`"expires_at":"` + conformanceExpires + `"}`

	auth, err := terminalbackend.ParseAttachAuthorization([]byte(raw))
	if err != nil {
		t.Fatalf("ParseAttachAuthorization() error = %v", err)
	}
	return auth
}

func requireConformanceRefusal(t *testing.T, err error, wantCode, wantDetail string) {
	t.Helper()

	var refusal *terminalbackend.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("error = %v, want *Error with code %q at %q", err, wantCode, wantDetail)
	}
	if refusal.Code != wantCode || refusal.Detail != wantDetail {
		t.Errorf("refusal = %q at %q, want %q at %q", refusal.Code, refusal.Detail, wantCode, wantDetail)
	}
}

// TestParseInstanceStateAdmitsOnlyTheEightClosedStates drives the state
// domain at the production entry: every member of the §4.C enum is
// admitted, and refused samples are refused as a protocol error rather
// than mapped onto a near neighbour. The refused list is a sampling
// bound over an unbounded string domain, not a closure — it pins the
// refusal shape. The closure is derived: the admitted set must equal the
// parser's switch-case list exactly (pinned by
// TestDerivedVocabulariesAreExactlyTheAdmittedSets), so a ninth admitted
// state reddens there even though no refused sample names it.
func TestParseInstanceStateAdmitsOnlyTheEightClosedStates(t *testing.T) {
	t.Parallel()

	for _, state := range []string{
		"absent", "creating", "parked", "active",
		"quiescing", "stopped", "stale_fenced", "unavailable",
	} {
		if _, err := terminalbackend.ParseInstanceState(state); err != nil {
			t.Errorf("ParseInstanceState(%q) error = %v, want admission", state, err)
		}
	}
	for _, state := range []string{
		"", "Absent", "ACTIVE", "running", "detached", "stale-fenced",
		"parked ", " parked", "absent\x00", "creating\n",
		strings.Repeat("a", 1024),
	} {
		_, err := terminalbackend.ParseInstanceState(state)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "lifecycle state vocabulary")
	}
}

// TestParseOperationAdmitsOnlyTheTenClosedOperations drives the operation
// domain at the production entry: every member of the §4.D enum is
// admitted. The refused list is a sampling bound over an unbounded
// string domain, not a closure — it pins the refusal shape. The closure
// is derived: the admitted set must equal the parser's switch-case list
// exactly (pinned by TestDerivedVocabulariesAreExactlyTheAdmittedSets),
// so an eleventh admitted operation reddens there even though no refused
// sample names it.
func TestParseOperationAdmitsOnlyTheTenClosedOperations(t *testing.T) {
	t.Parallel()

	for _, operation := range []string{
		"manifest", "probe", "create", "attach", "status",
		"quiesce-input", "wait-safe-boundary", "request-stop",
		"terminate-stale", "restore",
	} {
		if _, err := terminalbackend.ParseOperation(operation); err != nil {
			t.Errorf("ParseOperation(%q) error = %v, want admission", operation, err)
		}
	}
	for _, operation := range []string{
		"", "Create", "ATTACH", "quiesce_input", "quiesce", "detach",
		"status ", " create", "terminate_stale", "delete",
		strings.Repeat("a", 1024),
	} {
		_, err := terminalbackend.ParseOperation(operation)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "operation vocabulary")
	}
}

// transitionFixture is one exact §4.C table row restated as an expectation:
// every allowed source of one operation with its success target and exact
// side effects.
type transitionFixture struct {
	operation   string
	source      string
	interactive bool
	target      string
	effects     []string
}

// TestCheckTransitionAdmitsEveryTableRow drives the whole §4.C matrix
// through the production entry: 8 status observations, both create
// targets, both attach sources, and every other row with its exact side
// effects. manifest and probe carry no instance source or target.
func TestCheckTransitionAdmitsEveryTableRow(t *testing.T) {
	t.Parallel()

	statusSources := []string{
		"absent", "creating", "parked", "active",
		"quiescing", "stopped", "stale_fenced", "unavailable",
	}
	var fixtures []transitionFixture
	for _, source := range statusSources {
		fixtures = append(fixtures, transitionFixture{
			operation: "status", source: source, target: source,
		})
	}
	fixtures = append(fixtures,
		transitionFixture{
			operation: "create", source: "absent", interactive: true,
			target:  "active",
			effects: []string{"binding_persisted", "wrapper_started"},
		},
		transitionFixture{
			operation: "create", source: "stopped", interactive: true,
			target:  "active",
			effects: []string{"binding_persisted", "wrapper_started"},
		},
		transitionFixture{
			operation: "create", source: "absent", interactive: false,
			target:  "parked",
			effects: []string{"binding_persisted", "wrapper_started"},
		},
		transitionFixture{
			operation: "create", source: "stopped", interactive: false,
			target:  "parked",
			effects: []string{"binding_persisted", "wrapper_started"},
		},
		transitionFixture{
			operation: "attach", source: "parked", target: "active",
			effects: []string{"attach_client_created"},
		},
		transitionFixture{
			operation: "attach", source: "active", target: "active",
			effects: []string{"attach_client_created"},
		},
		transitionFixture{
			operation: "quiesce-input", source: "active", target: "quiescing",
			effects: []string{"input_closed"},
		},
		transitionFixture{
			operation: "quiesce-input", source: "parked", target: "quiescing",
			effects: []string{"input_closed"},
		},
		transitionFixture{
			operation: "wait-safe-boundary", source: "quiescing", target: "quiescing",
			effects: []string{"safe_boundary_observed"},
		},
		transitionFixture{
			operation: "request-stop", source: "quiescing", target: "stopped",
			effects: []string{
				"graceful_stop_requested", "process_closed", "backend_store_closed",
			},
		},
		transitionFixture{
			operation: "terminate-stale", source: "stale_fenced", target: "stopped",
			effects: []string{"stale_incarnation_terminated", "process_closed"},
		},
		transitionFixture{
			operation: "terminate-stale", source: "unavailable", target: "stopped",
			effects: []string{"stale_incarnation_terminated", "process_closed"},
		},
		transitionFixture{
			operation: "restore", source: "absent", target: "parked",
			effects: []string{"binding_persisted", "wrapper_restored"},
		},
		transitionFixture{
			operation: "restore", source: "stopped", target: "parked",
			effects: []string{"binding_persisted", "wrapper_restored"},
		},
		transitionFixture{
			operation: "restore", source: "unavailable", target: "parked",
			effects: []string{"binding_persisted", "wrapper_restored"},
		},
	)

	for _, fixture := range fixtures {
		target, effects, err := terminalbackend.CheckTransition(
			fixture.operation, fixture.source, fixture.interactive)
		if err != nil {
			t.Errorf("CheckTransition(%q, %q, interactive=%v) error = %v",
				fixture.operation, fixture.source, fixture.interactive, err)
			continue
		}
		if string(target) != fixture.target {
			t.Errorf("CheckTransition(%q, %q, interactive=%v) target = %q, want %q",
				fixture.operation, fixture.source, fixture.interactive, target, fixture.target)
		}
		var got []string
		for _, effect := range effects {
			got = append(got, string(effect))
		}
		if len(got) != len(fixture.effects) {
			t.Errorf("CheckTransition(%q, %q) effects = %v, want %v",
				fixture.operation, fixture.source, got, fixture.effects)
			continue
		}
		for index := range got {
			if got[index] != fixture.effects[index] {
				t.Errorf("CheckTransition(%q, %q) effects = %v, want %v",
					fixture.operation, fixture.source, got, fixture.effects)
				break
			}
		}
	}

	for _, operation := range []string{"manifest", "probe"} {
		target, effects, err := terminalbackend.CheckTransition(operation, "", false)
		if err != nil {
			t.Errorf("CheckTransition(%q, %q) error = %v, want admission", operation, "", err)
			continue
		}
		if target != "" || len(effects) != 0 {
			t.Errorf("CheckTransition(%q, %q) = (%q, %v), want no instance target or effect",
				operation, "", target, effects)
		}
	}
}

// TestCheckTransitionRefusesEveryIllegalSource drives the full negative
// matrix: every operation against every source it does not admit,
// including unknown operations and states. A known operation against a
// disallowed source fails its local precondition; an unknown member is a
// protocol error. manifest and probe refuse any instance source at all.
func TestCheckTransitionRefusesEveryIllegalSource(t *testing.T) {
	t.Parallel()

	sources := []string{
		"", "absent", "creating", "parked", "active",
		"quiescing", "stopped", "stale_fenced", "unavailable",
	}
	admitted := map[string]map[string]bool{
		"manifest": {"": true}, "probe": {"": true},
		"create":             {"absent": true, "stopped": true},
		"attach":             {"parked": true, "active": true},
		"status":             {},
		"quiesce-input":      {"active": true, "parked": true},
		"wait-safe-boundary": {"quiescing": true},
		"request-stop":       {"quiescing": true},
		"terminate-stale":    {"stale_fenced": true, "unavailable": true},
		"restore":            {"absent": true, "stopped": true, "unavailable": true},
	}
	for _, source := range sources {
		admitted["status"][source] = source != ""
	}

	operations := []string{
		"manifest", "probe", "create", "attach", "status",
		"quiesce-input", "wait-safe-boundary", "request-stop",
		"terminate-stale", "restore",
	}
	for _, operation := range operations {
		for _, source := range sources {
			if admitted[operation][source] {
				continue
			}
			// An empty source names no state at all: the state
			// member is unknown, not merely disallowed.
			if source == "" {
				_, _, err := terminalbackend.CheckTransition(operation, source, false)
				requireConformanceRefusal(t, err,
					"terminal_backend_protocol_error", "lifecycle state vocabulary")
				continue
			}
			// manifest and probe carry no instance at all: any
			// source violates their scope, not a transition.
			if operation == "manifest" || operation == "probe" {
				_, _, err := terminalbackend.CheckTransition(operation, source, false)
				requireConformanceRefusal(t, err,
					"local_precondition_failed", "lifecycle instance scope")
				continue
			}
			for _, interactive := range []bool{false, true} {
				_, _, err := terminalbackend.CheckTransition(operation, source, interactive)
				requireConformanceRefusal(t, err,
					"local_precondition_failed", "lifecycle transition")
			}
		}
	}

	for _, operation := range []string{"", "delete", "CREATE", "launch"} {
		_, _, err := terminalbackend.CheckTransition(operation, "active", false)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "operation vocabulary")
	}
	for _, source := range []string{"running", "ACTIVE", "parked ", "bogus"} {
		_, _, err := terminalbackend.CheckTransition("attach", source, false)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "lifecycle state vocabulary")
	}
}

// TestCheckTransitionNeverEntersFencingOrBootstrapStates proves the two
// §4.C entry rules structurally: no backend operation enters
// stale_fenced (AX fencing observation does), and creating is never a
// success target (it exists only between the receipt and the first side
// effect). The sweep covers every transitioning operation, every source,
// and both create modes, so a row added with either target reddens here.
// status is excluded by name: it observes the source rather than
// entering anything, so status-of-creating reports creating without
// entering it.
func TestCheckTransitionNeverEntersFencingOrBootstrapStates(t *testing.T) {
	t.Parallel()

	operations := []string{
		"create", "attach", "quiesce-input", "wait-safe-boundary",
		"request-stop", "terminate-stale", "restore",
	}
	sources := []string{
		"absent", "creating", "parked", "active",
		"quiescing", "stopped", "stale_fenced", "unavailable",
	}
	for _, operation := range operations {
		for _, source := range sources {
			for _, interactive := range []bool{false, true} {
				target, _, err := terminalbackend.CheckTransition(operation, source, interactive)
				if err != nil {
					continue
				}
				if target == "stale_fenced" {
					t.Errorf("CheckTransition(%q, %q) enters stale_fenced: "+
						"no backend operation enters the fencing state", operation, source)
				}
				if target == "creating" {
					t.Errorf("CheckTransition(%q, %q) targets creating: "+
						"the bootstrap state is never a success target", operation, source)
				}
			}
		}
	}
}

// TestOnlyQuiesceInputEntersQuiescing proves the §4.C exclusivity rule:
// entering quiescing from any other state happens only through
// quiesce-input. wait-safe-boundary stays in quiescing without entering
// it, and status only observes, so both are excluded by the
// target-differs-from-source condition rather than by name.
func TestOnlyQuiesceInputEntersQuiescing(t *testing.T) {
	t.Parallel()

	operations := []string{
		"create", "attach", "status",
		"wait-safe-boundary", "request-stop", "terminate-stale", "restore",
	}
	sources := []string{
		"absent", "creating", "parked", "active",
		"quiescing", "stopped", "stale_fenced", "unavailable",
	}
	for _, operation := range operations {
		for _, source := range sources {
			for _, interactive := range []bool{false, true} {
				target, _, err := terminalbackend.CheckTransition(operation, source, interactive)
				if err != nil {
					continue
				}
				if target == "quiescing" && source != "quiescing" {
					t.Errorf("CheckTransition(%q, %q) enters quiescing: "+
						"only quiesce-input enters it", operation, source)
				}
			}
		}
	}
}

// allowedErrorCodesByOperation mirrors the production §4.C "Allowed error
// codes" table (allowedOperationErrors in conformance.go) exactly, one
// full row per operation — not a sample. The admit test below proves the
// mirror admits nothing production refuses; the complement test proves
// production admits nothing the mirror withholds from the pinned catalog.
// Together they pin every row exactly: a widened production row reddens
// the complement test, a narrowed one reddens the admit test.
var allowedErrorCodesByOperation = map[string][]string{
	"manifest": {
		"terminal_backend_protocol_error",
		"terminal_backend_protocol_incompatible",
		"terminal_backend_process_failed",
		"terminal_backend_timeout",
		"terminal_backend_integrity_failure",
	},
	"probe": {
		"terminal_backend_untrusted",
		"terminal_backend_manifest_probe_mismatch",
		"terminal_backend_implementation_drift",
		"terminal_backend_protocol_error",
		"terminal_backend_process_failed",
		"terminal_backend_timeout",
		"terminal_backend_integrity_failure",
	},
	"create": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"terminal_backend_unavailable",
		"terminal_backend_timeout",
		"terminal_backend_process_failed",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"attach": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"terminal_backend_unavailable",
		"terminal_backend_timeout",
		"terminal_backend_process_failed",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"status": {
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"terminal_backend_unavailable",
		"terminal_backend_timeout",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"quiesce-input": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"terminal_backend_timeout",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"wait-safe-boundary": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"quiesce_timeout",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"request-stop": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"stop_timeout",
		"terminal_backend_process_failed",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"terminate-stale": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"terminal_backend_timeout",
		"terminal_backend_process_failed",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
	"restore": {
		"idempotency_mismatch",
		"local_precondition_failed",
		"terminal_backend_unauthorized",
		"terminal_backend_stale_generation",
		"terminal_backend_capability_unproven",
		"terminal_backend_restore_mismatch",
		"terminal_backend_unavailable",
		"terminal_backend_timeout",
		"terminal_backend_process_failed",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
	},
}

// conformanceOperations is the ten-operation vocabulary the error mapping
// is defined over, matching exactly the operations ParseOperation admits
// (pinned by TestDerivedVocabulariesAreExactlyTheAdmittedSets, which
// derives the admitted set from the parser's switch-case list). Both
// direction tests iterate it, so a mirror row can never silently cover
// nine operations while the suite claims ten.
var conformanceOperations = []string{
	"manifest", "probe", "create", "attach", "status",
	"quiesce-input", "wait-safe-boundary", "request-stop",
	"terminate-stale", "restore",
}

// TestCheckErrorAllowedAdmitsEverySpecRowCode drives every code of every
// mirror row through the production gate: the full per-operation sets,
// including the restore mismatch that belongs only to restore and the
// quiesce/stop timeouts that belong only to their rows. An over-strict
// production row reddens here.
func TestCheckErrorAllowedAdmitsEverySpecRowCode(t *testing.T) {
	t.Parallel()

	if len(allowedErrorCodesByOperation) != len(conformanceOperations) {
		t.Fatalf("mirror covers %d operations, want all %d; a missing row admits vacuously",
			len(allowedErrorCodesByOperation), len(conformanceOperations))
	}
	for _, operation := range conformanceOperations {
		codes, known := allowedErrorCodesByOperation[operation]
		if !known || len(codes) == 0 {
			t.Fatalf("mirror has no row for operation %q; a missing row admits vacuously", operation)
		}
		for _, code := range codes {
			if err := terminalbackend.CheckErrorAllowed(operation, code); err != nil {
				t.Errorf("CheckErrorAllowed(%q, %q) error = %v, want admission",
					operation, code, err)
			}
		}
	}
}

// TestCheckErrorAllowedRefusesEveryUnlistedCode is the narrowing arm of
// the error mapping, derived as the complement — not a sample. For every
// operation, every code in the pinned catalog error vocabulary
// (catalog.Current().Errors) that the mirror row withholds must be
// refused with the operation-vocabulary arm, so a widened production row
// reddens here. In particular terminal_backend_unavailable is refused for
// quiesce-input, wait-safe-boundary, request-stop, and terminate-stale,
// the four rows that withhold it.
func TestCheckErrorAllowedRefusesEveryUnlistedCode(t *testing.T) {
	t.Parallel()

	var catalogued []string
	cataloguedSet := map[string]bool{}
	for _, entry := range catalog.Current().Errors {
		code := string(entry.Code)
		catalogued = append(catalogued, code)
		cataloguedSet[code] = true
	}
	if len(catalogued) == 0 {
		t.Fatal("pinned catalog carries zero error codes; the complement proves nothing")
	}
	// The mirror must live inside the catalog: a mirror code outside the
	// pinned vocabulary would be admitted without ever entering the
	// complement domain, and the bijection below would prove nothing.
	for _, operation := range conformanceOperations {
		for _, code := range allowedErrorCodesByOperation[operation] {
			if !cataloguedSet[code] {
				t.Errorf("mirror row %q admits %q, outside the pinned catalog vocabulary; "+
					"the complement cannot pin what it cannot name", operation, code)
			}
		}
	}

	for _, operation := range conformanceOperations {
		allowed := map[string]bool{}
		for _, code := range allowedErrorCodesByOperation[operation] {
			allowed[code] = true
		}
		for _, code := range catalogued {
			if allowed[code] {
				continue
			}
			err := terminalbackend.CheckErrorAllowed(operation, code)
			requireConformanceRefusal(t, err,
				"terminal_backend_protocol_error", "operation error vocabulary")
		}
	}

	// Codes outside any vocabulary are an unbounded string domain, so no
	// test can close them; these probes pin the shape (unknown strings
	// refuse) without claiming the closure the catalog complement gives.
	for _, operation := range conformanceOperations {
		for _, code := range []string{"", "bogus_code", "terminal_backend_bogus"} {
			err := terminalbackend.CheckErrorAllowed(operation, code)
			requireConformanceRefusal(t, err,
				"terminal_backend_protocol_error", "operation error vocabulary")
		}
	}

	if err := terminalbackend.CheckErrorAllowed("launch", "terminal_backend_timeout"); err == nil {
		t.Errorf("CheckErrorAllowed(launch, ...) = nil, want operation vocabulary refusal")
	} else {
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "operation vocabulary")
	}
}

// TestIdempotencyKeyDerivesCanonicalForms proves the per-operation key
// shapes of §4.C and §4.1: the key is derived, never caller-chosen free
// text, and the segment order is fixed.
func TestIdempotencyKeyDerivesCanonicalForms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		operation string
		segments  []string
		want      string
	}{
		{"manifest", []string{"req-1"}, "req-1"},
		{"probe", []string{"req-2"}, "req-2"},
		{"create", []string{conformanceSessionA, "bootstrap-1"},
			conformanceSessionA + "/bootstrap-1"},
		{"restore", []string{conformanceSessionA, "bootstrap-2"},
			conformanceSessionA + "/bootstrap-2"},
		{"attach", []string{"instance-1", "client-1"}, "instance-1/client-1"},
		{"quiesce-input", []string{"instance-1", "quiesce", "gen-1"},
			"instance-1/quiesce/gen-1"},
		{"wait-safe-boundary",
			[]string{"instance-1", "boundary", "gen-1", "provider_quiescence"},
			"instance-1/boundary/gen-1/provider_quiescence"},
		{"request-stop", []string{"instance-1", "stop", "sha256:abc"},
			"instance-1/stop/sha256:abc"},
		{"terminate-stale",
			[]string{"instance-1", "terminate", "lease-1", "41"},
			"instance-1/terminate/lease-1/41"},
		{"status", []string{conformanceSessionA}, conformanceSessionA},
		{"status",
			[]string{conformanceSessionA, "instance-1", "ax.tmux", "1.2.3", "1.0.0", "gen-1"},
			conformanceSessionA + "/instance-1/ax.tmux/1.2.3/1.0.0/gen-1"},
	}
	for _, tc := range cases {
		got, err := terminalbackend.IdempotencyKey(tc.operation, tc.segments...)
		if err != nil {
			t.Errorf("IdempotencyKey(%q, %v) error = %v, want admission",
				tc.operation, tc.segments, err)
			continue
		}
		if got != tc.want {
			t.Errorf("IdempotencyKey(%q, %v) = %q, want %q",
				tc.operation, tc.segments, got, tc.want)
		}
	}
}

// TestIdempotencyKeyRefusesWrongShapes narrows the key gate: a wrong
// segment count or an empty segment is a protocol error, so a key that
// collapses two windows into one cannot be derived.
func TestIdempotencyKeyRefusesWrongShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		operation string
		segments  []string
	}{
		{"manifest", nil},
		{"manifest", []string{"a", "b"}},
		{"probe", nil},
		{"create", []string{"only-session"}},
		{"create", []string{"a", "b", "c"}},
		{"restore", []string{"only-session"}},
		{"attach", []string{"only-instance"}},
		{"attach", []string{"a", "b", "c"}},
		{"quiesce-input", []string{"instance-1", "quiesce"}},
		{"request-stop", []string{"instance-1", "stop"}},
		{"wait-safe-boundary", []string{"instance-1", "boundary", "gen-1"}},
		{"terminate-stale", []string{"instance-1", "terminate", "lease-1"}},
		{"status", nil},
		{"create", []string{"", "bootstrap-1"}},
		{"attach", []string{"instance-1", ""}},
	}
	for _, tc := range cases {
		_, err := terminalbackend.IdempotencyKey(tc.operation, tc.segments...)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "idempotency key shape")
	}

	_, err := terminalbackend.IdempotencyKey("launch", "a")
	requireConformanceRefusal(t, err,
		"terminal_backend_protocol_error", "operation vocabulary")
}

// TestLedgerBindsReplaysAndRefusesMismatch proves the §4.1 receipt
// lifecycle at the production entry: the first bind stores, an identical
// retry replays without a new effect, and a changed operation or result
// in the window is idempotency_mismatch. A missing receipt replays
// nothing, so absence is never proven by the ledger alone.
func TestLedgerBindsReplaysAndRefusesMismatch(t *testing.T) {
	t.Parallel()

	ledger := terminalbackend.NewLedger()
	const key = "session-1/bootstrap-1"

	first, err := ledger.Bind(key, "create", "result-1")
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if first.Key != key || first.Operation != "create" || first.ResultID != "result-1" {
		t.Fatalf("Bind() = %+v, want the stored receipt", first)
	}

	replayed, err := ledger.Bind(key, "create", "result-1")
	if err != nil {
		t.Fatalf("identical Bind() error = %v, want replay", err)
	}
	if replayed != first {
		t.Errorf("identical Bind() = %+v, want replay of %+v", replayed, first)
	}

	stored, known := ledger.Replay(key)
	if !known || stored != first {
		t.Errorf("Replay() = %+v, %v, want %+v, true", stored, known, first)
	}
	if _, known := ledger.Replay("session-1/bootstrap-2"); known {
		t.Errorf("Replay(unknown) = true: a missing receipt proves nothing")
	}

	for _, tc := range []struct {
		name      string
		operation terminalbackend.Operation
		result    string
	}{
		{"changed operation", "restore", "result-1"},
		{"changed result", "create", "result-2"},
		{"changed both", "attach", "result-9"},
	} {
		_, err := ledger.Bind(key, tc.operation, tc.result)
		requireConformanceRefusal(t, err, "idempotency_mismatch", "idempotency key conflict")
	}

	if _, err := ledger.Bind("", "create", "result-1"); err == nil {
		t.Errorf("Bind(empty key) = nil, want key shape refusal")
	} else {
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "idempotency key shape")
	}
	if _, err := ledger.Bind(key, "launch", "result-1"); err == nil {
		t.Errorf("Bind(unknown operation) = nil, want operation vocabulary refusal")
	} else {
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "operation vocabulary")
	}

	var nilLedger *terminalbackend.Ledger
	if _, err := nilLedger.Bind(key, "create", "result-1"); err == nil {
		t.Errorf("nil Bind() = nil, want ledger unavailable")
	} else {
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "idempotency ledger unavailable")
	}
	if _, known := nilLedger.Replay(key); known {
		t.Errorf("nil Replay() = true")
	}
	if exported := nilLedger.Export(); exported != nil {
		t.Errorf("nil Export() = %q, want nil", exported)
	}
}

// TestLedgerSurvivesControllerCrashViaExportImport is the crash evidence
// the deliverable requires for durable-state mutation: receipts bound
// before the crash replay identically from the exported image on a fresh
// ledger, and the export is byte-stable. A truncated, malformed, or
// ambiguous image is refused, never admitted as an empty table.
func TestLedgerSurvivesControllerCrashViaExportImport(t *testing.T) {
	t.Parallel()

	ledger := terminalbackend.NewLedger()
	bindings := []struct {
		key       string
		operation terminalbackend.Operation
		result    string
	}{
		{"session-b/bootstrap-1", "create", "result-b"},
		{"session-a/bootstrap-1", "restore", "result-a"},
		{"instance-1/client-1", "attach", "result-c"},
	}
	for _, binding := range bindings {
		if _, err := ledger.Bind(binding.key, binding.operation, binding.result); err != nil {
			t.Fatalf("Bind(%q) error = %v", binding.key, err)
		}
	}

	first := ledger.Export()
	second := ledger.Export()
	if string(first) != string(second) {
		t.Fatalf("Export() is not byte-stable:\n%q\n%q", first, second)
	}
	// Sorted by key: instance-1, then session-a, then session-b.
	want := "instance-1/client-1\nattach\nresult-c\n" +
		"session-a/bootstrap-1\nrestore\nresult-a\n" +
		"session-b/bootstrap-1\ncreate\nresult-b\n"
	if string(first) != want {
		t.Errorf("Export() = %q, want sorted stable %q", first, want)
	}

	recovered, err := terminalbackend.ImportLedger(first)
	if err != nil {
		t.Fatalf("ImportLedger() error = %v", err)
	}
	for _, binding := range bindings {
		stored, known := recovered.Replay(binding.key)
		if !known {
			t.Errorf("recovered Replay(%q) unknown: the crash lost a bound receipt", binding.key)
			continue
		}
		if stored.Operation != binding.operation || stored.ResultID != binding.result {
			t.Errorf("recovered Replay(%q) = %+v, want operation %q result %q",
				binding.key, stored, binding.operation, binding.result)
		}
	}

	empty, err := terminalbackend.ImportLedger(terminalbackend.NewLedger().Export())
	if err != nil {
		t.Fatalf("ImportLedger(empty) error = %v", err)
	}
	if _, known := empty.Replay("anything"); known {
		t.Errorf("empty ImportLedger replays a receipt")
	}

	for _, image := range [][]byte{
		[]byte("only-one-line\n"),
		[]byte("key\ncreate\n"),
		[]byte("key\nlaunch\nresult-1\n"),
		[]byte("\ncreate\nresult-1\n"),
		[]byte("key\ncreate\n\n"),
		[]byte("key\ncreate\nresult-1\nkey\nrestore\nresult-2\n"),
	} {
		if _, err := terminalbackend.ImportLedger(image); err == nil {
			t.Errorf("ImportLedger(%q) = nil, want refusal", image)
		}
	}
	// A truncated image is an error, never an empty table: the refusal
	// must carry a code, not merely be non-nil.
	_, err = terminalbackend.ImportLedger([]byte("key\ncreate\n"))
	requireConformanceRefusal(t, err,
		"terminal_backend_protocol_error", "idempotency ledger image")
	_, err = terminalbackend.ImportLedger([]byte("key\ncreate\nresult-1\nkey\nrestore\nresult-2\n"))
	requireConformanceRefusal(t, err,
		"idempotency_mismatch", "idempotency ledger image")
}

// TestParseAttachAuthorizationAdmitsClosedFixture proves the gate is
// reachable: the exact six-member ownership-neutral policy parses.
func TestParseAttachAuthorizationAdmitsClosedFixture(t *testing.T) {
	t.Parallel()

	auth := testAttachAuthorization(t)
	if auth.PolicyEvidenceID != testDigest {
		t.Errorf("PolicyEvidenceID = %q, want %q", auth.PolicyEvidenceID, testDigest)
	}
	if auth.AuthorizingHostID != conformanceHost {
		t.Errorf("AuthorizingHostID = %q, want %q", auth.AuthorizingHostID, conformanceHost)
	}
	if auth.Transport != "local_only" {
		t.Errorf("Transport = %q, want local_only", auth.Transport)
	}
	if !auth.InputAuthorized {
		t.Errorf("InputAuthorized = false, want true")
	}
	if auth.IssuedAt.String() != conformanceIssued {
		t.Errorf("IssuedAt = %q, want %q", auth.IssuedAt, conformanceIssued)
	}
	if auth.ExpiresAt.String() != conformanceExpires {
		t.Errorf("ExpiresAt = %q, want %q", auth.ExpiresAt, conformanceExpires)
	}
}

// TestAttachAuthorizationCarriesNoLease is the structural half of
// ownership neutrality: the policy struct has exactly the six closed
// members, so a lease, epoch, holder, or authorization-kind field cannot
// be added without reddening here. The parsing half below refuses such
// a member in a document.
func TestAttachAuthorizationCarriesNoLease(t *testing.T) {
	t.Parallel()

	fields := reflect.TypeOf(terminalbackend.AttachAuthorization{}).NumField()
	if fields != 6 {
		t.Errorf("AttachAuthorization has %d fields, want 6: "+
			"the attach policy carries no lease member", fields)
	}
	authType := reflect.TypeOf(terminalbackend.AttachAuthorization{})
	for _, forbidden := range []string{"Lease", "LeaseID", "LeaseEpoch", "Holder", "Owner", "Ownership"} {
		for index := 0; index < authType.NumField(); index++ {
			if strings.Contains(authType.Field(index).Name, forbidden) {
				t.Errorf("AttachAuthorization.%s names %q: "+
					"the attach policy cannot carry ownership",
					authType.Field(index).Name, forbidden)
			}
		}
	}
}

// TestParseAttachAuthorizationRefusesNonNeutralPolicy is the attack the
// neutrality rule exists for: a lease tuple smuggled into an attach
// policy, whether under its AXAuthorization names or any other unknown
// member, is refused as an unknown member before any other check runs.
// Missing members, mistyped members, and an expiry that is not strictly
// after issue are refused alongside.
func TestParseAttachAuthorizationRefusesNonNeutralPolicy(t *testing.T) {
	t.Parallel()

	base := map[string]string{
		"policy_evidence_id":  `"` + testDigest + `"`,
		"authorizing_host_id": `"` + conformanceHost + `"`,
		"transport":           `"local_only"`,
		"input_authorized":    `true`,
		"issued_at":           `"` + conformanceIssued + `"`,
		"expires_at":          `"` + conformanceExpires + `"`,
	}
	build := func(override map[string]string, extra string) []byte {
		members := []string{}
		for _, name := range []string{
			"policy_evidence_id", "authorizing_host_id", "transport",
			"input_authorized", "issued_at", "expires_at",
		} {
			value := base[name]
			if replacement, ok := override[name]; ok {
				if replacement == "" {
					continue
				}
				value = replacement
			}
			members = append(members, `"`+name+`":`+value)
		}
		if extra != "" {
			members = append(members, extra)
		}
		return []byte("{" + strings.Join(members, ",") + "}")
	}

	// Every lease-shaped smuggling attempt, refused as an unknown member.
	for _, extra := range []string{
		`"lease_id":"0198f4c8-8e50-4f66-8f70-1234567890ab"`,
		`"lease_epoch":41`,
		`"holder_host_id":"` + conformanceHost + `"`,
		`"authorization_kind":"control"`,
		`"authorization_evidence_id":"` + testDigest + `"`,
		`"owner":"` + conformanceHost + `"`,
	} {
		_, err := terminalbackend.ParseAttachAuthorization(build(nil, extra))
		requireConformanceRefusal(t, err,
			"terminal_backend_manifest_probe_mismatch", "document members")
	}

	// Missing and mistyped members.
	_, err := terminalbackend.ParseAttachAuthorization(build(map[string]string{"transport": ""}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_manifest_probe_mismatch", "document members")
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{"input_authorized": `"yes"`}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_manifest_probe_mismatch", "document member type")
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{"transport": `"courier_pigeon"`}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_protocol_error", "presentation transport vocabulary")
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{"transport": `"ssh_tunnel"`}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_protocol_error", "presentation transport vocabulary")
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{"policy_evidence_id": `"not-a-digest"`}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_manifest_probe_mismatch", "document digest")
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{"authorizing_host_id": `"not-a-uuid"`}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_manifest_probe_mismatch", "document member type")

	// Expiry equal to or before issue authorizes nothing.
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{"expires_at": `"` + conformanceIssued + `"`}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_unauthorized", "attach authorization expiry")
	_, err = terminalbackend.ParseAttachAuthorization(
		build(map[string]string{
			"issued_at":  `"` + conformanceExpires + `"`,
			"expires_at": `"` + conformanceIssued + `"`,
		}, ""))
	requireConformanceRefusal(t, err,
		"terminal_backend_unauthorized", "attach authorization expiry")

	// A malformed read is an error, never an absent policy.
	for _, raw := range [][]byte{
		[]byte(`not json`),
		[]byte(`[1,2,3]`),
		[]byte(`{"policy_evidence_id":"` + testDigest + `"}`),
	} {
		if _, err := terminalbackend.ParseAttachAuthorization(raw); err == nil {
			t.Errorf("ParseAttachAuthorization(%q) = nil, want refusal", raw)
		}
	}
}

// TestCheckAttachRequestBindsTransportInputAndExpiry proves the attach
// row at the production entry: the request must equal the authorized
// transport and input boolean, a relay transport is refused even when
// named, and an expired policy authorizes nothing.
func TestCheckAttachRequestBindsTransportInputAndExpiry(t *testing.T) {
	t.Parallel()

	auth := testAttachAuthorization(t)
	if err := terminalbackend.CheckAttachRequest(auth, "local_only", true, conformanceNow()); err != nil {
		t.Errorf("CheckAttachRequest() error = %v, want admission", err)
	}

	mismatched, err := terminalbackend.ParseAttachAuthorization([]byte(
		`{"policy_evidence_id":"` + testDigest + `",` +
			`"authorizing_host_id":"` + conformanceHost + `",` +
			`"transport":"trusted_private_mesh",` +
			`"input_authorized":false,` +
			`"issued_at":"` + conformanceIssued + `",` +
			`"expires_at":"` + conformanceExpires + `"}`))
	if err != nil {
		t.Fatalf("ParseAttachAuthorization() error = %v", err)
	}
	if err := terminalbackend.CheckAttachRequest(
		mismatched, "trusted_private_mesh", false, conformanceNow()); err != nil {
		t.Errorf("CheckAttachRequest(mesh, false) error = %v, want admission", err)
	}

	wrongTransport := terminalbackend.CheckAttachRequest(auth, "trusted_private_mesh", true, conformanceNow())
	requireConformanceRefusal(t, wrongTransport,
		"terminal_backend_unauthorized", "attach authorization binding")
	wrongInput := terminalbackend.CheckAttachRequest(auth, "local_only", false, conformanceNow())
	requireConformanceRefusal(t, wrongInput,
		"terminal_backend_unauthorized", "attach authorization binding")
	unknownTransport := terminalbackend.CheckAttachRequest(auth, "courier_pigeon", true, conformanceNow())
	requireConformanceRefusal(t, unknownTransport,
		"terminal_backend_protocol_error", "presentation transport vocabulary")

	expired := conformanceNow().Add(48 * time.Hour)
	requireConformanceRefusal(t,
		terminalbackend.CheckAttachRequest(auth, "local_only", true, expired),
		"terminal_backend_unauthorized", "attach authorization expiry")

	// Production refuses at the expiry instant (!now.Before(expires)), so
	// the boundary is pinned from both sides: admitted just before,
	// refused exactly at. Flipping the check to now.After(expires)
	// admits the instant and reddens the refusal below.
	expiry, err := time.Parse(time.RFC3339Nano, conformanceExpires)
	if err != nil {
		t.Fatalf("parse conformance expiry: %v", err)
	}
	if err := terminalbackend.CheckAttachRequest(auth, "local_only", true, expiry.Add(-time.Second)); err != nil {
		t.Errorf("CheckAttachRequest(just before expiry) error = %v, want admission", err)
	}
	requireConformanceRefusal(t,
		terminalbackend.CheckAttachRequest(auth, "local_only", true, expiry),
		"terminal_backend_unauthorized", "attach authorization expiry")

	relayAuth, err := terminalbackend.ParseAttachAuthorization([]byte(
		`{"policy_evidence_id":"` + testDigest + `",` +
			`"authorizing_host_id":"` + conformanceHost + `",` +
			`"transport":"third_party_relay",` +
			`"input_authorized":false,` +
			`"issued_at":"` + conformanceIssued + `",` +
			`"expires_at":"` + conformanceExpires + `"}`))
	if err != nil {
		t.Fatalf("ParseAttachAuthorization(relay) error = %v", err)
	}
	requireConformanceRefusal(t,
		terminalbackend.CheckAttachRequest(relayAuth, "third_party_relay", false, conformanceNow()),
		"terminal_backend_unauthorized", "attach relay transport")
}

// TestCheckAttachResultRequiresTripleEquality proves the result rule: the
// reported input boolean must equal both the request and the policy. A
// backend granting input the policy withheld is refused even when the
// request itself was authorized.
func TestCheckAttachResultRequiresTripleEquality(t *testing.T) {
	t.Parallel()

	auth := testAttachAuthorization(t)
	if err := terminalbackend.CheckAttachResult(true, true, auth); err != nil {
		t.Errorf("CheckAttachResult(true, true) error = %v, want admission", err)
	}

	granted := terminalbackend.CheckAttachResult(false, true, auth)
	requireConformanceRefusal(t, granted,
		"terminal_backend_unauthorized", "attach input binding")
	revoked := terminalbackend.CheckAttachResult(true, false, auth)
	requireConformanceRefusal(t, revoked,
		"terminal_backend_unauthorized", "attach input binding")
	// Only the auth conjunct catches this one: request and result agree
	// (both false) but the policy authorized input, so dropping the
	// resultInputAuthorized != auth.InputAuthorized leg admits it.
	withheld := terminalbackend.CheckAttachResult(false, false, auth)
	requireConformanceRefusal(t, withheld,
		"terminal_backend_unauthorized", "attach input binding")
}

// TestCheckEntrypointAdmitsOnlyAxPane proves the §4.A stable entrypoint:
// exactly ax, pane, and the canonical session string. A raw provider
// command, a reordered argv, or a session that is not the one under
// creation cannot serve as the durable entry point.
func TestCheckEntrypointAdmitsOnlyAxPane(t *testing.T) {
	t.Parallel()

	if err := terminalbackend.CheckEntrypoint(
		[]string{"ax", "pane", conformanceSessionA}, conformanceSessionA); err != nil {
		t.Errorf("CheckEntrypoint() error = %v, want admission", err)
	}

	refused := []struct {
		name    string
		argv    []string
		session string
	}{
		{"raw provider command", []string{"codex", "--session", conformanceSessionA}, conformanceSessionA},
		{"provider binary with pane", []string{"codex", "pane", conformanceSessionA}, conformanceSessionA},
		{"reordered argv", []string{"pane", "ax", conformanceSessionA}, conformanceSessionA},
		{"uppercase binary", []string{"AX", "pane", conformanceSessionA}, conformanceSessionA},
		{"missing session", []string{"ax", "pane"}, conformanceSessionA},
		{"extra argv", []string{"ax", "pane", conformanceSessionA, "--extra"}, conformanceSessionA},
		{"empty argv", nil, conformanceSessionA},
		{"wrong session", []string{"ax", "pane", conformanceSessionB}, conformanceSessionA},
		{"non-uuid session", []string{"ax", "pane", "not-a-session"}, "not-a-session"},
		{"uuidv4 session", []string{"ax", "pane", "0198f4c8-8e50-4f66-8f70-1234567890ab"},
			"0198f4c8-8e50-4f66-8f70-1234567890ab"},
	}
	for _, tc := range refused {
		err := terminalbackend.CheckEntrypoint(tc.argv, tc.session)
		if err == nil {
			t.Errorf("CheckEntrypoint(%q, %q) = nil, want refusal", tc.argv, tc.session)
			continue
		}
		var refusal *terminalbackend.Error
		if !errors.As(err, &refusal) || refusal.Code != "local_precondition_failed" {
			t.Errorf("CheckEntrypoint(%q, %q) = %v, want local_precondition_failed",
				tc.argv, tc.session, err)
			continue
		}
		if refusal.Detail != "entrypoint argv" && refusal.Detail != "entrypoint session binding" {
			t.Errorf("CheckEntrypoint(%q, %q) detail = %q, want an entrypoint clause",
				tc.argv, tc.session, refusal.Detail)
		}
	}
}

// TestCheckEntrypointNamesItsArm pins the two refusal clauses separately:
// an argv-shape violation and a session-binding violation are distinct
// arms, so merging them reddens here.
func TestCheckEntrypointNamesItsArm(t *testing.T) {
	t.Parallel()

	err := terminalbackend.CheckEntrypoint(
		[]string{"codex", conformanceSessionA}, conformanceSessionA)
	requireConformanceRefusal(t, err, "local_precondition_failed", "entrypoint argv")

	err = terminalbackend.CheckEntrypoint(
		[]string{"ax", "pane", conformanceSessionB}, conformanceSessionA)
	requireConformanceRefusal(t, err, "local_precondition_failed", "entrypoint session binding")
}

// TestCheckStatusResultEnforcesLookupRules proves the §4.C status result
// contract: the canonical false form on non-match with no fallback, null
// provider observation unless requested and evidenced, and attachability
// only in parked or active with evidenced attach capability.
func TestCheckStatusResultEnforcesLookupRules(t *testing.T) {
	t.Parallel()

	present := true
	lastOp := "0198f4c8-8e50-7f66-8f70-1234567890ab"
	lastEffect := terminalbackend.EffectWrapperStarted
	matched := terminalbackend.StatusResult{
		State:             "active",
		IdentityMatch:     true,
		WrapperPresent:    true,
		ProviderPresent:   &present,
		Attachable:        true,
		LastOperationID:   &lastOp,
		LastEffect:        &lastEffect,
		ProviderRequested: true,
		ProviderEvidenced: true,
		AttachEvidenced:   true,
	}
	if err := terminalbackend.CheckStatusResult(true, matched); err != nil {
		t.Errorf("CheckStatusResult(match) error = %v, want admission", err)
	}

	// The canonical false form is admitted and authorizes no fallback.
	canonical := terminalbackend.StatusResult{State: "absent"}
	if err := terminalbackend.CheckStatusResult(false, canonical); err != nil {
		t.Errorf("CheckStatusResult(canonical false) error = %v, want admission", err)
	}

	// Every deviation from the false form is refused.
	nonCanonical := []terminalbackend.StatusResult{
		{State: "active"},
		{State: "absent", IdentityMatch: true},
		{State: "absent", WrapperPresent: true},
		{State: "absent", ProviderPresent: &present},
		{State: "absent", Attachable: true},
		{State: "absent", LastOperationID: &lastOp},
		{State: "absent", LastEffect: &lastEffect},
	}
	for _, result := range nonCanonical {
		err := terminalbackend.CheckStatusResult(false, result)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "status identity binding")
	}

	// provider_present without request, or without evidence, is refused.
	unrequested := matched
	unrequested.ProviderRequested = false
	requireConformanceRefusal(t,
		terminalbackend.CheckStatusResult(true, unrequested),
		"terminal_backend_protocol_error", "status provider observation")
	unevidenced := matched
	unevidenced.ProviderEvidenced = false
	requireConformanceRefusal(t,
		terminalbackend.CheckStatusResult(true, unevidenced),
		"terminal_backend_protocol_error", "status provider observation")

	// Attachability is a two-state rule over the closed state enum, and
	// the complement is derived, not sampled: every state the production
	// parser admits (derived from its switch-case list, pinned by
	// TestDerivedVocabulariesAreExactlyTheAdmittedSets) outside
	// parked/active is refused with evidenced attach capability held, so
	// widening the admitted set by one member reddens here. A sample
	// naming one excluded state would prove the shape, not the rule.
	states := derivedParserVocabulary(t, "conformance.go", "ParseInstanceState")
	if len(states) != 8 {
		t.Fatalf("derived %d instance states, want the closed eight; "+
			"the attachability complement cannot be derived", len(states))
	}
	for _, state := range states {
		probe := matched
		probe.State = terminalbackend.InstanceState(state)
		switch state {
		case "parked", "active":
			probe.ProviderPresent = nil
			probe.ProviderRequested = false
			probe.ProviderEvidenced = false
			if err := terminalbackend.CheckStatusResult(true, probe); err != nil {
				t.Errorf("CheckStatusResult(%s, evidenced) error = %v, want admission", state, err)
			}
		default:
			requireConformanceRefusal(t,
				terminalbackend.CheckStatusResult(true, probe),
				"local_precondition_failed", "status attachability")
		}
	}
	unproven := matched
	unproven.AttachEvidenced = false
	requireConformanceRefusal(t,
		terminalbackend.CheckStatusResult(true, unproven),
		"local_precondition_failed", "status attachability")

	// An unknown state or effect is a vocabulary refusal, never a pass.
	bogus := matched
	bogus.State = "running"
	requireConformanceRefusal(t,
		terminalbackend.CheckStatusResult(true, bogus),
		"terminal_backend_protocol_error", "lifecycle state vocabulary")
	bogusEffect := terminalbackend.SideEffect("side_effect_unknown")
	bogus = matched
	bogus.LastEffect = &bogusEffect
	requireConformanceRefusal(t,
		terminalbackend.CheckStatusResult(true, bogus),
		"terminal_backend_protocol_error", "side effect vocabulary")
}

// TestReplicationClassificationIsClosed proves the §4.E table through the
// public entry: every listed member classifies exactly once, and the
// classes are disjoint. Moving one member between classes reddens here
// before any replication decision can silently change. This direction
// alone proves want ⊆ production; the reverse direction and the size pin
// live in TestReplicationMembersAreExactlyTheClosedTable, which reads the
// live table from inside the package, so adding a member reddens there
// even though this iteration cannot observe it.
func TestReplicationClassificationIsClosed(t *testing.T) {
	t.Parallel()

	want := map[string]terminalbackend.ReplicationClass{
		"manifest_id": "safe_evidence", "probe_id": "safe_evidence",
		"evidence_id":               "safe_evidence",
		"terminal_backend_id":       "safe_evidence",
		"implementation_version":    "safe_evidence",
		"protocol_version":          "safe_evidence",
		"protocol_versions":         "safe_evidence",
		"platform":                  "safe_evidence",
		"platforms":                 "safe_evidence",
		"os_version":                "safe_evidence",
		"conformance_fixture_id":    "safe_evidence",
		"capability_claims":         "safe_evidence",
		"evidence_ids":              "safe_evidence",
		"backend_generation_digest": "safe_evidence",
		"capability":                "safe_evidence",
		"dependent_operations":      "safe_evidence",
		"evidence_requirements":     "safe_evidence",
		"facts":                     "safe_evidence",
		"issuer":                    "safe_evidence",
		"issuer_id":                 "safe_evidence",
		"observed_at":               "safe_evidence",
		"expires_at":                "safe_evidence",

		"binding_id": "host_local", "terminal_instance_id": "host_local",
		"session_id": "host_local", "host_id": "host_local",
		"host_incarnation_id":   "host_local",
		"backend_generation":    "host_local",
		"native_reference":      "host_local",
		"idempotency_receipt":   "host_local",
		"last_effect":           "host_local",
		"last_operation_id":     "host_local",
		"supersedes_binding_id": "host_local",
		"created_at":            "host_local",

		"availability": "derived_cache", "attachable": "derived_cache",
		"wrapper_present": "derived_cache", "provider_present": "derived_cache",
		"identity_match": "derived_cache", "state": "derived_cache",
		"probed_at": "derived_cache",

		"attach_descriptor":   "sensitive_owner_only",
		"ipc_handle":          "sensitive_owner_only",
		"tmux_socket":         "sensitive_owner_only",
		"named_pipe":          "sensitive_owner_only",
		"attach_credential":   "sensitive_owner_only",
		"relay_credential":    "sensitive_owner_only",
		"backend_auth":        "sensitive_owner_only",
		"auth_database":       "sensitive_owner_only",
		"gui_attestation":     "sensitive_owner_only",
		"login_attestation":   "sensitive_owner_only",
		"provider_credential": "sensitive_owner_only",
		"evidence_secret":     "sensitive_owner_only",
		"environment_values":  "sensitive_owner_only",

		"native_pid": "forbidden", "process_handle": "forbidden",
		"endpoint": "forbidden", "token": "forbidden",
		"terminal_output": "forbidden", "scrollback": "forbidden",
		"unrestricted_environment": "forbidden",
		"credential_detail":        "forbidden",
		"live_process_fact":        "forbidden",
		"raw_native_reference":     "forbidden",
	}
	for member, class := range want {
		got, known := terminalbackend.ClassifyReplication(member)
		if !known {
			t.Errorf("ClassifyReplication(%q) unknown, want %q", member, class)
			continue
		}
		if got != class {
			t.Errorf("ClassifyReplication(%q) = %q, want %q", member, got, class)
		}
	}
	for _, unknown := range []string{"", "session_record", "MANIFEST_ID", "manifest_id "} {
		if _, known := terminalbackend.ClassifyReplication(unknown); known {
			t.Errorf("ClassifyReplication(%q) known: unknown members must not classify", unknown)
		}
	}
}

// TestCheckReplicableRefusesEveryNonSafeClass proves the exclusion at the
// production entry: one witness from each non-safe class plus an unknown
// member is refused, while a safe-evidence set (and the empty set, which
// replicates nothing) is admitted.
func TestCheckReplicableRefusesEveryNonSafeClass(t *testing.T) {
	t.Parallel()

	safe := []string{
		"manifest_id", "probe_id", "evidence_id",
		"terminal_backend_id", "implementation_version", "protocol_version",
		"platform", "conformance_fixture_id", "capability_claims",
		"backend_generation_digest", "facts", "issuer_id",
	}
	if err := terminalbackend.CheckReplicable(safe); err != nil {
		t.Errorf("CheckReplicable(safe) error = %v, want admission", err)
	}
	if err := terminalbackend.CheckReplicable(nil); err != nil {
		t.Errorf("CheckReplicable(nil) error = %v, want admission", err)
	}

	for _, member := range []string{
		"binding_id", "terminal_instance_id", "backend_generation",
		"attachable", "availability", "probed_at",
		"attach_descriptor", "tmux_socket", "provider_credential",
		"native_pid", "terminal_output", "scrollback", "token",
		"credential_detail", "live_process_fact",
		"tomorrow_s_exclusion",
	} {
		err := terminalbackend.CheckReplicable([]string{"manifest_id", member})
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "replication exclusion")
	}
}

// TestHistoricalTranslationMapsOnlyTheImmutablePair proves the §4.E
// translation through the production entry: the two immutable v0.4.3
// values map forward with legacy_unreported version and generation. The
// refused list is a sampling bound over an unbounded string domain, not a
// closure — it pins the refusal shape, including the reviewer's escape
// probe "screen". The closure is pinned white-box by
// TestLegacyForwardIsExactlyTheImmutablePair, which holds the live map to
// exactly two rows plus the size: a third translating name reddens there
// deterministically, where a reverse-projection sample is a coin flip over
// map iteration order.
func TestHistoricalTranslationMapsOnlyTheImmutablePair(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		legacy string
		wantID string
	}{
		{"tmux", terminalbackend.BuiltinTmux},
		{"conpty", terminalbackend.BuiltinConpty},
	} {
		binding, err := terminalbackend.TranslateLegacyBackend(tc.legacy)
		if err != nil {
			t.Errorf("TranslateLegacyBackend(%q) error = %v, want admission", tc.legacy, err)
			continue
		}
		if binding.BackendID != tc.wantID {
			t.Errorf("TranslateLegacyBackend(%q).BackendID = %q, want %q",
				tc.legacy, binding.BackendID, tc.wantID)
		}
		if binding.ImplementationVersion != terminalbackend.LegacyUnreported ||
			binding.Generation != terminalbackend.LegacyUnreported {
			t.Errorf("TranslateLegacyBackend(%q) = %+v, want legacy_unreported version and generation",
				tc.legacy, binding)
		}
	}

	for _, legacy := range []string{
		"", "TMUX", "Tmux", "tmux ", " conpty",
		"ax.tmux", "ax.conpty", "superlogical", "ghostty", "wezterm",
		"screen",
	} {
		_, err := terminalbackend.TranslateLegacyBackend(legacy)
		requireConformanceRefusal(t, err,
			"incompatible_schema", "legacy backend identity")
	}
}

// TestLegacyReverseProjectionExistsOnlyForThePair proves the reverse
// direction: the two canonical built-ins project back, every other valid
// ID is incompatible rather than fallback, and a malformed identity is
// refused by the registry grammar first.
func TestLegacyReverseProjectionExistsOnlyForThePair(t *testing.T) {
	t.Parallel()

	legacy, err := terminalbackend.ProjectToLegacy(terminalbackend.BuiltinTmux)
	if err != nil || legacy != "tmux" {
		t.Errorf("ProjectToLegacy(ax.tmux) = %q, %v, want tmux, nil", legacy, err)
	}
	legacy, err = terminalbackend.ProjectToLegacy(terminalbackend.BuiltinConpty)
	if err != nil || legacy != "conpty" {
		t.Errorf("ProjectToLegacy(ax.conpty) = %q, %v, want conpty, nil", legacy, err)
	}

	_, err = terminalbackend.ProjectToLegacy("vendor.term")
	requireConformanceRefusal(t, err, "incompatible_schema", "legacy reverse projection")

	_, err = terminalbackend.ProjectToLegacy("vendor.screen")
	requireConformanceRefusal(t, err, "incompatible_schema", "legacy reverse projection")

	// A reserved-namespace squat is not an ID at all: the registry
	// grammar refuses it before projection can consider it.
	_, err = terminalbackend.ProjectToLegacy("ax.tmuxx")
	requireConformanceRefusal(t, err,
		"terminal_backend_not_found", "terminal_backend_id reserved namespace")

	_, err = terminalbackend.ProjectToLegacy("TMUX")
	requireConformanceRefusal(t, err,
		"terminal_backend_not_found", "terminal_backend_id grammar")
}

// TestLegacyBindingCarriesNoCapabilities is the structural proof that
// translation infers nothing: the binding has exactly three fields and
// none of them names a capability, claim, or evidence set. Forward and
// reverse round-trip without touching history: the argument is unchanged
// by construction of a pure function.
func TestLegacyBindingCarriesNoCapabilities(t *testing.T) {
	t.Parallel()

	bindingType := reflect.TypeOf(terminalbackend.LegacyBinding{})
	if bindingType.NumField() != 3 {
		t.Errorf("LegacyBinding has %d fields, want 3: "+
			"translation carries no inferred capabilities", bindingType.NumField())
	}
	for index := 0; index < bindingType.NumField(); index++ {
		name := bindingType.Field(index).Name
		for _, forbidden := range []string{"Capabilit", "Claim", "Evidence", "Probe", "Manifest"} {
			if strings.Contains(name, forbidden) {
				t.Errorf("LegacyBinding.%s names %q: translation infers no capabilities",
					name, forbidden)
			}
		}
	}

	for _, legacy := range []string{"tmux", "conpty"} {
		before := legacy
		binding, err := terminalbackend.TranslateLegacyBackend(legacy)
		if err != nil {
			t.Fatalf("TranslateLegacyBackend(%q) error = %v", legacy, err)
		}
		if legacy != before {
			t.Errorf("TranslateLegacyBackend mutated its argument")
		}
		back, err := terminalbackend.ProjectToLegacy(binding.BackendID)
		if err != nil {
			t.Fatalf("ProjectToLegacy(%q) error = %v", binding.BackendID, err)
		}
		if back != legacy {
			t.Errorf("round trip %q -> %q -> %q, want identity", legacy, binding.BackendID, back)
		}
	}
}

// TestAdmitProbeRefusesNilRegistry closes a gap the refusal-arm
// derivation exposed: no test drove the nil-registry arm of AdmitProbe.
// The nil check runs before any parsing, so no document fixture is
// needed to reach it.
func TestAdmitProbeRefusesNilRegistry(t *testing.T) {
	t.Parallel()

	var nilRegistry *terminalbackend.Registry
	_, err := nilRegistry.AdmitProbe(nil, nil, nil, "", time.Now().UTC(), nil)
	requireConformanceRefusal(t, err,
		"terminal_backend_not_found", "registry unavailable")
}

// TestUnsignedEvidenceBytesNeverFailsOnSchemaAdmittedInput states the
// bound the two "evidence canonical bytes" arms carry: UnsignedEvidenceBytes
// serializes a fixed map of strings, booleans, and nulls built from typed
// Evidence, so neither encoding/json (which fails only on channels,
// functions, or Marshaler errors) nor the JCS transform (which fails only
// on malformed input, and Marshal output is always well-formed) can fail
// on schema-admitted input. The arms stay as fail-closed defense; this
// test proves the bound by driving the nastiest strings the schema
// admits — invalid UTF-8, empty, and maximal — and requiring success.
// If Evidence ever gains a field type outside string/bool/null, this
// bound must be re-proven, not assumed.
func TestUnsignedEvidenceBytesNeverFailsOnSchemaAdmittedInput(t *testing.T) {
	t.Parallel()

	platform, err := scalar.ParsePlatform("linux")
	if err != nil {
		t.Fatalf("ParsePlatform() error = %v", err)
	}
	observed, err := scalar.ParseTimestamp(conformanceIssued)
	if err != nil {
		t.Fatalf("ParseTimestamp() error = %v", err)
	}
	expires, err := scalar.ParseTimestamp(conformanceExpires)
	if err != nil {
		t.Fatalf("ParseTimestamp() error = %v", err)
	}
	for _, osVersion := range []string{
		"15.6.1",
		"",
		strings.Repeat("v", 256),
		"invalid-utf8-\xff\xfe-lone-surrogate-\xed\xa0\x80",
		"controls-\x00\x01\x02-newline-\n-tab-\t",
	} {
		evidence := terminalbackend.Evidence{
			TerminalBackendID:       "com.example.term",
			ImplementationVersion:   "1.2.3",
			ProtocolVersion:         "1.0.0",
			BackendGenerationDigest: testDigest,
			Capability:              "graceful_stop",
			Platform:                platform,
			OSVersion:               osVersion,
			ConformanceFixtureID:    testDigest,
			ObservedAt:              observed,
			ExpiresAt:               expires,
			Issuer:                  "ax_local_probe",
			IssuerID:                testDigest,
			Facts:                   []string{"fixture_passed", "runtime_probe_passed"},
		}
		unsigned, err := terminalbackend.UnsignedEvidenceBytes(evidence)
		if err != nil {
			t.Errorf("UnsignedEvidenceBytes(os_version=%q) error = %v, want success: "+
				"the evidence canonical bytes arms are dead on schema-admitted input", osVersion, err)
			continue
		}
		if len(unsigned) == 0 {
			t.Errorf("UnsignedEvidenceBytes(os_version=%q) returned no bytes", osVersion)
		}
	}
}

// TestParseSideEffectAdmitsOnlyTheTenClosedEffects drives the last
// vocabulary the harness parses: every member of the §4.C effect enum is
// admitted. The refused list is a sampling bound over an unbounded
// string domain, not a closure — it pins the refusal shape. The closure
// is derived: the admitted set must equal the parser's switch-case list
// exactly (pinned by TestDerivedVocabulariesAreExactlyTheAdmittedSets),
// so an eleventh admitted effect reddens there even though no refused
// sample names it.
func TestParseSideEffectAdmitsOnlyTheTenClosedEffects(t *testing.T) {
	t.Parallel()

	for _, effect := range []string{
		"binding_persisted", "wrapper_started", "attach_client_created",
		"input_closed", "safe_boundary_observed", "graceful_stop_requested",
		"process_closed", "backend_store_closed",
		"stale_incarnation_terminated", "wrapper_restored",
	} {
		if _, err := terminalbackend.ParseSideEffect(effect); err != nil {
			t.Errorf("ParseSideEffect(%q) error = %v, want admission", effect, err)
		}
	}
	for _, effect := range []string{"", "wrapper_started ", "process-closed", "pane_started", "input_reopened"} {
		_, err := terminalbackend.ParseSideEffect(effect)
		requireConformanceRefusal(t, err,
			"terminal_backend_protocol_error", "side effect vocabulary")
	}
}

// derivedParserVocabulary derives the admitted set of a closed-vocabulary
// parser from the production AST: every case-list identifier of the
// function's switch, resolved through the file's string const
// declarations. It fails closed on an unresolvable case, a missing or
// doubled switch, a missing default, or an empty set, so a production
// vocabulary the test cannot read is a red rather than a vacuous pass.
// A case written as a conversion (SideEffect("input_reopened")) instead
// of a const identifier is unresolvable by construction and fails here.
func derivedParserVocabulary(t *testing.T, file, function string) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v; an unparseable derivation proves nothing", file, err)
	}
	constValues := map[string]string{}
	for _, declaration := range parsed.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			literal, ok := value.Values[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			wire, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatalf("unquote const %s: %v", value.Names[0].Name, err)
			}
			constValues[value.Names[0].Name] = wire
		}
	}
	if len(constValues) == 0 {
		t.Fatalf("resolved zero string consts from %s; the scanner is broken, not the package", file)
	}
	var target *ast.FuncDecl
	for _, declaration := range parsed.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if !ok || candidate.Name.Name != function {
			continue
		}
		target = candidate
	}
	if target == nil || target.Body == nil {
		t.Fatalf("parser %s not found in %s; the derivation cannot name its vocabulary", function, file)
	}
	switches := []*ast.SwitchStmt{}
	ast.Inspect(target.Body, func(node ast.Node) bool {
		switchStatement, ok := node.(*ast.SwitchStmt)
		if ok {
			switches = append(switches, switchStatement)
		}
		return true
	})
	if len(switches) != 1 {
		t.Fatalf("parser %s holds %d switches, want exactly one; the derivation cannot name its vocabulary",
			function, len(switches))
	}
	derived := []string{}
	defaults := 0
	for _, statement := range switches[0].Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok {
			t.Fatalf("parser %s switch holds a non-case statement; the derivation cannot name its vocabulary", function)
		}
		if clause.List == nil {
			defaults++
			continue
		}
		for _, expression := range clause.List {
			identifier, ok := expression.(*ast.Ident)
			if !ok {
				t.Fatalf("parser %s case is not a const identifier; "+
					"an admittable member the derivation cannot name ships unwitnessed", function)
			}
			wire, known := constValues[identifier.Name]
			if !known {
				t.Fatalf("parser %s case %q resolves to no string const; "+
					"an admittable member the derivation cannot name ships unwitnessed",
					function, identifier.Name)
			}
			derived = append(derived, wire)
		}
	}
	if defaults != 1 {
		t.Fatalf("parser %s holds %d defaults, want exactly one refusal; the vocabulary is not closed", function, defaults)
	}
	if len(derived) == 0 {
		t.Fatalf("derived zero admitted members for %s; the scanner is broken, not the package", function)
	}
	seen := map[string]bool{}
	for _, member := range derived {
		if seen[member] {
			t.Fatalf("parser %s admits %q twice; the derivation cannot name a duplicated vocabulary", function, member)
		}
		seen[member] = true
	}
	return derived
}

// TestDerivedVocabulariesAreExactlyTheAdmittedSets pins the four closed
// vocabularies the harness parses by derivation rather than sampling:
// the admitted set derived from each parser's switch-case list must
// equal the exact member list in both directions, so an eleventh
// operation, a ninth state, an eleventh effect, or a fourth transport
// reddens here even though no refused sample names it. The operation
// direction additionally pins conformanceOperations, the vocabulary the
// error-mapping tests iterate, to the derived operation set.
func TestDerivedVocabulariesAreExactlyTheAdmittedSets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		function string
		want     []string
	}{
		{"ParseInstanceState", []string{
			"absent", "creating", "parked", "active",
			"quiescing", "stopped", "stale_fenced", "unavailable",
		}},
		{"ParseOperation", []string{
			"manifest", "probe", "create", "attach", "status",
			"quiesce-input", "wait-safe-boundary", "request-stop",
			"terminate-stale", "restore",
		}},
		{"ParseSideEffect", []string{
			"binding_persisted", "wrapper_started", "attach_client_created",
			"input_closed", "safe_boundary_observed", "graceful_stop_requested",
			"process_closed", "backend_store_closed",
			"stale_incarnation_terminated", "wrapper_restored",
		}},
		{"parseTransport", []string{
			"local_only", "trusted_private_mesh", "third_party_relay",
		}},
	}
	for _, tc := range cases {
		derived := derivedParserVocabulary(t, "conformance.go", tc.function)
		want := map[string]bool{}
		for _, member := range tc.want {
			want[member] = true
		}
		for _, member := range derived {
			if !want[member] {
				t.Errorf("%s derives admitted %q, outside the pinned set; "+
					"a widened vocabulary ships unwitnessed", tc.function, member)
			}
		}
		derivedSet := map[string]bool{}
		for _, member := range derived {
			derivedSet[member] = true
		}
		for _, member := range tc.want {
			if !derivedSet[member] {
				t.Errorf("%s no longer derives admitted %q; "+
					"a narrowed vocabulary breaks the contract", tc.function, member)
			}
		}
		if len(derived) != len(tc.want) {
			t.Errorf("%s derives %d admitted members, want exactly %d; "+
				"the vocabulary is not the pinned set", tc.function, len(derived), len(tc.want))
		}
	}

	derivedOperations := derivedParserVocabulary(t, "conformance.go", "ParseOperation")
	if len(conformanceOperations) != len(derivedOperations) {
		t.Fatalf("conformanceOperations holds %d operations, production derives %d; "+
			"the error mapping is not defined over the admitted vocabulary",
			len(conformanceOperations), len(derivedOperations))
	}
	derivedSet := map[string]bool{}
	for _, operation := range derivedOperations {
		derivedSet[operation] = true
	}
	for _, operation := range conformanceOperations {
		if !derivedSet[operation] {
			t.Errorf("conformanceOperations holds %q, outside the derived operation vocabulary; "+
				"the error mapping iterates a member production refuses", operation)
		}
	}
}
