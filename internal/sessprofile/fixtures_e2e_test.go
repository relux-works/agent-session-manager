package sessprofile

import (
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// fixtureChain is one P0 -> E1 -> C1 history: the creation profile
// P0, the authoritative change E1 to P1, and the checkpoint heads C1
// closing over E1.
type fixtureChain struct {
	repository *sessrepo.Repository
	record     Record
	events     []Event
	p0         string
	p1         string
	e1         string
	heads      []string
}

// seedFixtureChain builds P0 -> E1 -> C1 in a fresh repository: a
// session.created event, a first launch carrying the creation pair,
// the authoritative profile.changed E1 to P1, and a
// checkpoint.created event closing the C1 heads over E1.
func seedFixtureChain(t *testing.T, p0, p1 string, taskBoard bool) *fixtureChain {
	t.Helper()
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", p0)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	launchType := "provider.launched"
	var launch map[string]any
	if taskBoard {
		launchType = "task_board.launched"
		launch = taskBoardLaunchedPayload(p0, "")
	} else {
		launch = launchedPayload("codex", "0.147.0", p0, "", p0)
	}
	second := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, launchType, launch)
	change := appendTestEvent(t, repository, testSessionID, []string{second}, 1, testLeaseID, 3, "profile.changed", changedPayload(p0, p1, true))
	announced := appendTestEvent(t, repository, testSessionID, []string{change}, 1, testLeaseID, 4, "checkpoint.created", checkpointAnnouncedPayload())
	record, events := decodeTestChain(t, repository, testSessionID)
	return &fixtureChain{repository: repository, record: record, events: events, p0: p0, p1: p1, e1: change, heads: []string{announced}}
}

// closurePair derives the C1 pair through the production closure
// entry and requires the exact P1/E1 authority.
func (chain *fixtureChain) closurePair(t *testing.T) Pair {
	t.Helper()
	pair, err := DeriveForHeads(chain.record, chain.events, chain.heads)
	if err != nil {
		t.Fatalf("DeriveForHeads(C1) error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: chain.p1, Source: chain.e1, HasSource: true}, "C1 pair")
	projected, err := (&Projector{Repo: chain.repository}).ProjectForHeads(testSessionID, chain.heads)
	if err != nil {
		t.Fatalf("ProjectForHeads(C1) error = %v", err)
	}
	mustPairEqual(t, projected, pair, "ProjectForHeads(C1)")
	return pair
}

func TestFixtureProfileDirectTakeover(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, false)
	pair := chain.closurePair(t)
	// Bundle is absent on the direct path (no bundle object is
	// projected for direct sessions); materialization finalize,
	// plugin resume, and the resumed event all carry P1/E1.
	final, err := FinalizePair(ActivationDirectResumed, pair, "", false)
	if err != nil {
		t.Fatalf("FinalizePair error = %v", err)
	}
	mustPairEqual(t, final, pair, "finalize pair")
	if err := CheckFinalizeParams(chain.p1, true, chain.e1, true, final); err != nil {
		t.Fatalf("CheckFinalizeParams error = %v", err)
	}
	if err := CheckResumeRequestProfile(chain.p1, pair); err != nil {
		t.Fatalf("CheckResumeRequestProfile error = %v", err)
	}
	if err := CheckResumedPair(pair, pair); err != nil {
		t.Fatalf("CheckResumedPair error = %v", err)
	}
}

func TestFixtureProfileDirectResume(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, false)
	pair := chain.closurePair(t)
	// Plugin resume and the resumed event carry P1/E1; the
	// creation P0 is ignored, never a fallback.
	if pair.Profile == chain.p0 && !pair.HasSource {
		t.Fatal("C1 pair fell back to the creation value")
	}
	if err := CheckResumeRequestProfile(chain.p1, pair); err != nil {
		t.Fatalf("CheckResumeRequestProfile error = %v", err)
	}
	if err := CheckResumedPair(Pair{Profile: chain.p1, Source: chain.e1, HasSource: true}, pair); err != nil {
		t.Fatalf("CheckResumedPair error = %v", err)
	}
	if err := CheckResumedPair(Pair{Profile: chain.p0}, pair); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckResumedPair(P0) error = %v, want integrity_failure", err)
	}
}

func TestFixtureProfileDirectFork(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, false)
	pair := chain.closurePair(t)
	// The new Session Record creation profile is P1; the fork
	// event carries P1/null as new-session authority plus
	// source_profile_event_id=E1; the resumed event in the new
	// session carries P1/null.
	creation := ForkProjection(pair)
	if creation != chain.p1 {
		t.Fatalf("ForkProjection = %q, want %q", creation, chain.p1)
	}
	forkPair := Pair{Profile: creation}
	if err := CheckForkPair(forkPair, chain.e1, true, creation, chain.e1, true); err != nil {
		t.Fatalf("CheckForkPair error = %v", err)
	}
	if err := CheckResumedPair(forkPair, forkPair); err != nil {
		t.Fatalf("CheckResumedPair(new session) error = %v", err)
	}
	final, err := FinalizePair(ActivationDirectResumed, pair, creation, true)
	if err != nil {
		t.Fatalf("FinalizePair(fork) error = %v", err)
	}
	mustPairEqual(t, final, forkPair, "fork finalize pair")
	if err := CheckFinalizeParams(creation, true, "", false, final); err != nil {
		t.Fatalf("CheckFinalizeParams(fork) error = %v", err)
	}
}

func TestFixtureProfileTBTakeover(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, true)
	pair := chain.closurePair(t)
	// The bundle projection carries P1/E1; the journaled bridge
	// resume and the events use the same pair.
	bundle := BundlePair(pair)
	mustPairEqual(t, bundle, pair, "bundle pair")
	if err := CheckBundlePair(bundle.Profile, bundle.Source, bundle.HasSource, pair); err != nil {
		t.Fatalf("CheckBundlePair error = %v", err)
	}
	if err := CheckBridgeProfile(chain.p1, pair); err != nil {
		t.Fatalf("CheckBridgeProfile error = %v", err)
	}
	if err := CheckResumedPair(pair, pair); err != nil {
		t.Fatalf("CheckResumedPair error = %v", err)
	}
	final, err := FinalizePair(ActivationTaskBoardResumed, pair, "", false)
	if err != nil {
		t.Fatalf("FinalizePair error = %v", err)
	}
	mustPairEqual(t, final, pair, "finalize pair")
}

func TestFixtureProfileTBResume(t *testing.T) {
	chain := seedFixtureChain(t, ProfileYOLO, ProfileStandard, true)
	pair := chain.closurePair(t)
	// Bundle, bridge resume, and events agree on P1/E1 — here in
	// the yolo-to-standard direction.
	if err := CheckBundlePair(chain.p1, chain.e1, true, pair); err != nil {
		t.Fatalf("CheckBundlePair error = %v", err)
	}
	if err := CheckBridgeProfile(chain.p1, pair); err != nil {
		t.Fatalf("CheckBridgeProfile error = %v", err)
	}
	if err := CheckResumedPair(Pair{Profile: chain.p1, Source: chain.e1, HasSource: true}, pair); err != nil {
		t.Fatalf("CheckResumedPair error = %v", err)
	}
}

func TestFixtureProfileTBFork(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, true)
	pair := chain.closurePair(t)
	// The source bundle carries P1/E1; the new Session Record and
	// the bridge resume use P1/null; the fork event retains E1
	// only in source_profile_event_id.
	if err := CheckBundlePair(chain.p1, chain.e1, true, pair); err != nil {
		t.Fatalf("CheckBundlePair(source bundle) error = %v", err)
	}
	creation := ForkProjection(pair)
	if err := CheckBridgeProfile(creation, Pair{Profile: creation}); err != nil {
		t.Fatalf("CheckBridgeProfile(new session) error = %v", err)
	}
	if err := CheckForkPair(Pair{Profile: creation}, chain.e1, true, creation, chain.e1, true); err != nil {
		t.Fatalf("CheckForkPair error = %v", err)
	}
}

func TestFixtureProfilePIEqualMapping(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, false)
	pair := chain.closurePair(t)
	tuple := provhost.BuildTuple{ProviderID: "pi", ProviderVersion: "0.73.1", Platform: "macos", Architecture: "arm64"}
	standard, err := provhost.ResolveMapping("pi", ProfileStandard, tuple)
	if err != nil {
		t.Fatalf("ResolveMapping(pi standard) error = %v", err)
	}
	yolo, err := provhost.ResolveMapping("pi", ProfileYOLO, tuple)
	if err != nil {
		t.Fatalf("ResolveMapping(pi yolo) error = %v", err)
	}
	if !standard.Equivalent || !yolo.Equivalent {
		t.Fatalf("pi equivalence not disclosed: %+v %+v", standard, yolo)
	}
	// Equal provider tool sets are not authority to erase E1: ax
	// still persists and reports P1/E1.
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: chain.e1, HasSource: true}, "PI closure pair")
	if err := CheckResumedPair(pair, pair); err != nil {
		t.Fatalf("CheckResumedPair error = %v", err)
	}
}

func TestFixtureIntegrityFailures(t *testing.T) {
	chain := seedFixtureChain(t, ProfileStandard, ProfileYOLO, false)
	pair := chain.closurePair(t)
	// Any fixture that uses P0 after C1, omits E1 where the
	// source is required, or accepts a bundle profile
	// inconsistent with C1 is integrity_failure before
	// activation.
	if err := CheckLaunchPair(Pair{Profile: chain.p0}, pair); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckLaunchPair(P0 after C1) error = %v, want integrity_failure", err)
	}
	if err := CheckResumedPair(Pair{Profile: chain.p1}, pair); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckResumedPair(missing E1) error = %v, want integrity_failure", err)
	}
	if err := CheckBundlePair(chain.p0, "", false, pair); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckBundlePair(inconsistent) error = %v, want integrity_failure", err)
	}
	if err := CheckFinalizeParams(chain.p0, true, "", false, pair); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("CheckFinalizeParams(P0) error = %v, want integrity_failure", err)
	}
}
