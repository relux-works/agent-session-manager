package canonicaljson

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// This file discharges the obligations derived in coupling_derivation_test.go.
// Every case drives a real production entry — CalculateObjectIdentity and
// VerifyObjectIdentity for identity-addressed records, ValidateObservationEvent
// or ValidateObservationStream for the Observation Event, which Section 10.1
// declares is not identity-addressed.

// ---------------------------------------------------------------------------
// Candidates
// ---------------------------------------------------------------------------

// couplingCandidate is one candidate driven at one production entry.
type couplingCandidate struct {
	selfField   SelfField
	object      map[string]any
	stream      []map[string]any
	isStream    bool
	observation bool
}

func identityCandidate(selfField SelfField, object map[string]any) couplingCandidate {
	return couplingCandidate{selfField: selfField, object: object}
}

func observationCandidate(object map[string]any) couplingCandidate {
	return couplingCandidate{object: object, observation: true}
}

func streamCandidate(objects ...map[string]any) couplingCandidate {
	return couplingCandidate{stream: objects, isStream: true}
}

func (candidate couplingCandidate) assertRefuses(t *testing.T, want string) {
	t.Helper()
	switch {
	case candidate.isStream:
		encoded := make([][]byte, 0, len(candidate.stream))
		for _, object := range candidate.stream {
			encoded = append(encoded, mustJSON(t, object))
		}
		err := ValidateObservationStream(encoded)
		if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateObservationStream error = %v, want observation refusal containing %q", err, want)
		}
	case candidate.observation:
		err := ValidateObservationEvent(mustJSON(t, candidate.object))
		if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateObservationEvent error = %v, want observation refusal containing %q", err, want)
		}
	default:
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, candidate.object), candidate.selfField, want)
	}
}

func (candidate couplingCandidate) assertAccepts(t *testing.T) {
	t.Helper()
	switch {
	case candidate.isStream:
		encoded := make([][]byte, 0, len(candidate.stream))
		for _, object := range candidate.stream {
			encoded = append(encoded, mustJSON(t, object))
		}
		if err := ValidateObservationStream(encoded); err != nil {
			t.Fatalf("ValidateObservationStream error = %v, want acceptance", err)
		}
	case candidate.observation:
		if err := ValidateObservationEvent(mustJSON(t, candidate.object)); err != nil {
			t.Fatalf("ValidateObservationEvent error = %v, want acceptance", err)
		}
	default:
		assertIdentityEntriesAcceptShape(t, mustJSON(t, candidate.object), candidate.selfField)
	}
}

// ---------------------------------------------------------------------------
// Presence-coupling coverage
// ---------------------------------------------------------------------------

// presenceCouplingProof discharges one single-sided violation of one derived
// coupling. Each build must produce a candidate that violates the coupling in
// that direction and in no other declared way, so the case pins its own clause
// rather than refusing on an earlier disjunct.
type presenceCouplingProof struct {
	key       string
	direction presenceDirection
	// spec quotes the pinned declaration the coupling implements, or discloses
	// that the rule is implementation-defined.
	spec    string
	refusal string
	builds  func(t *testing.T) []couplingCandidate
}

// TestEveryPresenceCouplingIsProvenInBothSingleDirections is the coverage
// assertion for the class. It fails when a derived coupling has no proof for one
// of its two single-sided violations, and when a proof claims a coupling that no
// production site declares.
//
// The failure it prevents, reproduced by review on this leaf: narrowing
// `requiresError != errorPresent` to `requiresError && !errorPresent` left the
// entire configured gate set green while ValidateObservationEvent attested a
// success-result Observation Event carrying a non-null error_code.
func TestEveryPresenceCouplingIsProvenInBothSingleDirections(t *testing.T) {
	t.Parallel()

	obligations := derivePresenceCouplingObligations(t)
	proven := make(map[presenceObligation]int)
	for _, proof := range presenceCouplingProofs() {
		proven[presenceObligation{key: proof.key, direction: proof.direction}]++
	}

	var missing, extra []string
	for obligation, required := range obligations {
		if proven[obligation] < required {
			missing = append(missing, fmt.Sprintf("%s: %d site(s), %d proof(s)", obligation, required, proven[obligation]))
		}
	}
	for obligation, count := range proven {
		if required := obligations[obligation]; count > required {
			extra = append(extra, fmt.Sprintf("%s: %d proof(s), %d site(s)", obligation, count, required))
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("presence couplings with no single-direction proof:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("presence-coupling proofs claiming a site that does not exist:\n  %s", strings.Join(extra, "\n  "))
	}
}

// TestEveryPresenceCouplingSingleDirectionViolationIsRefused runs the table.
func TestEveryPresenceCouplingSingleDirectionViolationIsRefused(t *testing.T) {
	for _, proof := range presenceCouplingProofs() {
		t.Run(proof.key+" ["+string(proof.direction)+"]", func(t *testing.T) {
			candidates := proof.builds(t)
			if len(candidates) == 0 {
				t.Fatalf("%s [%s] built zero candidates", proof.key, proof.direction)
			}
			for _, candidate := range candidates {
				candidate.assertRefuses(t, proof.refusal)
			}
		})
	}
}

func presenceCouplingProofs() []presenceCouplingProof {
	const (
		gitFeaturesRefusal  = "GitFeatures sparse pattern IDs must both be non-null exactly when sparse_checkout is true"
		bootstrapRefusal    = "session.bootstrap_aborted after identity requires exactly one identity field"
		checkpointRefusal   = "Checkpoint Record requires exactly one of provider_manifest_id and task_board_bundle_id"
		goalRefusal         = "task-board goal ID and revision must both be null or both non-null"
		observationRefusal  = "Observation Event partial/failure requires non-null error_code and every other result requires null"
		tombstoneRefusal    = "tombstone.resolved resulting_entry_digest must be non-null exactly for resurrected"
		sparsePatternMember = "sparse_patterns_blob_id"
	)
	single := func(build func(t *testing.T) couplingCandidate) func(*testing.T) []couplingCandidate {
		return func(t *testing.T) []couplingCandidate {
			return []couplingCandidate{build(t)}
		}
	}
	gitFeatures := func(sparse, first, second bool) func(t *testing.T) couplingCandidate {
		return func(t *testing.T) couplingCandidate {
			object := validGitWorkspaceGroupObject()
			features := gitWorkspaceMember(object)["features"].(map[string]any)
			features["sparse_checkout"] = sparse
			features[sparsePatternMember] = nullableDigest(first, '8')
			features["sparse_patterns_blob_descriptor_id"] = nullableDigest(second, '9')
			return identityCandidate(SelfManifestID, object)
		}
	}

	return []presenceCouplingProof{
		{
			// Isolation: the sibling clause `sparse != (firstPresent &&
			// secondPresent)` is held false by keeping sparse equal to
			// firstPresent && secondPresent, so only this clause can fire.
			key:       "closed_shapes.go|validateGitFeatures|firstPresent != secondPresent",
			direction: presenceLeftOnly,
			spec:      "GitFeatures: sparse_patterns_blob_id and sparse_patterns_blob_descriptor_id are both non-null exactly when sparse_checkout is true",
			refusal:   gitFeaturesRefusal,
			builds:    single(gitFeatures(false, true, false)),
		},
		{
			key:       "closed_shapes.go|validateGitFeatures|firstPresent != secondPresent",
			direction: presenceRightOnly,
			spec:      "GitFeatures: sparse_patterns_blob_id and sparse_patterns_blob_descriptor_id are both non-null exactly when sparse_checkout is true",
			refusal:   gitFeaturesRefusal,
			builds:    single(gitFeatures(false, false, true)),
		},
		{
			// Isolation: the sibling clause `firstPresent != secondPresent` is
			// held false by giving the two pattern members the same presence.
			key:       "closed_shapes.go|validateGitFeatures|sparse != (firstPresent && secondPresent)",
			direction: presenceLeftOnly,
			spec:      "GitFeatures: sparse_checkout true requires both sparse pattern IDs",
			refusal:   gitFeaturesRefusal,
			builds:    single(gitFeatures(true, false, false)),
		},
		{
			key:       "closed_shapes.go|validateGitFeatures|sparse != (firstPresent && secondPresent)",
			direction: presenceRightOnly,
			spec:      "GitFeatures: sparse_checkout false forbids both sparse pattern IDs",
			refusal:   gitFeaturesRefusal,
			builds:    single(gitFeatures(false, true, true)),
		},
		{
			// literalBool is the shared helper behind every declared literal
			// boolean payload member. Its two directions are proven at the two
			// call sites that declare opposite literals, so neither
			// `value && !expected` nor `!value && expected` survives.
			key:       "core_records.go|literalBool|value != expected",
			direction: presenceLeftOnly,
			spec:      "session.bootstrap_aborted: resume_allowed is false",
			refusal:   "member resume_allowed must be false",
			builds: single(func(t *testing.T) couplingCandidate {
				object := bootstrapAbortedEvent(t)
				object["payload"].(map[string]any)["resume_allowed"] = true
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			key:       "core_records.go|literalBool|value != expected",
			direction: presenceRightOnly,
			spec:      "session.bootstrap_aborted: process_closed is true",
			refusal:   "member process_closed must be true",
			builds: single(func(t *testing.T) couplingCandidate {
				object := bootstrapAbortedEvent(t)
				object["payload"].(map[string]any)["process_closed"] = false
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			key:       "core_records.go|validateBootstrapAbortedPayload|providerPresent == managerPresent",
			direction: presenceBoth,
			spec:      "session.bootstrap_aborted: after identity is established exactly one identity field is non-null",
			refusal:   bootstrapRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := bootstrapAbortedEvent(t)
				payload := object["payload"].(map[string]any)
				payload["failure_phase"] = "after_identity"
				payload["provider_identity_record_id"] = zeroDigest
				payload["manager_session_ref"] = "manager-1"
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			key:       "core_records.go|validateBootstrapAbortedPayload|providerPresent == managerPresent",
			direction: presenceNeither,
			spec:      "session.bootstrap_aborted: after identity is established exactly one identity field is non-null",
			refusal:   bootstrapRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := bootstrapAbortedEvent(t)
				payload := object["payload"].(map[string]any)
				payload["failure_phase"] = "before_checkpoint"
				payload["provider_identity_record_id"] = nil
				payload["manager_session_ref"] = nil
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			key:       "core_records.go|validateCheckpointRecord|providerPresent == boardPresent",
			direction: presenceBoth,
			spec:      "Checkpoint Record: exactly one of provider_manifest_id and task_board_bundle_id is non-null",
			refusal:   checkpointRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := validCheckpointRecordObject(true)
				object["task_board_bundle_id"] = zeroDigest
				return identityCandidate(SelfCheckpointID, object)
			}),
		},
		{
			key:       "core_records.go|validateCheckpointRecord|providerPresent == boardPresent",
			direction: presenceNeither,
			spec:      "Checkpoint Record: exactly one of provider_manifest_id and task_board_bundle_id is non-null",
			refusal:   checkpointRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := validCheckpointRecordObject(true)
				object["provider_manifest_id"] = nil
				return identityCandidate(SelfCheckpointID, object)
			}),
		},
		{
			key:       "core_records.go|validateGoalPair|goalPresent != revisionPresent",
			direction: presenceLeftOnly,
			spec:      "task_board.launched: board_goal_id and board_goal_revision are both null or both non-null",
			refusal:   goalRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := taskBoardLaunchedEvent(t)
				object["payload"].(map[string]any)["board_goal_revision"] = nil
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			// The reverse half: a revision with no goal ID. Without this case
			// `goalPresent != revisionPresent` narrowed to
			// `goalPresent && !revisionPresent` admits it.
			key:       "core_records.go|validateGoalPair|goalPresent != revisionPresent",
			direction: presenceRightOnly,
			spec:      "task_board.launched: board_goal_id and board_goal_revision are both null or both non-null",
			refusal:   goalRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := taskBoardLaunchedEvent(t)
				object["payload"].(map[string]any)["board_goal_id"] = nil
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			key:       "core_records.go|validateObservationEvent|requiresError != errorPresent",
			direction: presenceLeftOnly,
			spec:      "Observation Event: error_code is non-null exactly for a partial or failure result",
			refusal:   observationRefusal,
			builds:    observationResultErrorCandidates(true),
		},
		{
			// The half review found unproven. Every admitted result outside
			// {partial, failure} is driven carrying a non-null error_code, at
			// both the single-event and the stream entry, so neither
			// `requiresError && !errorPresent` nor a result-by-result narrowing
			// of the complement survives.
			key:       "core_records.go|validateObservationEvent|requiresError != errorPresent",
			direction: presenceRightOnly,
			spec:      "Observation Event: error_code is non-null exactly for a partial or failure result",
			refusal:   observationRefusal,
			builds:    observationResultErrorCandidates(false),
		},
		{
			key:       "core_records.go|validateTombstoneResolvedPayload|(resolution == \"resurrected\") != present",
			direction: presenceLeftOnly,
			spec:      "tombstone.resolved: resulting_entry_digest is non-null exactly for a resurrected resolution",
			refusal:   tombstoneRefusal,
			builds: single(func(t *testing.T) couplingCandidate {
				object := tombstoneResolvedEvent(t)
				payload := object["payload"].(map[string]any)
				payload["resolution"] = "resurrected"
				payload["resulting_entry_digest"] = nil
				return identityCandidate(SelfEventID, object)
			}),
		},
		{
			// The reverse half, driven for every admitted resolution outside
			// resurrected rather than for one hand-picked value.
			key:       "core_records.go|validateTombstoneResolvedPayload|(resolution == \"resurrected\") != present",
			direction: presenceRightOnly,
			spec:      "tombstone.resolved: resulting_entry_digest is non-null exactly for a resurrected resolution",
			refusal:   tombstoneRefusal,
			builds: func(t *testing.T) []couplingCandidate {
				var candidates []couplingCandidate
				for _, resolution := range admittedValues(t, "validateTombstoneResolvedPayload", "resolution") {
					if resolution == "resurrected" {
						continue
					}
					object := tombstoneResolvedEvent(t)
					payload := object["payload"].(map[string]any)
					payload["resolution"] = resolution
					payload["resulting_entry_digest"] = zeroDigest
					candidates = append(candidates, identityCandidate(SelfEventID, object))
				}
				return candidates
			},
		},
	}
}

// observationResultErrorCandidates derives the two sides of the Observation
// Event result/error coupling from the production vocabulary rather than from a
// hand-written list, so a sixth result value becomes a case without anyone
// editing this file. requiresError selects the {partial, failure} side.
func observationResultErrorCandidates(requiresError bool) func(t *testing.T) []couplingCandidate {
	return func(t *testing.T) []couplingCandidate {
		var candidates []couplingCandidate
		for _, result := range admittedValues(t, "validateObservationEvent", "result") {
			needsError := result == "partial" || result == "failure"
			if needsError != requiresError {
				continue
			}
			build := func() map[string]any {
				object := validObservationEventObject()
				object["result"] = result
				if requiresError {
					object["error_code"] = nil
				} else {
					object["error_code"] = "boom"
				}
				if result == "started" {
					// started declares a null duration; setting it keeps this
					// case violating only the result/error coupling.
					object["duration_ms"] = nil
				}
				return object
			}
			candidates = append(candidates,
				observationCandidate(build()),
				streamCandidate(build()),
			)
		}
		if len(candidates) == 0 {
			t.Fatalf("derived no admitted Observation Event result with requiresError=%t", requiresError)
		}
		return candidates
	}
}

// admittedValues resolves a closed vocabulary from the production sources
// through the same derivation the closed-vocabulary inventory uses, so a
// vocabulary that is widened or narrowed changes these cases too.
func admittedValues(t *testing.T, function, member string) []string {
	t.Helper()
	for _, site := range deriveClosedVocabularySites(t) {
		if site.function == function && site.member == member {
			return site.values
		}
	}
	t.Fatalf("no derived vocabulary for %s member %q", function, member)
	return nil
}

func nullableDigest(present bool, digit byte) any {
	if !present {
		return nil
	}
	return digestWithDigit(digit)
}

func bootstrapAbortedEvent(t *testing.T) map[string]any {
	t.Helper()
	return validSessionEventObject(lowestSessionEventVersion(t, "session.bootstrap_aborted"), "session.bootstrap_aborted")
}

func taskBoardLaunchedEvent(t *testing.T) map[string]any {
	t.Helper()
	return validSessionEventObject(lowestSessionEventVersion(t, "task_board.launched"), "task_board.launched")
}

func tombstoneResolvedEvent(t *testing.T) map[string]any {
	t.Helper()
	return validSessionEventObject(lowestSessionEventVersion(t, "tombstone.resolved"), "tombstone.resolved")
}

// ---------------------------------------------------------------------------
// Literal-boundary coverage
// ---------------------------------------------------------------------------

// literalBoundaryProof discharges one derived comparison at one of the values
// where it flips. refusal empty means the candidate must be accepted.
type literalBoundaryProof struct {
	key     string
	value   int
	spec    string
	refusal string
	build   func(t *testing.T, value int) couplingCandidate
}

// literalBoundarySubsumption records a boundary value that cannot be driven at
// a production entry because an earlier refusal always fires first. The
// subsuming refusal is named and the naming is pinned, so "subsumed" cannot
// wave a boundary through.
type literalBoundarySubsumption struct {
	subsumingRefusal string
	provingTest      string
}

// subsumedLiteralBoundaries is asserted exactly by the coverage test.
var subsumedLiteralBoundaries = map[literalBoundaryObligation]literalBoundarySubsumption{
	{key: "core_records.go|ValidateObservationStream|sequence != 1", value: 0}: {
		subsumingRefusal: "member sequence must be greater than zero",
		provingTest:      "TestObservationSequenceZeroIsRefusedBeforeTheStreamStartBoundary",
	},
	{key: "core_records.go|validateLeaseRecord|epoch != 1", value: 0}: {
		subsumingRefusal: "member epoch must be greater than zero",
		provingTest:      "TestEveryLiteralBoundaryIsDrivenAtTheValuesWhereItFlips",
	},
	{key: "core_records.go|validateLeaseRecord|epoch == 1", value: 0}: {
		subsumingRefusal: "member epoch must be greater than zero",
		provingTest:      "TestEveryLiteralBoundaryIsDrivenAtTheValuesWhereItFlips",
	},
}

// TestEveryLiteralBoundaryHasAProofAtEachFlipValue is the coverage assertion for
// the class.
//
// The failure it prevents, reproduced by review on this leaf: `epoch != 1` was
// proven only at epoch 4, so narrowing it to `epoch >= 3` left the gate set
// green while CalculateObjectIdentity attested an epoch-2 lease with a null
// predecessor. `epoch != 1` narrowed to `epoch > 1` is an equivalent mutant, so
// no deletion or operator-rewrite sweep could have surfaced it.
func TestEveryLiteralBoundaryHasAProofAtEachFlipValue(t *testing.T) {
	t.Parallel()

	obligations := deriveLiteralBoundaryObligations(t)
	proven := make(map[literalBoundaryObligation]int)
	for _, proof := range literalBoundaryProofs() {
		proven[literalBoundaryObligation{key: proof.key, value: proof.value}]++
	}
	for obligation := range subsumedLiteralBoundaries {
		proven[obligation]++
	}

	var missing, extra []string
	for obligation, required := range obligations {
		if proven[obligation] < required {
			missing = append(missing, fmt.Sprintf("%s: %d site(s), %d proof(s)", obligation, required, proven[obligation]))
		}
	}
	for obligation, count := range proven {
		if required := obligations[obligation]; count > required {
			extra = append(extra, fmt.Sprintf("%s: %d proof(s), %d site(s)", obligation, count, required))
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("literal boundaries with no proof at a flip value:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("literal-boundary proofs claiming a value no site declares:\n  %s", strings.Join(extra, "\n  "))
	}
}

// TestSubsumedLiteralBoundariesNameATestThatExists keeps the subsumption escape
// hatch honest.
func TestSubsumedLiteralBoundariesNameATestThatExists(t *testing.T) {
	t.Parallel()

	names := packageTestFunctionNames(t)
	for obligation, subsumption := range subsumedLiteralBoundaries {
		if _, ok := names[subsumption.provingTest]; !ok {
			t.Errorf("subsumed boundary %s names %s, which does not exist in this package", obligation, subsumption.provingTest)
		}
	}
}

// TestInitializationTimeComparisonExemptionsAreRealAndPanicking asserts the
// exemption set exactly against the production sources: an exempted function
// must exist, and it must be one that runs at package initialization rather than
// on a candidate.
func TestInitializationTimeComparisonExemptionsAreRealAndPanicking(t *testing.T) {
	t.Parallel()

	declared := make(map[string]struct{}, len(initializationTimeComparisonFunctions))
	for name := range initializationTimeComparisonFunctions {
		declared[name] = struct{}{}
	}
	found := make(map[string]struct{})
	forEachProductionComparison(t, func(context comparisonContext) {
		if context.file != literalBoundaryScope {
			return
		}
		if _, ok := declared[context.function]; ok {
			found[context.function] = struct{}{}
		}
	})
	for name := range declared {
		if _, ok := found[name]; !ok {
			t.Errorf("exempted function %s declares no comparison in %s; the exemption is obsolete", name, literalBoundaryScope)
		}
	}
	// The exemption only holds while every caller of an exempted function is a
	// package-level initializer that panics rather than returning a refusal to a
	// caller. Derived, so a new caller from a validator reddens this gate.
	for name := range declared {
		callers := derivePackageCallers(t, name)
		if len(callers) == 0 {
			t.Errorf("exempted function %s has no caller; the exemption is obsolete", name)
			continue
		}
		for _, caller := range callers {
			if !packageLevelInitializers(t)[caller] {
				t.Errorf("exempted function %s is called from %s, which is not a package-level initializer; "+
					"its comparisons now run on candidate input and must be proven", name, caller)
			}
		}
	}
}

// derivePackageCallers returns every production function that calls the named
// function.
func derivePackageCallers(t *testing.T, callee string) []string {
	t.Helper()

	_, paths := packageProductionFiles(t)
	seen := make(map[string]struct{})
	for _, path := range paths {
		file := parseProductionFile(t, path)
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil || function.Name.Name == callee {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == callee {
					seen[function.Name.Name] = struct{}{}
				}
				return true
			})
		}
	}
	callers := make([]string, 0, len(seen))
	for name := range seen {
		callers = append(callers, name)
	}
	sort.Strings(callers)
	return callers
}

// packageLevelInitializers returns the functions whose only role is to build a
// package-level variable: the function is called from a top-level `var`
// declaration and panics instead of returning a refusal.
func packageLevelInitializers(t *testing.T) map[string]bool {
	t.Helper()

	_, paths := packageProductionFiles(t)
	initializers := make(map[string]bool)
	panics := make(map[string]bool)
	for _, path := range paths {
		file := parseProductionFile(t, path)
		for _, declaration := range file.Decls {
			switch node := declaration.(type) {
			case *ast.GenDecl:
				if node.Tok != token.VAR {
					continue
				}
				for _, specification := range node.Specs {
					value, ok := specification.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, expression := range value.Values {
						call, ok := expression.(*ast.CallExpr)
						if !ok {
							continue
						}
						if identifier, ok := call.Fun.(*ast.Ident); ok {
							initializers[identifier.Name] = true
						}
					}
				}
			case *ast.FuncDecl:
				if node.Body == nil {
					continue
				}
				ast.Inspect(node.Body, func(inner ast.Node) bool {
					if _, ok := inner.(*ast.CallExpr); ok {
						if call := inner.(*ast.CallExpr); call != nil {
							if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "panic" {
								panics[node.Name.Name] = true
							}
						}
					}
					return true
				})
			}
		}
	}
	for name := range initializers {
		initializers[name] = panics[name]
	}
	return initializers
}

// TestEveryLiteralBoundaryIsDrivenAtTheValuesWhereItFlips runs the table.
func TestEveryLiteralBoundaryIsDrivenAtTheValuesWhereItFlips(t *testing.T) {
	for _, proof := range literalBoundaryProofs() {
		t.Run(fmt.Sprintf("%s at %d", proof.key, proof.value), func(t *testing.T) {
			candidate := proof.build(t, proof.value)
			if proof.refusal == "" {
				candidate.assertAccepts(t)
				return
			}
			candidate.assertRefuses(t, proof.refusal)
		})
	}
}

// TestObservationSequenceZeroIsRefusedBeforeTheStreamStartBoundary pins the
// refusal that subsumes the stream start boundary below 1.
func TestObservationSequenceZeroIsRefusedBeforeTheStreamStartBoundary(t *testing.T) {
	object := validObservationEventObject()
	object["sequence"] = json.Number("0")
	observationCandidate(object).assertRefuses(t, "member sequence must be greater than zero")
	streamCandidate(object).assertRefuses(t, "member sequence must be greater than zero")
}

func literalBoundaryProofs() []literalBoundaryProof {
	leaseAt := func(epoch int, reason string, predecessor, checkpoint any) func(t *testing.T, value int) couplingCandidate {
		return func(t *testing.T, value int) couplingCandidate {
			object := validLeaseRecordObject()
			object["epoch"] = json.Number(fmt.Sprint(epoch))
			object["reason"] = reason
			object["predecessor_lease_id"] = predecessor
			object["checkpoint_id"] = checkpoint
			return identityCandidate(SelfRecordID, object)
		}
	}
	observationStream := func(count int) func(t *testing.T, value int) couplingCandidate {
		return func(t *testing.T, value int) couplingCandidate {
			objects := make([]map[string]any, 0, count)
			for index := range count {
				event := validObservationEventObject()
				event["sequence"] = json.Number(fmt.Sprint(index + 1))
				objects = append(objects, event)
			}
			return streamCandidate(objects...)
		}
	}
	sizedOpaque := func(t *testing.T, value int) couplingCandidate {
		object := validProviderIdentityRecordObject()
		object["opaque_identity"] = numberedOpaqueIdentity(value)
		return identityCandidate(SelfRecordID, object)
	}
	opaqueValueOfLength := func(t *testing.T, value int) couplingCandidate {
		object := validProviderIdentityRecordObject()
		object["opaque_identity"] = map[string]any{"key": strings.Repeat("x", value)}
		return identityCandidate(SelfRecordID, object)
	}
	terminalOfLength := func(t *testing.T, value int) couplingCandidate {
		return identityCandidate(SelfEventID, sessionEventWithTerminalBackendID(
			"terminal.created", "a"+strings.Repeat("b", value-1)))
	}
	safeBoundaryCounter := func(t *testing.T, value int) couplingCandidate {
		object := validCheckpointRecordObject(true)
		object["safe_boundary"].(map[string]any)["open_processes"] = json.Number(fmt.Sprint(value))
		return identityCandidate(SelfCheckpointID, object)
	}
	leaseEpochLiteral := func(t *testing.T, value int) couplingCandidate {
		object := validLeaseRecordObject()
		object["epoch"] = json.Number(fmt.Sprint(value))
		return identityCandidate(SelfRecordID, object)
	}
	sanitizedURLs := func(t *testing.T, value int) couplingCandidate {
		object := validWorkspaceGroupRecordObject()
		member := object["members"].([]any)[0].(map[string]any)
		member["sanitized_remote_urls"] = numberedSanitizedURLs(value + 1)
		return identityCandidate(SelfRecordID, object)
	}
	workspaceMembers := func(t *testing.T, value int) couplingCandidate {
		object := validWorkspaceGroupRecordObject()
		object["members"] = numberedManagedWorkspaceMembers(value + 1)
		return identityCandidate(SelfRecordID, object)
	}

	const (
		leaseSpec       = `Section 5.3 Lease Record predecessor_lease_id: "Null only at epoch 1"`
		leaseCreateSpec = `Section 5.3 Lease Record: "An epoch-1 create lease MUST have a null predecessor"; checkpoint_id "Null only for epoch-1 create"`
		streamSpec      = "Section 18.1 Observation stream: the first event has sequence 1 and each later sequence is exactly one greater"
	)

	return []literalBoundaryProof{
		// ValidateObservationStream: len(events) == 0
		{
			key: "core_records.go|ValidateObservationStream|len(events) == 0", value: 0,
			spec: streamSpec, refusal: "stream must contain at least one event",
			build: func(t *testing.T, value int) couplingCandidate { return streamCandidate() },
		},
		{
			key: "core_records.go|ValidateObservationStream|len(events) == 0", value: 1,
			spec: streamSpec, build: observationStream(1),
		},
		// ValidateObservationStream: index == 0 selects the start-of-stream arm.
		{
			key: "core_records.go|ValidateObservationStream|index == 0", value: 0,
			spec: streamSpec, build: observationStream(1),
		},
		{
			key: "core_records.go|ValidateObservationStream|index == 0", value: 1,
			spec: streamSpec, build: observationStream(2),
		},
		// ValidateObservationStream: sequence != 1
		{
			key: "core_records.go|ValidateObservationStream|sequence != 1", value: 1,
			spec: streamSpec, build: observationStream(1),
		},
		{
			key: "core_records.go|ValidateObservationStream|sequence != 1", value: 2,
			spec: streamSpec, refusal: "stream sequence must start at 1, got 2",
			build: func(t *testing.T, value int) couplingCandidate {
				event := validObservationEventObject()
				event["sequence"] = json.Number(fmt.Sprint(value))
				return streamCandidate(event)
			},
		},
		// requirePositiveUint: value == 0
		{
			key: "core_records.go|requirePositiveUint|value == 0", value: 0,
			spec:  "Section 5.3 Lease Record epoch is a positive integer",
			build: leaseEpochLiteral, refusal: "member epoch must be greater than zero",
		},
		{
			key: "core_records.go|requirePositiveUint|value == 0", value: 1,
			spec: "Section 5.3 Lease Record epoch is a positive integer",
			build: func(t *testing.T, value int) couplingCandidate {
				return leaseAt(value, "create", nil, nil)(t, value)
			},
		},
		// requireTerminalBackendID: len(value) > 128
		{
			key: "core_records.go|requireTerminalBackendID|len(value) > 128", value: 128,
			spec: "Section 4.B terminal_backend_id is 1..128 ASCII bytes", build: terminalOfLength,
		},
		{
			key: "core_records.go|requireTerminalBackendID|len(value) > 128", value: 129,
			spec: "Section 4.B terminal_backend_id is 1..128 ASCII bytes", build: terminalOfLength,
			refusal: "must contain 1..128 ASCII bytes",
		},
		// validateLeaseRecord: epoch != 1
		{
			key: "core_records.go|validateLeaseRecord|epoch != 1", value: 1,
			spec: leaseSpec, build: leaseAt(1, "create", nil, nil),
		},
		{
			// The case review found missing. Epoch 2 is the first epoch at which
			// "Null only at epoch 1" starts biting, and every previous negative
			// used epoch 4, so `epoch >= 3` admitted this record.
			key: "core_records.go|validateLeaseRecord|epoch != 1", value: 2,
			spec: leaseSpec, refusal: "Lease Record predecessor_lease_id must be non-null after epoch 1",
			build: leaseAt(2, "graceful_takeover", nil, zeroDigest),
		},
		// validateLeaseRecord: epoch == 1 selects the epoch-1-create arm.
		{
			key: "core_records.go|validateLeaseRecord|epoch == 1", value: 1,
			spec: leaseCreateSpec, refusal: "Lease Record epoch-1 create predecessor_lease_id must be null",
			build: leaseAt(1, "create", priorLease, nil),
		},
		{
			// At epoch 2 the same create lease is admitted, so a narrowing of
			// `epoch == 1` to `epoch <= 2` reddens here rather than staying green.
			key: "core_records.go|validateLeaseRecord|epoch == 1", value: 2,
			spec: leaseCreateSpec, build: leaseAt(2, "create", priorLease, zeroDigest),
		},
		// validateProviderIdentityRecord: len(opaque) > 32
		{
			key: "core_records.go|validateProviderIdentityRecord|len(opaque) > 32", value: 32,
			spec: "Section 5.5 Provider Identity Record opaque_identity holds at most 32 members", build: sizedOpaque,
		},
		{
			key: "core_records.go|validateProviderIdentityRecord|len(opaque) > 32", value: 33,
			spec: "Section 5.5 Provider Identity Record opaque_identity holds at most 32 members", build: sizedOpaque,
			refusal: "opaque_identity exceeds maximum length 32",
		},
		// validateProviderIdentityRecord: length < 1
		{
			key: "core_records.go|validateProviderIdentityRecord|length < 1", value: 0,
			spec: "Section 5.5 opaque_identity values are 1..1024 Unicode characters", build: opaqueValueOfLength,
			refusal: `opaque_identity["key"] must contain 1..1024 Unicode characters`,
		},
		{
			key: "core_records.go|validateProviderIdentityRecord|length < 1", value: 1,
			spec: "Section 5.5 opaque_identity values are 1..1024 Unicode characters", build: opaqueValueOfLength,
		},
		// validateProviderIdentityRecord: length > 1024
		{
			key: "core_records.go|validateProviderIdentityRecord|length > 1024", value: 1024,
			spec: "Section 5.5 opaque_identity values are 1..1024 Unicode characters", build: opaqueValueOfLength,
		},
		{
			key: "core_records.go|validateProviderIdentityRecord|length > 1024", value: 1025,
			spec: "Section 5.5 opaque_identity values are 1..1024 Unicode characters", build: opaqueValueOfLength,
			refusal: `opaque_identity["key"] must contain 1..1024 Unicode characters`,
		},
		// validateSafeBoundaryEvidence: value != 0
		{
			key: "core_records.go|validateSafeBoundaryEvidence|value != 0", value: 0,
			spec: "Section 5.4 Safe Boundary Evidence open_processes and open_database_handles are zero", build: safeBoundaryCounter,
		},
		{
			key: "core_records.go|validateSafeBoundaryEvidence|value != 0", value: 1,
			spec: "Section 5.4 Safe Boundary Evidence open_processes and open_database_handles are zero", build: safeBoundaryCounter,
			refusal: "Safe Boundary Evidence open_processes must be zero",
		},
		// validateSortedUniqueSanitizedGitURLs: index > 0
		{
			key: "core_records.go|validateSortedUniqueSanitizedGitURLs|index > 0", value: 0,
			spec: "Section 5.6 sanitized_remote_urls is strictly sorted and unique", build: sanitizedURLs,
		},
		{
			key: "core_records.go|validateSortedUniqueSanitizedGitURLs|index > 0", value: 1,
			spec: "Section 5.6 sanitized_remote_urls is strictly sorted and unique",
			build: func(t *testing.T, value int) couplingCandidate {
				candidate := sanitizedURLs(t, value)
				member := candidate.object["members"].([]any)[0].(map[string]any)
				urls := member["sanitized_remote_urls"].([]any)
				urls[0], urls[1] = urls[1], urls[0]
				return candidate
			},
			refusal: "member sanitized_remote_urls must be strictly sorted and unique",
		},
		// validateWorkspaceGroupRecord: index > 0
		{
			key: "core_records.go|validateWorkspaceGroupRecord|index > 0", value: 0,
			spec: "Section 2 Members are sorted by workspace_id (SPEC.md:2146)", build: workspaceMembers,
		},
		{
			key: "core_records.go|validateWorkspaceGroupRecord|index > 0", value: 1,
			spec: "Section 2 Members are sorted by workspace_id (SPEC.md:2146)",
			build: func(t *testing.T, value int) couplingCandidate {
				candidate := workspaceMembers(t, value)
				members := candidate.object["members"].([]any)
				members[0], members[1] = members[1], members[0]
				return candidate
			},
			refusal: "Workspace Group Record members must be sorted by workspace_id",
		},
	}
}
