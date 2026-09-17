package sessstate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// sessstateRefusalRows rosters every derived refuse site with the
// boundary-driven test that exercises it. The derivation
// (deriveSessstateRefusalSites) and this roster must match in both
// directions: a site without a row, or a row without a site, fails.
// Subtests name the vector after the slash; Go records the slash form
// with spaces rendered as underscores.
func sessstateRefusalRows() map[string]string {
	return map[string]string{
		"decode.go:28":     "TestDecodeRecordRefusesMemberVectors/frame",
		"decode.go:32":     "TestDecodeRecordRefusesMemberVectors/schema",
		"decode.go:36":     "TestDecodeRecordRefusesMemberVectors/session_id",
		"decode.go:40":     "TestDecodeRecordRefusesMemberVectors/record_id_absent",
		"decode.go:43":     "TestDecodeRecordRefusesMemberVectors/record_id_grammar",
		"decode.go:81":     "TestDecodeEventRefusesMemberVectors/frame",
		"decode.go:85":     "TestDecodeEventRefusesMemberVectors/schema",
		"decode.go:89":     "TestDecodeEventRefusesMemberVectors/schema_version",
		"decode.go:93":     "TestDecodeEventRefusesMemberVectors/session_id",
		"decode.go:97":     "TestDecodeEventRefusesMemberVectors/event_type",
		"decode.go:101":    "TestDecodeEventRefusesMemberVectors/lease_epoch",
		"decode.go:105":    "TestDecodeEventRefusesMemberVectors/lease_id_absent",
		"decode.go:108":    "TestDecodeEventRefusesMemberVectors/lease_id_grammar",
		"decode.go:112":    "TestDecodeEventRefusesMemberVectors/lease_sequence",
		"decode.go:116":    "TestDecodeEventRefusesMemberVectors/created_by_host_id",
		"decode.go:120":    "TestDecodeEventRefusesMemberVectors/predecessors_malformed",
		"decode.go:123":    "TestDecodeEventRefusesMemberVectors/predecessors_empty",
		"decode.go:127":    "TestDecodeEventRefusesMemberVectors/event_id_absent",
		"decode.go:130":    "TestDecodeEventRefusesMemberVectors/event_id_grammar",
		"decode.go:212":    "TestReduceRefusesDerivationVectors/string_absent",
		"decode.go:216":    "TestReduceRefusesDerivationVectors/string_not_string",
		"decode.go:225":    "TestReduceRefusesDerivationVectors/bool_absent",
		"decode.go:229":    "TestReduceRefusesDerivationVectors/bool_not_bool",
		"decode.go:240":    "TestReduceRefusesDerivationVectors/uint_absent",
		"decode.go:244":    "TestReduceRefusesDerivationVectors/uint_not_uint",
		"decode.go:255":    "TestReduceRefusesDerivationVectors/nullable_absent",
		"decode.go:262":    "TestReduceRefusesDerivationVectors/nullable_not_string",
		"decode.go:265":    "TestReduceRefusesDerivationVectors/nullable_malformed",
		"project.go:38":    "TestProjectRefusesWithoutRepository",
		"project.go:52":    "TestProjectRefusesUnknownSession",
		"sessstate.go:321": "TestReduceRefusesContinuityVectors/empty_record_identity",
		"sessstate.go:375": "TestReduceRefusesContinuityVectors/unregistered_version",
		"sessstate.go:378": "TestReduceRefusesCrossSessionEvent",
		"sessstate.go:423": "TestReduceRefusesContinuityVectors/zero_sequence",
		"sessstate.go:426": "TestReduceRefusesContinuityVectors/no_predecessors",
		"sessstate.go:430": "TestReduceRefusesContinuityVectors/first_links_extras",
		"sessstate.go:433": "TestReduceRefusesContinuityVectors/first_opens_past_one",
		"sessstate.go:442": "TestReduceRefusesDuplicatedEvent",
		"sessstate.go:444": "TestReduceRefusesContinuityVectors/gap",
		"sessstate.go:448": "TestReduceRefusesContinuityVectors/successor_restarts_past_one",
		"sessstate.go:452": "TestReduceRefusesContinuityVectors/omits_prior_head",
		"sessstate.go:489": "TestReduceRefusesDivergentEnvelope",
		"sessstate.go:502": "TestReduceRefusesStaleEnvelope",
		"sessstate.go:528": "TestReduceRefusesContinuityVectors/stopped_opens_chain",
		"sessstate.go:534": "TestReduceRefusesUnlistedTransitions/running-to-stopped",
		"sessstate.go:601": "TestReduceGatesFailedToCreatingRetry/new_lease_refused",
		"sessstate.go:604": "TestReduceGatesFailedToCreatingRetry/checkpoint_refused",
		"sessstate.go:607": "TestReduceGatesFailedToCreatingRetry/retry_past_abort_refused",
		"sessstate.go:621": "TestReduceRefusesDerivationVectors/v4_binding_malformed",
		"sessstate.go:628": "TestReduceRefusesDerivationVectors/v4_backend_refused",
		"sessstate.go:637": "TestReduceRefusesDerivationVectors/v1_backend_unknown",
		"sessstate.go:683": "TestReduceRefusesDerivationVectors/identity_digest_malformed",
		"sessstate.go:706": "TestReduceRefusesDerivationVectors/checkpoint_digest_malformed",
		"sessstate.go:716": "TestReduceRefusesContinuityVectors/checkpoint_opens_chain",
		"sessstate.go:749": "TestReduceRefusesStoppedWithNullCheckpoint",
		"sessstate.go:755": "TestReduceRefusesDerivationVectors/abort_closure_with_checkpoint",
		"sessstate.go:758": "TestReduceRefusesBootstrapAbortStopWithoutAbort",
		"sessstate.go:771": "TestReduceRefusesDerivationVectors/resume_checkpoint_malformed",
		"sessstate.go:780": "TestReduceRefusesDerivationVectors/resume_v4_binding_malformed",
		"sessstate.go:787": "TestReduceRefusesDerivationVectors/resume_v4_backend_refused",
		"sessstate.go:796": "TestReduceRefusesDerivationVectors/resume_v1_backend_unknown",
		"sessstate.go:814": "TestReduceGatesFailedToCreatingRetry/abort_refused",
		"sessstate.go:834": "TestReduceRefusesLeaseLinkageMismatch/wrong_predecessor",
		"sessstate.go:841": "TestReduceRefusesLeaseLinkageMismatch/transfer_outside_envelope",
		"sessstate.go:849": "TestReduceRefusesLeaseLinkageMismatch/envelope_outside_payload",
		"sessstate.go:865": "TestReduceRefusesTaskBoardRepeatMismatch/foreign_provider",
		"sessstate.go:876": "TestReduceRefusesTaskBoardRepeatMismatch/foreign_creation_lease",
		"sessstate.go:924": "TestReduceRefusesUnionVectors/empty_lease_head",
		"sessstate.go:927": "TestReduceRefusesUnionVectors/malformed_lease_id",
	}
}

// TestRefusalSitesAreCensused requires the derived refusal inventory
// to equal the roster in both directions.
func TestRefusalSitesAreCensused(t *testing.T) {
	files, fileSet := invcore.MustScanProduction(t, ".")
	_, derived, failures := deriveSessstateRefusalSites(files, fileSet)
	for _, failure := range failures {
		t.Error(failure)
	}
	rows := map[string]struct{}{}
	for site := range sessstateRefusalRows() {
		rows[site] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "sessstate refusals", derived, rows)
}

// stateRows rosters every SessionState spelling with the test driving
// it through the production Reduce entry.
func stateRows() map[string]string {
	return map[string]string{
		"creating":      "TestReduceDerivesEverySessionState/creating",
		"running":       "TestReduceDerivesEverySessionState/running",
		"idle":          "TestReduceDerivesEverySessionState/idle",
		"quiescing":     "TestReduceDerivesEverySessionState/quiescing",
		"checkpointing": "TestReduceDerivesEverySessionState/checkpointing",
		"stopped":       "TestReduceDerivesEverySessionState/stopped",
		"materializing": "TestReduceDerivesEverySessionState/materializing",
		"parked":        "TestReduceDerivesEverySessionState/parked",
		"failed":        "TestReduceDerivesEverySessionState/failed",
		"stale":         "TestReduceDerivesEverySessionState/stale",
		"tombstoned":    "TestReduceDerivesEverySessionState/tombstoned",
	}
}

// deriveStateSpellings derives the SessionState spellings
// structurally: every State-typed const value, plus State()
// conversions over string literals. A State() conversion over anything
// else — a variable, a call — fails closed as unclassifiable, since a
// state value with no listed spelling is exactly the defect. It also
// returns the declared State const names for the use-position check.
func deriveStateSpellings(files []invcore.ProductionFile, fileSet *token.FileSet) (map[string]struct{}, map[string]struct{}, []string) {
	spelled := map[string]struct{}{}
	constNames := map[string]struct{}{}
	var failures []string
	_ = fileSet
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			spec, ok := node.(*ast.ValueSpec)
			if ok {
				typeIdent, ok := spec.Type.(*ast.Ident)
				if ok && typeIdent.Name == "State" {
					for index, name := range spec.Names {
						constNames[name.Name] = struct{}{}
						if index >= len(spec.Values) {
							failures = append(failures, production.Name+": State const "+name.Name+" without a value fails closed")
							continue
						}
						literal, ok := spec.Values[index].(*ast.BasicLit)
						if !ok || literal.Kind != token.STRING {
							failures = append(failures, production.Name+": State const "+name.Name+" without a string literal fails closed")
							continue
						}
						value, err := strconv.Unquote(literal.Value)
						if err != nil {
							failures = append(failures, production.Name+": State const "+name.Name+" holds an unreadable literal")
							continue
						}
						spelled[value] = struct{}{}
					}
				}
				return true
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			conversion, ok := call.Fun.(*ast.Ident)
			if !ok || conversion.Name != "State" || len(call.Args) != 1 {
				return true
			}
			switch argument := call.Args[0].(type) {
			case *ast.BasicLit:
				if argument.Kind != token.STRING {
					failures = append(failures, production.Name+": State() conversion over a non-string fails closed")
					return true
				}
				value, err := strconv.Unquote(argument.Value)
				if err != nil {
					failures = append(failures, production.Name+": State() conversion holds an unreadable literal")
					return true
				}
				spelled[value] = struct{}{}
			case *ast.Ident:
				// A use position, not a definition: the name must be a
				// declared State const, checked after the walk.
				spelled["use:"+argument.Name] = struct{}{}
			default:
				failures = append(failures, production.Name+": State() conversion over a computed value fails closed as unclassifiable")
			}
			return true
		})
	}
	for key := range spelled {
		if !strings.HasPrefix(key, "use:") {
			continue
		}
		delete(spelled, key)
		if _, ok := constNames[strings.TrimPrefix(key, "use:")]; !ok {
			failures = append(failures, "State() conversion over undeclared name "+strings.TrimPrefix(key, "use:")+" fails closed")
		}
	}
	return spelled, constNames, failures
}

// TestStateSpaceIsCensused requires the derived state spellings to
// equal the roster in both directions, every roster spelling to live
// exactly once as a production literal, and every roster value to
// derive from a production Reduce path in this test.
func TestStateSpaceIsCensused(t *testing.T) {
	files, fileSet := invcore.MustScanProduction(t, ".")
	derived, _, failures := deriveStateSpellings(files, fileSet)
	for _, failure := range failures {
		t.Error(failure)
	}
	rows := map[string]struct{}{}
	for spelling := range stateRows() {
		rows[spelling] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "sessstate states", derived, rows)
	assertStateLiteralsAreSingleSourced(t, files)

	record := buildRecord(t, nil)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	created := eventSpec{typ: "session.created", payload: createdPayload(decoded.RecordID)}
	launched := eventSpec{typ: "provider.launched", payload: launchedPayload("codex")}
	idle := eventSpec{typ: "session.idle", payload: idlePayload()}
	checkpoint := eventSpec{typ: "checkpoint.created", payload: checkpointPayload()}
	stopped := eventSpec{typ: "session.stopped", payload: stoppedPayload()}
	transferred := eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)}
	chains := map[string][]eventSpec{
		"creating":      {created},
		"running":       {created, launched},
		"idle":          {created, launched, idle},
		"quiescing":     {created, launched, idle, {typ: "session.quiescing", payload: quiescingPayload()}},
		"checkpointing": {created, launched, idle, checkpoint},
		"stopped":       {created, launched, idle, checkpoint, checkpoint, stopped},
		"materializing": {created, launched, idle, checkpoint, checkpoint, stopped, transferred},
		"parked": {created, launched, idle, checkpoint, checkpoint, stopped, transferred,
			{epoch: 2, leaseID: testLeaseIDB, typ: "session.parked", payload: parkedPayload(testLeaseIDB)}},
		"failed": {created, launched, {typ: "session.failed", payload: failedPayload()}},
		"stale": {created, launched, idle,
			{epoch: 2, leaseID: testLeaseIDB, typ: "lease.forced", payload: forcedPayload(testLeaseIDB, 1)}},
		"tombstoned": {created, launched, idle, checkpoint, checkpoint, stopped,
			{typ: "session.tombstoned", payload: tombstonedPayload()}},
	}
	produced := map[string]struct{}{}
	for want, chain := range chains {
		projection := reduceDirect(t, record, chain)
		if string(projection.State) != want {
			t.Errorf("census chain state = %q, want %q", projection.State, want)
		}
		produced[string(projection.State)] = struct{}{}
	}
	// A path producing an uncensused value fails here: the produced
	// set must equal the roster exactly, in both directions.
	invcore.MustCheckBothDirections(t, "sessstate produced states", produced, rows)
	if ordered := States(); len(ordered) != len(rows) {
		t.Fatalf("States() has %d entries, roster has %d", len(ordered), len(rows))
	} else {
		for index, spelling := range ordered {
			if _, ok := rows[spelling]; !ok {
				t.Fatalf("States()[%d] = %q outside the roster", index, spelling)
			}
		}
		// Declared order pins the Section 5.7 sequence exactly.
		want := []string{"creating", "running", "idle", "quiescing", "checkpointing", "stopped", "materializing", "parked", "failed", "stale", "tombstoned"}
		for index := range want {
			if ordered[index] != want[index] {
				t.Fatalf("States() order = %v, want Section 5.7 order %v", ordered, want)
			}
		}
	}
}

// assertStateLiteralsAreSingleSourced requires every roster spelling to
// appear exactly once as a string literal in production: the const
// block is the single source, and a second literal (a message, a
// duplicate table) fails the gate.
func assertStateLiteralsAreSingleSourced(t *testing.T, files []invcore.ProductionFile) {
	t.Helper()
	counts := map[string]int{}
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			counts[value]++
			return true
		})
	}
	for spelling := range stateRows() {
		if counts[spelling] != 1 {
			t.Errorf("state spelling %q appears %d times as a production literal, want exactly once", spelling, counts[spelling])
		}
	}
}

// eventRow names one handled event type with its class and driver.
type eventRow struct {
	class  string
	driver string
}

// eventRows rosters every handled v1 event type: lifecycle types move
// the derivation, fact types record data and keep the state. Unknown
// v1 types stay inert by default and never appear here.
func eventRows() map[string]eventRow {
	return map[string]eventRow{
		"session.created":           {"lifecycle", "TestReduceDerivesEverySessionState/creating"},
		"terminal.created":          {"lifecycle", "TestProjectDerivesFullLifecycle"},
		"provider.launched":         {"lifecycle", "TestReduceDerivesEverySessionState/running"},
		"provider.identified":       {"fact", "TestProjectDerivesFullLifecycle"},
		"session.idle":              {"lifecycle", "TestReduceDerivesEverySessionState/idle"},
		"session.quiescing":         {"lifecycle", "TestReduceDerivesEverySessionState/quiescing"},
		"checkpoint.created":        {"lifecycle", "TestReduceDerivesEverySessionState/checkpointing"},
		"sync.completed":            {"fact", "TestReduceKeepsFactEventsStateless"},
		"session.stopped":           {"lifecycle", "TestReduceDerivesEverySessionState/stopped"},
		"session.resumed":           {"lifecycle", "TestProjectDerivesFullLifecycle"},
		"session.bootstrap_aborted": {"lifecycle", "TestReduceDerivesFailedFromBootstrapAbortStop"},
		"lease.transferred":         {"lifecycle", "TestReduceDerivesEverySessionState/materializing"},
		"lease.forced":              {"lifecycle", "TestReduceDerivesEverySessionState/stale"},
		"session.parked":            {"lifecycle", "TestReduceDerivesEverySessionState/parked"},
		"session.failed":            {"lifecycle", "TestReduceDerivesEverySessionState/failed"},
		"fork.created":              {"fact", "TestReduceKeepsFactEventsStateless"},
		"profile.changed":           {"fact", "TestReduceKeepsFactEventsStateless"},
		"session.tombstoned":        {"lifecycle", "TestReduceDerivesEverySessionState/tombstoned"},
		"takeover.force_confirmed":  {"fact", "TestReduceKeepsFactEventsStateless"},
		"replica.replace_confirmed": {"fact", "TestReduceKeepsFactEventsStateless"},
		"task_board.launched":       {"lifecycle", "TestReduceDerivesTaskBoardLaunch"},
		"task_board.adopted":        {"fact", "TestReduceKeepsFactEventsStateless"},
		"tombstone.issued":          {"fact", "TestReduceKeepsFactEventsStateless"},
		"tombstone.resolved":        {"fact", "TestReduceKeepsFactEventsStateless"},
	}
}

// pinnedV1EventTypes is the Section 5.2 v1 registry in table order.
// The canonical owner pins the same registry against the catalog; this
// list pins the reducer's handling against the section.
func pinnedV1EventTypes() []string {
	return []string{
		"session.created", "terminal.created", "provider.launched",
		"provider.identified", "session.idle", "session.quiescing",
		"checkpoint.created", "sync.completed", "session.stopped",
		"session.resumed", "session.bootstrap_aborted",
		"lease.transferred", "lease.forced", "session.parked",
		"session.failed", "fork.created", "profile.changed",
		"session.tombstoned", "takeover.force_confirmed",
		"replica.replace_confirmed", "task_board.launched",
		"task_board.adopted", "tombstone.issued", "tombstone.resolved",
	}
}

// deriveHandledEventTypes derives the handled event_type literals
// from direct-selector dispatch only: the case literals of
// every switch over a `<value>.Type` selector, plus the string
// literals compared against a `<value>.Type` selector with == or !=.
// A non-literal case fails closed, and a switch dispatch site outside
// `effect` fails closed: a handling site the census cannot see is
// exactly the defect this denominator must not admit. This derivation alone
// is insufficient: TestEventTypeUsesAreOwned is the ownership prerequisite
// that refuses copies and helper escapes before downstream classification.
func deriveHandledEventTypes(files []invcore.ProductionFile, fileSet *token.FileSet) (map[string]struct{}, []string) {
	_ = fileSet
	derived := map[string]struct{}{}
	var failures []string
	isTypeSelector := func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		return ok && selector.Sel.Name == "Type"
	}
	collect := func(production invcore.ProductionFile, expression ast.Node) {
		literal, ok := expression.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			failures = append(failures, production.Name+": event-type case without a string literal fails closed")
			return
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			failures = append(failures, production.Name+": event-type case holds an unreadable literal")
			return
		}
		derived[value] = struct{}{}
	}
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			function, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}
			ast.Inspect(function.Body, func(inner ast.Node) bool {
				switch statement := inner.(type) {
				case *ast.SwitchStmt:
					if !isTypeSelector(statement.Tag) {
						return true
					}
					if function.Name.Name != "effect" {
						failures = append(failures, production.Name+": event-type dispatch outside effect in "+function.Name.Name+" fails closed")
					}
					for _, clause := range statement.Body.List {
						caseClause, ok := clause.(*ast.CaseClause)
						if !ok {
							continue
						}
						for _, expression := range caseClause.List {
							collect(production, expression)
						}
					}
				case *ast.BinaryExpr:
					if statement.Op != token.EQL && statement.Op != token.NEQ {
						return true
					}
					if isTypeSelector(statement.X) {
						collect(production, statement.Y)
					} else if isTypeSelector(statement.Y) {
						collect(production, statement.X)
					}
				}
				return true
			})
			return false
		})
	}
	return derived, failures
}

// TestEventHandlingIsCensused requires the derived handled set to
// equal the roster in both directions, and the roster to equal the
// pinned v1 registry exactly: a handled type without a row, a row
// without a handler, and a registry type with no handling decision
// each fail.
func TestEventHandlingIsCensused(t *testing.T) {
	files, fileSet := invcore.MustScanProduction(t, ".")
	derived, failures := deriveHandledEventTypes(files, fileSet)
	for _, failure := range failures {
		t.Error(failure)
	}
	rows := map[string]struct{}{}
	for eventType, row := range eventRows() {
		if row.class != "lifecycle" && row.class != "fact" {
			t.Errorf("event row %s carries class %q, want lifecycle or fact", eventType, row.class)
		}
		if row.driver == "" {
			t.Errorf("event row %s names no driver", eventType)
		}
		rows[eventType] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "sessstate handled events", derived, rows)
	pinned := map[string]struct{}{}
	for _, eventType := range pinnedV1EventTypes() {
		pinned[eventType] = struct{}{}
	}
	invcore.MustCheckBothDirections(t, "sessstate v1 registry", pinned, rows)
	var ordered []string
	for eventType := range rows {
		ordered = append(ordered, eventType)
	}
	sort.Strings(ordered)
	if len(ordered) != 24 {
		t.Fatalf("handled events = %d, want the 24-member v1 registry", len(ordered))
	}
	_ = strings.Join(ordered, ",")
}

// eventTypeUseRows is an ownership ledger, not a dataflow guess. Every
// production selector named Type must be one of these exact AST uses.
// Copies, helper arguments, address-taking, writes, closures and new owner
// sites fail closed before their downstream dispatch needs classification.
// This intentionally also rejects unrelated Type selectors for review.
// Bounds: this is a direct-field-access contract, not whole-program taint
// analysis. Reflection/unsafe/serialization of whole Event values and later
// interpretation of diagnostic strings are not proved by this instrument.
// The exact diagnostic statements below are permitted outputs, not trusted
// authority inputs. The census does not claim to prove their downstream use.
func eventTypeUseRows() map[string]struct{} {
	return map[string]struct{}{
		`sessstate.go|what|return "event " + event.Type + " " + event.ID`:              {},
		`sessstate.go|chainFold.effect|name := "event " + event.Type + " " + event.ID`: {},
		`sessstate.go|chainFold.effect|switch event.Type`:                              {},
		`sessstate.go|chainFold.effectLease|if event.Type == "lease.transferred"`:      {},
	}
}

func deriveEventTypeUses(files []invcore.ProductionFile, fileSet *token.FileSet) (map[string]struct{}, []string) {
	derived := map[string]struct{}{}
	var failures []string
	for _, production := range files {
		var stack []ast.Node
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, node)
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Type" {
				return true
			}
			owner := "<package>"
			var context ast.Node
			prefix := ""
			for i := len(stack) - 2; i >= 0; i-- {
				switch parent := stack[i].(type) {
				case *ast.FuncDecl:
					owner = parent.Name.Name
					if parent.Recv != nil {
						var receiver bytes.Buffer
						if err := format.Node(&receiver, fileSet, parent.Recv.List[0].Type); err != nil {
							failures = append(failures, err.Error())
						}
						owner = strings.TrimPrefix(receiver.String(), "*") + "." + owner
					}
				case *ast.FuncLit:
					// A closure has a distinct ownership boundary, even
					// when nested inside a registered function.
					prefix += "closure "
				}
				if context == nil {
					switch parent := stack[i].(type) {
					case *ast.SwitchStmt:
						if parent.Tag == selector {
							context, prefix = parent.Tag, "switch "
						} else {
							context = parent
						}
					case *ast.IfStmt:
						context, prefix = parent.Cond, "if "
					case ast.Stmt:
						context = parent
					case *ast.ValueSpec:
						context = parent
					}
				}
			}
			if context == nil {
				failures = append(failures, production.Name+": unclassifiable Type use")
				return true
			}
			var rendered bytes.Buffer
			if err := format.Node(&rendered, fileSet, context); err != nil {
				failures = append(failures, err.Error())
				return true
			}
			key := production.Name + "|" + owner + "|" + prefix + rendered.String()
			if _, duplicate := derived[key]; duplicate {
				failures = append(failures, "duplicate event-type owner use: "+key)
			}
			derived[key] = struct{}{}
			return true
		})
	}
	return derived, failures
}

func eventTypeOwnershipFailures(files []invcore.ProductionFile, fileSet *token.FileSet, rows map[string]struct{}) []string {
	derived, failures := deriveEventTypeUses(files, fileSet)
	unregistered, orphaned, diffFailures := invcore.DiffSets(derived, rows)
	failures = append(failures, diffFailures...)
	for _, site := range unregistered {
		failures = append(failures, "unregistered event-type use (alias/escape or unclassified owner): "+site)
	}
	for _, site := range orphaned {
		failures = append(failures, "orphan event-type owner row: "+site)
	}
	return failures
}

func TestEventTypeUsesAreOwned(t *testing.T) {
	files, fileSet := invcore.MustScanProduction(t, ".")
	for _, failure := range eventTypeOwnershipFailures(files, fileSet, eventTypeUseRows()) {
		t.Error(failure)
	}
	derived, _ := deriveEventTypeUses(files, fileSet)
	t.Logf("event-type uses: %d production sites / %d owner rows", len(derived), len(eventTypeUseRows()))
}

// Named static audit entry makes audit-only classification independently
// executable with no behavioral or census test selected.
func TestSessstateStaticAudit(t *testing.T) {
	files, fileSet := invcore.MustScanProduction(t, ".")
	failures := append(auditSessstateAliases(files, fileSet), auditSessstateUnrouted(files, fileSet)...)
	for _, failure := range failures {
		t.Error(fmt.Sprint(failure))
	}
}
