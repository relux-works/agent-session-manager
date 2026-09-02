package localstore

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// The projection rollback cases in projection_test.go inject a chosen error at
// a commit boundary, which proves the transaction boundaries hold but says
// nothing about how the implementation classifies a failure the engine raises
// on its own. A full volume is exactly such a failure, and it is the one whose
// misclassification is expensive: SQLITE_FULL routed to the corruption path
// would quarantine a perfectly valid index and then try to rebuild it from
// scratch, writing more data onto the volume that just ran out of room.
//
// These cases therefore make SQLite itself raise SQLITE_FULL. PRAGMA
// max_page_count pinned to the database's current size makes every page the
// statement needs unavailable, and SQLite reports that with the same result
// code and the same "database or disk is full" message a genuinely exhausted
// filesystem produces. The limit lives on the connection rather than in the
// file, so the constraint disappears the moment the failed open closes it and
// the recovery half of each case runs against an unconstrained database.

// exhaustProjectionVolume pins the index connection to its current page count.
// Any statement that needs to grow the database then fails with SQLITE_FULL.
func exhaustProjectionVolume(ctx context.Context, db *sql.DB) error {
	var pageCount int
	if err := db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return err
	}
	var applied int
	if err := db.QueryRowContext(ctx, fmt.Sprintf("PRAGMA max_page_count = %d", pageCount)).Scan(&applied); err != nil {
		return err
	}
	if applied != pageCount {
		return fmt.Errorf("max_page_count = %d, want the current size %d", applied, pageCount)
	}
	return nil
}

// TestOpenProjectionRebuildOnFullVolumeRollsBackWithoutQuarantiningTheIndex
// drives OpenProjection against an index whose volume cannot grow, with more
// authoritative objects than the existing rows can be replaced in place.
//
// Three properties are asserted, and each fails differently if the engine error
// were classified as corruption instead of as a rebuild failure: the sentinel
// would be ErrProjectionCorrupt, an index-recovery directory would appear under
// the state root, and the prior usable index would have been moved out from
// under the caller rather than left intact.
func TestOpenProjectionRebuildOnFullVolumeRollsBackWithoutQuarantiningTheIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	state, _ := paths.Path(PathState)
	stateRoot := state.Value.String()
	indexPath := filepath.Join(stateRoot, projectionFilename)

	established := putTestBlobs(t, store, "established", 1)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(established index) error = %v", err)
	}
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	before := requireProjectionIndexIdentity(t, indexPath)

	// Enough additional objects that replacing the single existing row cannot
	// be absorbed by the pages the deletion frees.
	all := append(established, putTestBlobs(t, store, "growth", 150)...)

	_, recovery, err := openProjection(ctx, paths, projectionHooks{afterDatabaseOpen: exhaustProjectionVolume})
	if !errors.Is(err, ErrProjectionRebuild) {
		t.Fatalf("openProjection(full volume) error = %v, want %v", err, ErrProjectionRebuild)
	}
	if errors.Is(err, ErrProjectionCorrupt) || errors.Is(err, ErrProjectionIncompatible) {
		t.Fatalf("openProjection(full volume) error = %v, want a full volume classified as a rebuild failure, not as corruption", err)
	}
	if !strings.Contains(err.Error(), "disk is full") {
		t.Fatalf("openProjection(full volume) error = %v, want the SQLITE_FULL cause preserved in the report", err)
	}
	if recovery != (ProjectionRecovery{}) {
		t.Fatalf("openProjection(full volume) recovery = %+v, want no recovery claimed", recovery)
	}
	assertNoProjectionRecoveryDirectory(t, stateRoot)
	if after := requireProjectionIndexIdentity(t, indexPath); !os.SameFile(before, after) {
		t.Fatal("the prior usable index was replaced by a rebuild that never committed")
	}
	requireProjectionRowCount(t, ctx, indexPath, 1)

	recovered, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(after the volume recovered) error = %v", err)
	}
	defer recovered.Close()
	if recovery.RecoveredCorruption {
		t.Fatalf("OpenProjection(after the volume recovered) recovery = %+v, want no corruption recovery", recovery)
	}
	objects, err := recovered.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != len(all) {
		t.Fatalf("recovered Objects() count = %d, want every authoritative object %d", len(objects), len(all))
	}
}

// TestOpenProjectionMigrationOnFullVolumeLeavesNoPartialSchemaBehind drives the
// same exhaustion at the earlier boundary, where the index file exists but
// carries no schema at all. The migration statements cannot allocate a page, so
// the property under test is idempotent recovery: the refused open must leave
// nothing half-created, and the next open on a volume with room must migrate
// and rebuild as if the first had never run.
func TestOpenProjectionMigrationOnFullVolumeLeavesNoPartialSchemaBehind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	state, _ := paths.Path(PathState)
	stateRoot := state.Value.String()
	indexPath := filepath.Join(stateRoot, projectionFilename)
	objects := putTestBlobs(t, store, "migration", 3)

	_, _, err := openProjection(ctx, paths, projectionHooks{afterDatabaseOpen: exhaustProjectionVolume})
	if !errors.Is(err, ErrProjectionMigration) {
		t.Fatalf("openProjection(full volume at migration) error = %v, want %v", err, ErrProjectionMigration)
	}
	if errors.Is(err, ErrProjectionCorrupt) || errors.Is(err, ErrProjectionIncompatible) {
		t.Fatalf("openProjection(full volume at migration) error = %v, want a full volume classified as a migration failure", err)
	}
	if !strings.Contains(err.Error(), "disk is full") {
		t.Fatalf("openProjection(full volume at migration) error = %v, want the SQLITE_FULL cause preserved in the report", err)
	}
	assertNoProjectionRecoveryDirectory(t, stateRoot)

	db := openRawProjection(t, indexPath)
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("rolled-back migration user_version = %d, want the unmigrated 0", version)
	}
	var declared int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&declared); err != nil {
		t.Fatal(err)
	}
	if declared != 0 {
		t.Fatalf("rolled-back migration left %d schema objects, want none", declared)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	projection, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(after the volume recovered) error = %v", err)
	}
	defer projection.Close()
	if recovery.RecoveredCorruption {
		t.Fatalf("OpenProjection(after the volume recovered) recovery = %+v, want no corruption recovery", recovery)
	}
	indexed, err := projection.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(indexed) != len(objects) {
		t.Fatalf("recovered Objects() count = %d, want every authoritative object %d", len(indexed), len(objects))
	}
}

// putTestBlobs installs count distinct verified blobs through the production
// object-store entry point and returns their digests. The label keeps separate
// calls in one test from producing the same content, and therefore the same
// immutable object, twice.
func putTestBlobs(t *testing.T, store *ObjectStore, label string, count int) []scalar.Digest {
	t.Helper()
	digests := make([]scalar.Digest, 0, count)
	for index := 0; index < count; index++ {
		content := []byte(fmt.Sprintf("full volume authoritative object %s/%d", label, index))
		digest := scalar.SHA256Digest(content)
		if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatalf("PutBlob(fixture %d) error = %v", index, err)
		}
		digests = append(digests, digest)
	}
	return digests
}

func requireProjectionIndexIdentity(t *testing.T, indexPath string) os.FileInfo {
	t.Helper()
	info, err := os.Lstat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func requireProjectionRowCount(t *testing.T, ctx context.Context, indexPath string, want int) {
	t.Helper()
	db := openRawProjection(t, indexPath)
	defer db.Close()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM immutable_objects").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("index row count = %d, want the prior usable %d", count, want)
	}
	var metadata int
	if err := db.QueryRowContext(ctx, "SELECT object_count FROM projection_metadata WHERE id = 1").Scan(&metadata); err != nil {
		t.Fatal(err)
	}
	if metadata != want {
		t.Fatalf("index metadata object_count = %d, want the prior usable %d", metadata, want)
	}
}

// assertNoProjectionRecoveryDirectory proves a failure was not routed to the
// quarantine-and-rebuild path. That path is the correct response to corruption
// and the wrong response to a full volume, where it would consume more of the
// space whose exhaustion caused the failure.
func assertNoProjectionRecoveryDirectory(t *testing.T, stateRoot string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(stateRoot, "index-recovery"))
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("index-recovery holds %d directories: the index was quarantined for a fault that is not corruption", len(entries))
	}
}
