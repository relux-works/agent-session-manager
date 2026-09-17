package config

// Port of the CR6 independent review probe TestReviewerLegacyCannotOverwriteCommittedV4
// (TASK-260909-2ez769_review-evidence-rev6.zip, probes/config/reviewer_legacy_test.go).
// RED-first regression for F1: a delayed legacy migration must never overwrite
// an admitted Config4 at an unchanged generation.
//
// The repair serializes legacy migration with joint commits under the shared
// exclusive authorization lock, so the reviewer's synchronous probe shape (a
// nested Migrate+Apply running inside the outer rename) now blocks on the
// held lock instead of overwriting: the pre-fix RED log shows the overwrite
// failure and the post-fix log shows the blocked stack (see the rev7
// evidence tarball). This port stages the same race as a goroutine against
// the gated rename — the CR5 restore-probe precedent for a repair that
// introduces blocking — and keeps the reviewer's final-state assertion.
// Symbol names deliberately differ from the review probe so the reviewer
// can drop the original probe file beside this candidate without a
// redeclaration.

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

type legacyRaceOutcome struct {
	result MigrationResult
	err    error
}

type legacyRaceFS struct {
	osMigrationFileSystem
	filename string
	atRename chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (f *legacyRaceFS) Rename(from, to string) error {
	if to == f.filename {
		f.once.Do(func() { close(f.atRename) })
		<-f.release
	}
	return f.osMigrationFileSystem.Rename(from, to)
}

func TestLegacyMigrateCannotOverwriteCommittedV4(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	before := fixture.trust(t).Generation
	snapshot, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _ := snapshot.Configuration()
	v2, err := encodeVersion2(loaded.Value, DecodeContext{RuntimePlatform: fixture.inputs.Platform})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename, v2, 0o600); err != nil {
		t.Fatal(err)
	}
	filesystem := &legacyRaceFS{filename: fixture.filename, atRename: make(chan struct{}), release: make(chan struct{})}
	outerDone := make(chan legacyRaceOutcome, 1)
	go func() {
		result, err := migrate(fixture.inputs, nil, MigrationOptions{TargetVersion: CurrentVersion}, filesystem)
		outerDone <- legacyRaceOutcome{result: result, err: err}
	}()
	select {
	case <-filesystem.atRename:
	case <-time.After(10 * time.Second):
		t.Fatal("outer legacy migrate did not reach the rename seam")
	}
	// The outer migrate holds the exclusive authorization lock, paused at
	// its replacement rename. A nested legacy migration prepared against
	// the same Config2 source must serialize behind it, never run beside
	// it: completion here would be the lost-update interleaving.
	nestedDone := make(chan legacyRaceOutcome, 1)
	go func() {
		result, err := Migrate(fixture.inputs, nil, MigrationOptions{TargetVersion: CurrentVersion})
		nestedDone <- legacyRaceOutcome{result: result, err: err}
	}()
	select {
	case outcome := <-nestedDone:
		t.Fatalf("nested legacy migrate completed while the outer held the lock: result=%+v err=%v", outcome.result, outcome.err)
	case <-time.After(250 * time.Millisecond):
	}
	close(filesystem.release)
	var outer legacyRaceOutcome
	select {
	case outer = <-outerDone:
	case <-time.After(10 * time.Second):
		t.Fatal("outer legacy migrate did not complete after release")
	}
	if outer.err != nil || !outer.result.Changed {
		t.Fatalf("outer legacy migrate result=%+v err=%v, want Changed with nil error", outer.result, outer.err)
	}
	var nested legacyRaceOutcome
	select {
	case nested = <-nestedDone:
	case <-time.After(10 * time.Second):
		t.Fatal("nested legacy migrate did not finish after the outer released the lock")
	}
	// The nested migrate prepared against Config2 but the outer already
	// committed Config3: resuming the delayed write must refuse stale
	// instead of reinstalling bytes over the committed state.
	if !errors.Is(nested.err, ErrMigrationStaleSource) {
		t.Fatalf("nested legacy migrate err = %v, want %v", nested.err, ErrMigrationStaleSource)
	}
	// A fresh joint migration now commits Config4 on the migrated source,
	// and the coherent reader admits exactly that pair: no delayed legacy
	// write overwrote it at an unchanged generation.
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Generation != before+1 {
		t.Fatalf("apply generation = %d, want %d", applied.Generation, before+1)
	}
	after, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	final, ok := after.Config.Configuration()
	if !ok {
		t.Fatal("missing final configuration")
	}
	if final.SourceVersion != Version4 || after.Generation != before+1 {
		t.Fatalf("final state = %s/generation %d, want %s/generation %d", final.SourceVersion, after.Generation, Version4, before+1)
	}
}
