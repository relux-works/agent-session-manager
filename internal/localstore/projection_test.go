package localstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestOpenProjectionProductionEntryConfiguresWALSchemaIndexesAndOwnerOnlyFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	for _, content := range [][]byte{[]byte("second"), []byte("first")} {
		digest := scalar.SHA256Digest(content)
		if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatalf("PutBlob() error = %v", err)
		}
	}

	projection, recovery, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
	if err != nil {
		t.Fatalf("OpenProjection() error = %v", err)
	}
	t.Cleanup(func() { _ = projection.Close() })
	if recovery.RecoveredCorruption || recovery.ObjectCount != 2 {
		t.Fatalf("OpenProjection() recovery = %+v, want clean two-object rebuild", recovery)
	}

	var journalMode string
	if err := projection.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
	var synchronous int
	if err := projection.db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if synchronous != 2 {
		t.Fatalf("synchronous = %d, want FULL (2)", synchronous)
	}
	var foreignKeys int
	if err := projection.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want enabled", foreignKeys)
	}

	for _, object := range projectionSchemaV1 {
		var count int
		if err := projection.db.QueryRowContext(ctx,
			"SELECT count(*) FROM sqlite_master WHERE type = ? AND name = ?", object.kind, object.name,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("derived schema object %s %q count = %d, want 1", object.kind, object.name, count)
		}
	}

	for _, suffix := range []string{"", "-wal", "-shm"} {
		info, statErr := os.Lstat(projection.Path() + suffix)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			t.Fatal(statErr)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			t.Errorf("projection sidecar %q mode = %s, want regular 0600", suffix, info.Mode())
		}
	}
	lockInfo, err := os.Lstat(projection.Path() + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	if !lockInfo.Mode().IsRegular() || lockInfo.Mode().Perm() != 0o600 {
		t.Errorf("projection lock mode = %s, want regular 0600", lockInfo.Mode())
	}
	if err := filepath.WalkDir(store.DataRoot(), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.HasPrefix(entry.Name(), "index.sqlite") {
			t.Errorf("SQLite transfer-excluded file appeared under durable data: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProjectionSchemaInventoryDerivesSQLiteInternalRowsFromCatalog(t *testing.T) {
	ctx := context.Background()
	inventory, err := deriveProjectionSchemaInventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	declaredTables := make(map[string]struct{})
	for _, object := range projectionSchemaV1 {
		if object.kind == "table" {
			declaredTables[object.name] = struct{}{}
		}
	}
	implicitIndexes := 0
	for _, entry := range inventory {
		if entry.hasStatement {
			continue
		}
		implicitIndexes++
		if entry.kind != "index" {
			t.Errorf("catalog-derived internal object = %+v, want index", entry)
		}
		if _, ok := declaredTables[entry.table]; !ok {
			t.Errorf("catalog-derived internal index table = %q, want a declared table", entry.table)
		}
	}
	if implicitIndexes == 0 {
		t.Fatal("catalog-derived schema inventory has no SQLite internal indexes")
	}
}

func TestOpenProjectionDeterministicallyRebuildsAndRemovesForgedRows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	contents := [][]byte{[]byte("zeta"), []byte("alpha"), []byte("middle")}
	for _, content := range contents {
		digest := scalar.SHA256Digest(content)
		if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
	}

	first, _, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	firstObjects, err := first.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	firstMetadata, err := first.Metadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.db.ExecContext(ctx, `
		INSERT INTO immutable_objects(storage_class, representation, digest, size, relative_path)
		VALUES('blob', 'raw', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 1, 'forged')
	`); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, _, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	secondObjects, err := second.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secondMetadata, err := second.Metadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(firstObjects, secondObjects) {
		t.Fatalf("rebuilt objects = %+v, want authoritative %+v", secondObjects, firstObjects)
	}
	if firstMetadata != secondMetadata {
		t.Fatalf("rebuilt metadata = %+v, want deterministic %+v", secondMetadata, firstMetadata)
	}
	for index := 1; index < len(secondObjects); index++ {
		if secondObjects[index-1].Digest.String() >= secondObjects[index].Digest.String() {
			t.Fatalf("Objects() order is not deterministic: %+v", secondObjects)
		}
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}

	third, _, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = third.Close() })
	thirdMetadata, err := third.Metadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if thirdMetadata != firstMetadata {
		t.Fatalf("identical retry metadata = %+v, want %+v", thirdMetadata, firstMetadata)
	}
}

func TestOpenProjectionRecoversCorruptIndexFromAuthoritativeObjects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	content := []byte("authoritative")
	digest := scalar.SHA256Digest(content)
	if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	state, _ := paths.Path(PathState)
	indexPath := filepath.Join(state.Value.String(), projectionFilename)
	if err := os.WriteFile(indexPath, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.WriteFile(indexPath+suffix, []byte("corrupt sidecar"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	projection, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(corrupt index) error = %v", err)
	}
	t.Cleanup(func() { _ = projection.Close() })
	if !recovery.RecoveredCorruption || recovery.RecoveryDirectory == "" || recovery.ObjectCount != 1 {
		t.Fatalf("recovery = %+v, want quarantined corruption and one rebuilt object", recovery)
	}
	quarantined, err := os.ReadFile(filepath.Join(recovery.RecoveryDirectory, projectionFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(quarantined) != "not a sqlite database" {
		t.Fatalf("quarantined index bytes = %q", quarantined)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Lstat(filepath.Join(recovery.RecoveryDirectory, projectionFilename+suffix)); err != nil {
			t.Fatalf("quarantined sidecar %s stat error = %v", suffix, err)
		}
	}
	objects, err := projection.Objects(ctx)
	if err != nil || len(objects) != 1 || objects[0].Digest != digest {
		t.Fatalf("Objects() = %+v, %v, want rebuilt authoritative digest %s", objects, err, digest)
	}
}

func TestOpenProjectionRecoveryNeverMutatesAuthoritativeBlobEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	content := []byte("retained immutable evidence")
	digest := scalar.SHA256Digest(content)
	put, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(put.Path)
	if err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte("corrupt projection"), 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Close()
	if !recovery.RecoveredCorruption {
		t.Fatalf("recovery = %+v, want corrupt-index recovery", recovery)
	}
	after, err := os.Lstat(put.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("projection recovery replaced authoritative immutable evidence")
	}
	got, err := os.ReadFile(put.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("authoritative bytes after recovery = %q, want %q", got, content)
	}
}

func TestOpenProjectionRefusesCorruptAuthoritativeObjectWithoutTreatingReadFailureAsAbsence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	content := []byte("authoritative")
	digest := scalar.SHA256Digest(content)
	put, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(put.Path, []byte("forged-value!"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(corrupt authoritative object) error = %v, want %v", err, ErrProjectionSourceIntegrity)
	}
	db := openRawProjection(t, indexPath)
	defer db.Close()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM immutable_objects").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("prior usable projection row count = %d, want 1 after source refusal", count)
	}
}

func TestOpenProjectionRefusesUnsafeSourceRootBeforeCreatingIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	objectsRoot := filepath.Join(store.DataRoot(), "objects")
	if err := os.Remove(filepath.Join(objectsRoot, "sha256")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(objectsRoot); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(filepath.Dir(store.DataRoot()), "foreign-objects")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, objectsRoot); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)

	if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(symlink source root) error = %v, want %v", err, ErrProjectionSourceIntegrity)
	}
	state, _ := paths.Path(PathState)
	if _, err := os.Lstat(filepath.Join(state.Value.String(), projectionFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("index stat after source refusal = %v, want not created", err)
	}
}

func TestOpenProjectionRefusesUnsafeExistingWALBeforeSQLiteCanFollowIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(filepath.Dir(indexPath), "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, indexPath+"-wal"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrUnsafeOwnership) {
		t.Fatalf("OpenProjection(symlink WAL) error = %v, want %v", err, ErrUnsafeOwnership)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unchanged" {
		t.Fatalf("unsafe WAL target bytes = %q, want unchanged", got)
	}
}

func TestOpenProjectionRefusesUnsafeExistingProcessLockWithoutFollowingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := projection.Path() + ".lock"
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(filepath.Dir(lockPath), "lock-sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, lockPath); err != nil {
		t.Fatal(err)
	}

	if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrUnsafeOwnership) {
		t.Fatalf("OpenProjection(symlink process lock) error = %v, want %v", err, ErrUnsafeOwnership)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unchanged" {
		t.Fatalf("unsafe process-lock target bytes = %q, want unchanged", got)
	}
}

func TestOpenProjectionRecoversClosedSchemaDriftInsteadOfTrustingUserVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	db := openRawProjection(t, indexPath)
	if _, err := db.ExecContext(ctx, "CREATE TABLE forged_projection_authority(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(schema drift) error = %v", err)
	}
	defer recovered.Close()
	if !recovery.RecoveredCorruption || recovery.RecoveryDirectory == "" {
		t.Fatalf("schema drift recovery = %+v, want quarantined clean rebuild", recovery)
	}
	var count int
	if err := recovered.db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE name = 'forged_projection_authority'",
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("forged schema object count after rebuild = %d, want 0", count)
	}
}

func TestOpenProjectionClosedSchemaRejectsEveryUndeclaredSQLiteObjectKind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	tests := []struct {
		name      string
		statement string
	}{
		{name: "table", statement: "CREATE TABLE forged_schema_table(value TEXT)"},
		{name: "index", statement: "CREATE INDEX forged_schema_index ON immutable_objects(relative_path)"},
		{name: "view", statement: "CREATE VIEW forged_schema_view AS SELECT digest FROM immutable_objects"},
		{
			name: "trigger",
			statement: `CREATE TRIGGER forged_schema_trigger
				AFTER INSERT ON immutable_objects
				BEGIN
					UPDATE immutable_objects
					SET relative_path = 'objects/attacker-controlled'
					WHERE digest = NEW.digest;
				END`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store := openTestStore(t)
			content := []byte("closed schema " + test.name)
			if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
				t.Fatal(err)
			}
			paths := store.resolvedPathsForTest(t)
			projection, _, err := OpenProjection(ctx, paths)
			if err != nil {
				t.Fatal(err)
			}
			indexPath := projection.Path()
			if err := projection.Close(); err != nil {
				t.Fatal(err)
			}
			db := openRawProjection(t, indexPath)
			if _, err := db.ExecContext(ctx, test.statement); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}

			recovered, recovery, err := OpenProjection(ctx, paths)
			if err != nil {
				t.Fatalf("OpenProjection(undeclared %s) error = %v", test.name, err)
			}
			defer recovered.Close()
			if !recovery.RecoveredCorruption {
				t.Fatalf("OpenProjection(undeclared %s) recovery = %+v, want closed-schema recovery", test.name, recovery)
			}
			objects, err := recovered.Objects(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(objects) != 1 || objects[0].RelativePath == "objects/attacker-controlled" {
				t.Fatalf("Objects() after %s recovery = %+v, want authoritative row", test.name, objects)
			}
		})
	}
}

func TestVerifyProjectionSchemaRefusesSQLitePrefixedObjectOfEveryKind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	tests := []struct {
		name      string
		statement string
	}{
		{name: "table", statement: "CREATE TABLE forged_prefixed_table(value TEXT)"},
		{name: "index", statement: "CREATE INDEX forged_prefixed_index ON immutable_objects(relative_path)"},
		{name: "view", statement: "CREATE VIEW forged_prefixed_view AS SELECT digest FROM immutable_objects"},
		{name: "trigger", statement: "CREATE TRIGGER forged_prefixed_trigger AFTER INSERT ON immutable_objects BEGIN SELECT 1; END"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store := openTestStore(t)
			projection, recovery, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
			if err != nil {
				t.Fatal(err)
			}
			if recovery.RecoveredCorruption {
				t.Fatalf("clean catalog-derived schema unexpectedly recovered: %+v", recovery)
			}
			indexPath := projection.Path()
			if err := projection.Close(); err != nil {
				t.Fatal(err)
			}
			db := openRawProjection(t, indexPath)
			defer db.Close()
			if _, err := db.ExecContext(ctx, test.statement); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, "PRAGMA writable_schema = ON"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx,
				"UPDATE sqlite_master SET name = ? WHERE name = ?",
				"sqlite_forged_"+test.name, "forged_prefixed_"+test.name,
			); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, "PRAGMA writable_schema = OFF"); err != nil {
				t.Fatal(err)
			}
			if err := verifyProjectionSchema(ctx, db); !errors.Is(err, ErrProjectionCorrupt) {
				t.Fatalf("verifyProjectionSchema(sqlite_ %s) error = %v, want %v", test.name, err, ErrProjectionCorrupt)
			}
		})
	}
}

func TestOpenProjectionRecoversSQLitePrefixedTriggerBeforeItCanRewriteRebuiltRows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	content := []byte("sqlite-prefixed trigger authority")
	if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	projection, initialRecovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	if initialRecovery.RecoveredCorruption {
		t.Fatalf("clean catalog-derived schema unexpectedly recovered: %+v", initialRecovery)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	db := openRawProjection(t, indexPath)
	if _, err := db.ExecContext(ctx, "PRAGMA writable_schema = ON"); err != nil {
		t.Fatal(err)
	}
	triggerSQL := `CREATE TRIGGER sqlite_forged
		AFTER INSERT ON immutable_objects
		BEGIN
			UPDATE immutable_objects
			SET relative_path = 'objects/attacker-controlled'
			WHERE digest = NEW.digest;
		END`
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sqlite_master(type, name, tbl_name, rootpage, sql)
		VALUES('trigger', 'sqlite_forged', 'immutable_objects', 0, ?)
	`, triggerSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA writable_schema = OFF"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(sqlite_-prefixed trigger) error = %v", err)
	}
	defer recovered.Close()
	if !recovery.RecoveredCorruption {
		t.Fatalf("OpenProjection(sqlite_-prefixed trigger) recovery = %+v, want recovery", recovery)
	}
	objects, err := recovered.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0].RelativePath == "objects/attacker-controlled" {
		t.Fatalf("Objects() after sqlite_-prefixed trigger recovery = %+v, want authoritative row", objects)
	}
}

func TestOpenProjectionRecoversDeclaredSchemaDefinitionDrift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	db := openRawProjection(t, indexPath)
	if _, err := db.ExecContext(ctx, "DROP TABLE immutable_objects"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE immutable_objects (
		storage_class TEXT NOT NULL,
		representation TEXT NOT NULL,
		digest TEXT NOT NULL,
		size INTEGER NOT NULL,
		relative_path TEXT NOT NULL,
		PRIMARY KEY(storage_class, representation, digest)
	) STRICT, WITHOUT ROWID`); err != nil {
		t.Fatal(err)
	}
	for _, object := range projectionSchemaV1 {
		if object.kind != "index" {
			continue
		}
		if _, err := db.ExecContext(ctx, object.sql); err != nil {
			t.Fatalf("restore declared index %q: %v", object.name, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(redefined table) error = %v", err)
	}
	defer recovered.Close()
	if !recovery.RecoveredCorruption {
		t.Fatalf("OpenProjection(redefined table) recovery = %+v, want definition-drift recovery", recovery)
	}
}

func TestVerifyProjectionSchemaRejectsKindDriftWithUnchangedNameAndSQL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	projection, _, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	db := openRawProjection(t, indexPath)
	defer db.Close()
	if _, err := db.ExecContext(ctx, "PRAGMA writable_schema = ON"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE sqlite_master SET type = 'view' WHERE name = 'immutable_objects'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA writable_schema = OFF"); err != nil {
		t.Fatal(err)
	}
	if err := verifyProjectionSchema(ctx, db); !errors.Is(err, ErrProjectionCorrupt) {
		t.Fatalf("verifyProjectionSchema(kind drift) error = %v, want %v", err, ErrProjectionCorrupt)
	}
}

func TestOpenProjectionRecoversValidHeaderInternalCorruptionAndMissingSchemaObject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	t.Run("valid header with corrupt body", func(t *testing.T) {
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		state, _ := paths.Path(PathState)
		indexPath := filepath.Join(state.Value.String(), projectionFilename)
		corrupt := make([]byte, 4096)
		copy(corrupt, []byte("SQLite format 3\x00"))
		if err := os.WriteFile(indexPath, corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
		projection, recovery, err := OpenProjection(context.Background(), paths)
		if err != nil {
			t.Fatalf("OpenProjection(internal corruption) error = %v", err)
		}
		defer projection.Close()
		if !recovery.RecoveredCorruption {
			t.Fatalf("internal corruption recovery = %+v, want recovery", recovery)
		}
	})

	t.Run("missing declared index", func(t *testing.T) {
		ctx := context.Background()
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		projection, _, err := OpenProjection(ctx, paths)
		if err != nil {
			t.Fatal(err)
		}
		indexPath := projection.Path()
		if err := projection.Close(); err != nil {
			t.Fatal(err)
		}
		db := openRawProjection(t, indexPath)
		if _, err := db.ExecContext(ctx, "DROP INDEX idx_immutable_objects_digest"); err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		recovered, recovery, err := OpenProjection(ctx, paths)
		if err != nil {
			t.Fatal(err)
		}
		defer recovered.Close()
		if !recovery.RecoveredCorruption {
			t.Fatalf("missing schema object recovery = %+v, want recovery", recovery)
		}
	})
}

func TestOpenProjectionMigrationConflictRollsBackAndUnsafeMainFileIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	t.Run("version zero schema conflict", func(t *testing.T) {
		ctx := context.Background()
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		state, _ := paths.Path(PathState)
		indexPath := filepath.Join(state.Value.String(), projectionFilename)
		db, err := sql.Open(projectionDriverName, "file:"+filepath.ToSlash(indexPath)+"?mode=rwc")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, "CREATE TABLE projection_metadata(forged TEXT)"); err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(indexPath, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrProjectionMigration) {
			t.Fatalf("OpenProjection(conflicting v0 schema) error = %v, want %v", err, ErrProjectionMigration)
		}
		db = openRawProjection(t, indexPath)
		defer db.Close()
		var version int
		if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			t.Fatalf("conflicting prior schema version = %d, want unchanged 0", version)
		}
	})

	t.Run("group-readable main index", func(t *testing.T) {
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		state, _ := paths.Path(PathState)
		indexPath := filepath.Join(state.Value.String(), projectionFilename)
		if err := os.WriteFile(indexPath, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(indexPath, 0o640); err != nil {
			t.Fatal(err)
		}
		if _, _, err := OpenProjection(context.Background(), paths); !errors.Is(err, ErrUnsafeOwnership) {
			t.Fatalf("OpenProjection(group-readable index) error = %v, want %v", err, ErrUnsafeOwnership)
		}
	})
}

func TestOpenProjectionRefusesUnsafeAuthoritativeModesAndRecoversTruncatedIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	t.Run("group-readable authoritative blob", func(t *testing.T) {
		content := []byte("authority")
		store := openTestStore(t)
		put, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(put.Path, 0o640); err != nil {
			t.Fatal(err)
		}
		if _, _, err := OpenProjection(context.Background(), store.resolvedPathsForTest(t)); !errors.Is(err, ErrProjectionSourceIntegrity) {
			t.Fatalf("OpenProjection(group-readable source) error = %v, want %v", err, ErrProjectionSourceIntegrity)
		}
	})

	t.Run("group-accessible digest shard", func(t *testing.T) {
		content := []byte("authority")
		store := openTestStore(t)
		put, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Dir(put.Path), 0o750); err != nil {
			t.Fatal(err)
		}
		if _, _, err := OpenProjection(context.Background(), store.resolvedPathsForTest(t)); !errors.Is(err, ErrProjectionSourceIntegrity) {
			t.Fatalf("OpenProjection(group-accessible shard) error = %v, want %v", err, ErrProjectionSourceIntegrity)
		}
	})

	t.Run("truncated index header", func(t *testing.T) {
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		state, _ := paths.Path(PathState)
		indexPath := filepath.Join(state.Value.String(), projectionFilename)
		if err := os.WriteFile(indexPath, []byte("short"), 0o600); err != nil {
			t.Fatal(err)
		}
		projection, recovery, err := OpenProjection(context.Background(), paths)
		if err != nil {
			t.Fatalf("OpenProjection(truncated index) error = %v", err)
		}
		defer projection.Close()
		if !recovery.RecoveredCorruption {
			t.Fatalf("truncated index recovery = %+v, want recovery", recovery)
		}
	})
}

func TestOpenProjectionRefusesEveryGroupReadableSidecarBeforeSQLiteOpensIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	for _, mainFileExists := range []bool{false, true} {
		for _, suffix := range []string{"-wal", "-shm", "-journal"} {
			name := "new-index" + suffix
			if mainFileExists {
				name = "existing-index" + suffix
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				store := openTestStore(t)
				paths := store.resolvedPathsForTest(t)
				state, _ := paths.Path(PathState)
				indexPath := filepath.Join(state.Value.String(), projectionFilename)
				if mainFileExists {
					projection, _, err := OpenProjection(ctx, paths)
					if err != nil {
						t.Fatal(err)
					}
					if err := projection.Close(); err != nil {
						t.Fatal(err)
					}
				}
				sidecarPath := indexPath + suffix
				if err := os.WriteFile(sidecarPath, []byte("must remain unopened"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(sidecarPath, 0o640); err != nil {
					t.Fatal(err)
				}

				if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrUnsafeOwnership) {
					t.Fatalf("OpenProjection(group-readable %s, main exists=%v) error = %v, want %v", suffix, mainFileExists, err, ErrUnsafeOwnership)
				}
				info, err := os.Lstat(sidecarPath)
				if err != nil {
					t.Fatalf("sidecar was touched before refusal: %v", err)
				}
				if info.Mode().Perm() != 0o640 {
					t.Fatalf("sidecar mode after refusal = %o, want unchanged 0640", info.Mode().Perm())
				}
			})
		}
	}
}

func TestProjectionPreviouslyUnpinnedRefusalClauses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	// WAL-safety is the first named deliverable of the projection, so the
	// journal-mode gate is pinned by value class rather than by one incidental
	// mode: every mode SQLite really falls back to when WAL is unavailable must
	// be refused, and "wal" must be accepted.
	t.Run("journal mode value class", func(t *testing.T) {
		ctx := context.Background()

		// Production call site. configureProjectionConnection is what
		// openProjectionDatabase calls; a database with no shared-memory
		// backing reports "memory" and has to be refused right there.
		memory, err := sql.Open(projectionDriverName, ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer memory.Close()
		if err := configureProjectionConnection(ctx, memory); !errors.Is(err, ErrProjection) {
			t.Fatalf("configureProjectionConnection(memory) error = %v, want %v", err, ErrProjection)
		}

		// SQLite converts any file-backed journal mode to WAL when asked, so
		// "delete"/"truncate"/"persist"/"off" cannot be observed at that call
		// site with a real index file — they are what SQLite reports when WAL
		// cannot be enabled at all (no shared memory, network filesystem, hot
		// journal). The mode strings below are therefore not hand-written
		// guesses: each one is materialized in a real database and read back
		// from SQLite before it is fed to the gate.
		directory := t.TempDir()
		for _, requested := range []string{"delete", "truncate", "persist", "off", "memory", "wal"} {
			path := filepath.Join(directory, requested+".sqlite")
			db, err := sql.Open(projectionDriverName,
				"file:"+filepath.ToSlash(path)+"?mode=rwc&_journal_mode="+requested)
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(1)
			if _, err := db.ExecContext(ctx, "CREATE TABLE journal_mode_probe(value TEXT)"); err != nil {
				t.Fatal(err)
			}
			var actual string
			if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&actual); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if actual != requested {
				t.Fatalf("SQLite journal_mode for %q = %q; the value class under test is stale", requested, actual)
			}

			err = requireProjectionJournalMode(actual)
			if actual == "wal" {
				if err != nil {
					t.Errorf("requireProjectionJournalMode(%q) error = %v, want accepted", actual, err)
				}
				continue
			}
			if !errors.Is(err, ErrProjection) {
				t.Errorf("requireProjectionJournalMode(%q) error = %v, want %v", actual, err, ErrProjection)
			}
		}
	})

	t.Run("quick check non-ok result", func(t *testing.T) {
		if err := requireProjectionQuickCheckResult("*** malformed ***"); !errors.Is(err, ErrProjectionCorrupt) {
			t.Fatalf("requireProjectionQuickCheckResult(non-ok) error = %v, want %v", err, ErrProjectionCorrupt)
		}
	})

	t.Run("blob size uint53 bounds", func(t *testing.T) {
		if err := validateProjectionBlobSize(int64(MaxBlobSize)); err != nil {
			t.Fatalf("validateProjectionBlobSize(at limit) error = %v", err)
		}
		if err := validateProjectionBlobSize(int64(MaxBlobSize + 1)); !errors.Is(err, ErrProjectionSourceIntegrity) {
			t.Fatalf("validateProjectionBlobSize(past limit) error = %v, want %v", err, ErrProjectionSourceIntegrity)
		}
		if err := validateProjectionBlobSize(-1); !errors.Is(err, ErrProjectionSourceIntegrity) {
			t.Fatalf("validateProjectionBlobSize(negative) error = %v, want %v", err, ErrProjectionSourceIntegrity)
		}
	})

	t.Run("blob changes size after stat", func(t *testing.T) {
		ctx := context.Background()
		store := openTestStore(t)
		content := []byte("source changes after stat")
		if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
		_, _, err := openProjection(ctx, store.resolvedPathsForTest(t), projectionHooks{
			afterBlobStat: func(path string) error {
				return os.Truncate(path, int64(len(content)-1))
			},
		})
		if !errors.Is(err, ErrProjectionSourceIntegrity) {
			t.Fatalf("openProjection(size changed after stat) error = %v, want %v", err, ErrProjectionSourceIntegrity)
		}
		if !strings.Contains(err.Error(), "changed size") {
			t.Fatalf("openProjection(size changed after stat) error = %v, want size cross-check refusal", err)
		}
	})

	// The post-rebuild ensureProjectionFileModes call exists specifically
	// because SQLite creates -wal and -shm itself, after the pre-open
	// verifyProjectionSidecars check has already run. Every suffix in the
	// clause is loosened on its own so narrowing the suffix list reddens.
	t.Run("post-rebuild owner mode", func(t *testing.T) {
		for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
			name := "main index"
			if suffix != "" {
				name = "sidecar " + suffix
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				store := openTestStore(t)
				content := []byte("post-rebuild owner mode " + suffix)
				if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
					t.Fatal(err)
				}
				projection, _, err := openProjection(ctx, store.resolvedPathsForTest(t), projectionHooks{
					afterRebuildCommit: func(path string) error {
						target := path + suffix
						if suffix == "-journal" {
							// A WAL connection never writes a rollback
							// journal, so the real case this clause element
							// covers is a stale journal left behind by a
							// crashed non-WAL writer after the pre-open check.
							if err := os.WriteFile(target, []byte("stale rollback journal"), 0o600); err != nil {
								return err
							}
						}
						// Fail loudly instead of letting a missing file turn
						// the chmod error into a vacuous ErrProjection pass.
						if _, err := os.Lstat(target); err != nil {
							return fmt.Errorf("post-rebuild target %q is absent: %w", target, err)
						}
						return os.Chmod(target, 0o640)
					},
				})
				if projection != nil {
					_ = projection.Close()
				}
				if !errors.Is(err, ErrProjection) {
					t.Fatalf("openProjection(group-readable %s) error = %v, want %v", name, err, ErrProjection)
				}
				if !strings.Contains(err.Error(), "verify index files") {
					t.Fatalf("openProjection(group-readable %s) error = %v, want the post-rebuild owner-only refusal", name, err)
				}
				if strings.Contains(err.Error(), "after rebuild commit") {
					t.Fatalf("openProjection(group-readable %s) failed inside the hook, not at the gate: %v", name, err)
				}
			})
		}
	})

	t.Run("symlink main index", func(t *testing.T) {
		store := openTestStore(t)
		paths := store.resolvedPathsForTest(t)
		state, _ := paths.Path(PathState)
		indexPath := filepath.Join(state.Value.String(), projectionFilename)
		target := filepath.Join(state.Value.String(), "index-target")
		if err := os.WriteFile(target, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, indexPath); err != nil {
			t.Fatal(err)
		}
		if _, _, err := OpenProjection(context.Background(), paths); !errors.Is(err, ErrUnsafeOwnership) {
			t.Fatalf("OpenProjection(symlink main index) error = %v, want %v", err, ErrUnsafeOwnership)
		}
	})

	t.Run("schema verifier noncurrent version", func(t *testing.T) {
		ctx := context.Background()
		store := openTestStore(t)
		projection, _, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
		if err != nil {
			t.Fatal(err)
		}
		defer projection.Close()
		if _, err := projection.db.ExecContext(ctx, "PRAGMA user_version = 0"); err != nil {
			t.Fatal(err)
		}
		if err := verifyProjectionSchema(ctx, projection.db); !errors.Is(err, ErrProjectionCorrupt) {
			t.Fatalf("verifyProjectionSchema(version zero) error = %v, want %v", err, ErrProjectionCorrupt)
		}
	})
}

func TestOpenProjectionRefusesMalformedNamespaceMembersAndIgnoresOnlyRegularStages(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	tests := []struct {
		name   string
		mutate func(t *testing.T, digestRoot string)
		wantOK bool
	}{
		{
			name: "regular file instead of shard",
			mutate: func(t *testing.T, digestRoot string) {
				if err := os.WriteFile(filepath.Join(digestRoot, "zz"), []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "missing digest root is an empty source",
			mutate: func(t *testing.T, digestRoot string) {
				if err := os.Remove(digestRoot); err != nil {
					t.Fatal(err)
				}
			},
			wantOK: true,
		},
		{
			name: "non-hex shard directory",
			mutate: func(t *testing.T, digestRoot string) {
				if err := os.Mkdir(filepath.Join(digestRoot, "gg"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "malformed digest leaf",
			mutate: func(t *testing.T, digestRoot string) {
				shard := filepath.Join(digestRoot, "aa")
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(shard, "short"), []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "directory with digest leaf shape",
			mutate: func(t *testing.T, digestRoot string) {
				shard := filepath.Join(digestRoot, "aa")
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(shard, strings.Repeat("a", 62)), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "regular crash stage is not authoritative",
			mutate: func(t *testing.T, digestRoot string) {
				shard := filepath.Join(digestRoot, "aa")
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(shard, stagedFilePrefix+"crash"), []byte("partial"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantOK: true,
		},
		{
			name: "unsafe crash stage is not silently ignored",
			mutate: func(t *testing.T, digestRoot string) {
				shard := filepath.Join(digestRoot, "aa")
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(digestRoot, "absent"), filepath.Join(shard, stagedFilePrefix+"unsafe")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "group-readable crash stage is not silently ignored",
			mutate: func(t *testing.T, digestRoot string) {
				shard := filepath.Join(digestRoot, "aa")
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
				stage := filepath.Join(shard, stagedFilePrefix+"unsafe-mode")
				if err := os.WriteFile(stage, []byte("partial"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(stage, 0o640); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink with valid digest leaf shape is refused",
			mutate: func(t *testing.T, digestRoot string) {
				shard := filepath.Join(digestRoot, "aa")
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
				leaf := strings.Repeat("a", 62)
				if err := os.Symlink(filepath.Join(digestRoot, "absent"), filepath.Join(shard, leaf)); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStore(t)
			digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
			test.mutate(t, digestRoot)
			projection, recovery, err := OpenProjection(context.Background(), store.resolvedPathsForTest(t))
			if test.wantOK {
				if err != nil {
					t.Fatalf("OpenProjection(stage) error = %v", err)
				}
				defer projection.Close()
				if recovery.ObjectCount != 0 {
					t.Fatalf("stage recovery object count = %d, want 0", recovery.ObjectCount)
				}
				return
			}
			if !errors.Is(err, ErrProjectionSourceIntegrity) {
				t.Fatalf("OpenProjection(malformed namespace) error = %v, want %v", err, ErrProjectionSourceIntegrity)
			}
		})
	}
}

// TestOpenProjectionRefusesUnknownDirectObjectNamespaceMember pins the
// exact-name closure over the objects root in scanAuthoritativeBlobs, which is
// what makes object_count and source_fingerprint attest a complete snapshot: a
// sibling namespace directory that the scan skipped would otherwise be indexed
// as absent.
//
// The class the closure exists for is the near-miss name, not only the
// obviously-foreign one, so both are driven. "sha256-old" shares the whole
// allowed name as a prefix and is therefore admitted by any narrowing of the
// comparison from equality to a prefix or containment match, while "blake3"
// is refused even by such a weakened clause.
func TestOpenProjectionRefusesUnknownDirectObjectNamespaceMember(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	for _, member := range []string{"blake3", "sha256-old"} {
		t.Run(member, func(t *testing.T) {
			store := openTestStore(t)
			unknown := filepath.Join(store.DataRoot(), "objects", member)
			if err := os.Mkdir(unknown, 0o700); err != nil {
				t.Fatal(err)
			}
			projection, _, err := OpenProjection(context.Background(), store.resolvedPathsForTest(t))
			if projection != nil {
				_ = projection.Close()
			}
			if !errors.Is(err, ErrProjectionSourceIntegrity) {
				t.Fatalf("OpenProjection(objects root member %q) error = %v, want %v", member, err, ErrProjectionSourceIntegrity)
			}
		})
	}
}

func TestOpenProjectionRefusesValidDigestStoredWithThreeCharacterShard(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	content := []byte("valid digest in invalid 3+61 split")
	digest := strings.TrimPrefix(scalar.SHA256Digest(content).String(), "sha256:")
	digestRoot := filepath.Join(store.DataRoot(), "objects", "sha256")
	wrongShard := filepath.Join(digestRoot, digest[:3])
	if err := os.Mkdir(wrongShard, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wrongShard, digest[3:]), content, 0o600); err != nil {
		t.Fatal(err)
	}

	projection, _, err := OpenProjection(context.Background(), store.resolvedPathsForTest(t))
	if projection != nil {
		_ = projection.Close()
	}
	if !errors.Is(err, ErrProjectionSourceIntegrity) {
		t.Fatalf("OpenProjection(valid digest with 3+61 split) error = %v, want %v", err, ErrProjectionSourceIntegrity)
	}
}

func TestOpenProjectionEmptySourceAndReadAPIsFailClosedOnInvalidState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	if err := os.Remove(filepath.Join(store.DataRoot(), "objects", "sha256")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(store.DataRoot(), "objects")); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	projection, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(absent source) error = %v", err)
	}
	if recovery.ObjectCount != 0 {
		t.Fatalf("absent source object count = %d, want 0", recovery.ObjectCount)
	}
	if _, err := projection.db.ExecContext(ctx, `
		INSERT INTO immutable_objects(storage_class, representation, digest, size, relative_path)
		VALUES('blob', 'raw', 'forged', 1, 'objects/forged')
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := projection.Objects(ctx); !errors.Is(err, ErrProjectionCorrupt) {
		t.Fatalf("Objects(forged row) error = %v, want %v", err, ErrProjectionCorrupt)
	}
	if _, err := projection.db.ExecContext(ctx, "DELETE FROM projection_metadata"); err != nil {
		t.Fatal(err)
	}
	if _, err := projection.Metadata(ctx); !errors.Is(err, ErrProjection) {
		t.Fatalf("Metadata(absent row) error = %v, want %v", err, ErrProjection)
	}
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}

	if _, _, err := OpenProjection(nil, paths); !errors.Is(err, ErrProjection) {
		t.Fatalf("OpenProjection(nil context) error = %v, want %v", err, ErrProjection)
	}
	var nilProjection *Projection
	if nilProjection.Path() != "" {
		t.Fatalf("nil Projection.Path() = %q, want empty", nilProjection.Path())
	}
	if _, err := nilProjection.Objects(ctx); !errors.Is(err, ErrProjection) {
		t.Fatalf("nil Projection.Objects() error = %v, want %v", err, ErrProjection)
	}
	if _, err := nilProjection.Metadata(ctx); !errors.Is(err, ErrProjection) {
		t.Fatalf("nil Projection.Metadata() error = %v, want %v", err, ErrProjection)
	}
	if err := nilProjection.Close(); err != nil {
		t.Fatalf("nil Projection.Close() error = %v", err)
	}
}

func TestProjectionInternalOperationalFailuresRemainTyped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	if _, _, err := OpenProjection(ctx, ResolvedPaths{}); !errors.Is(err, ErrProjection) {
		t.Fatalf("OpenProjection(zero paths) error = %v, want %v", err, ErrProjection)
	}
	if err := classifySQLiteOpenError("test read", errors.New("read failed")); !errors.Is(err, ErrProjection) || errors.Is(err, ErrProjectionCorrupt) {
		t.Fatalf("classifySQLiteOpenError(non-SQLite) = %v, want non-corruption projection error", err)
	}

	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	state, _ := paths.Path(PathState)
	indexPath := filepath.Join(state.Value.String(), projectionFilename)
	db, err := sql.Open(projectionDriverName, "file:"+filepath.ToSlash(indexPath)+"?mode=rwc")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := migrateProjection(ctx, db, projectionHooks{}); !errors.Is(err, ErrProjectionMigration) {
		t.Fatalf("migrateProjection(closed database) error = %v, want %v", err, ErrProjectionMigration)
	}
	if err := verifyProjectionSchema(ctx, db); !errors.Is(err, ErrProjection) {
		t.Fatalf("verifyProjectionSchema(closed database) error = %v, want %v", err, ErrProjection)
	}
	if err := rebuildProjection(ctx, db, nil, projectionHooks{}); !errors.Is(err, ErrProjectionRebuild) {
		t.Fatalf("rebuildProjection(closed database) error = %v, want %v", err, ErrProjectionRebuild)
	}
	if _, err := quarantineProjectionFiles(state.Value.String(), filepath.Join(state.Value.String(), "absent.sqlite")); err == nil {
		t.Fatal("quarantineProjectionFiles(absent files) error = nil, want refusal")
	}
	if err := os.WriteFile(indexPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(indexPath, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := ensureProjectionFileModes(indexPath); !errors.Is(err, ErrUnsafeOwnership) {
		t.Fatalf("ensureProjectionFileModes(group-readable) error = %v, want %v", err, ErrUnsafeOwnership)
	}
}

func TestOpenProjectionSchemaVersionAcceptsCurrentAndRefusesPastLimitAtProductionEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(current schema) error = %v", err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}

	db := openRawProjection(t, indexPath)
	if _, err := db.ExecContext(ctx, "PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenProjection(ctx, paths); !errors.Is(err, ErrProjectionIncompatible) {
		t.Fatalf("OpenProjection(schema past limit) error = %v, want %v", err, ErrProjectionIncompatible)
	}
	db = openRawProjection(t, indexPath)
	defer db.Close()
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("refused newer index user_version = %d, want unchanged 2", version)
	}
}

func TestOpenProjectionMigrationAndRebuildFailuresRollbackPriorUsableIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	paths := store.resolvedPathsForTest(t)
	migrationFailure := errors.New("injected migration boundary failure")
	if _, _, err := openProjection(ctx, paths, projectionHooks{
		beforeMigrationCommit: func() error { return migrationFailure },
	}); !errors.Is(err, migrationFailure) {
		t.Fatalf("openProjection(migration failure) error = %v, want %v", err, migrationFailure)
	}
	state, _ := paths.Path(PathState)
	indexPath := filepath.Join(state.Value.String(), projectionFilename)
	db := openRawProjection(t, indexPath)
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("rolled-back migration user_version = %d, want prior version 0", version)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	contentA := []byte("a")
	digestA := scalar.SHA256Digest(contentA)
	if _, err := store.PutBlob(digestA, 1, bytes.NewReader(contentA)); err != nil {
		t.Fatal(err)
	}
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	contentB := []byte("b")
	digestB := scalar.SHA256Digest(contentB)
	if _, err := store.PutBlob(digestB, 1, bytes.NewReader(contentB)); err != nil {
		t.Fatal(err)
	}
	rebuildFailure := errors.New("injected rebuild boundary failure")
	if _, _, err := openProjection(ctx, paths, projectionHooks{
		beforeRebuildCommit: func() error { return rebuildFailure },
	}); !errors.Is(err, rebuildFailure) {
		t.Fatalf("openProjection(rebuild failure) error = %v, want %v", err, rebuildFailure)
	}
	db = openRawProjection(t, indexPath)
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM immutable_objects").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rolled-back rebuild row count = %d, want prior usable 1", count)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Close()
	objects, err := recovered.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 || objects[0].Digest == objects[1].Digest {
		t.Fatalf("recovered Objects() = %+v, want both authoritative objects", objects)
	}
}

func TestConcurrentOpenProjectionWALRebuildsConvergeOnAuthoritativeSnapshot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	for _, content := range [][]byte{[]byte("one"), []byte("two"), []byte("three")} {
		if _, err := store.PutBlob(scalar.SHA256Digest(content), uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
	}
	paths := store.resolvedPathsForTest(t)
	const workers = 6
	start := make(chan struct{})
	errorsByWorker := make(chan error, workers)
	var group sync.WaitGroup
	group.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer group.Done()
			<-start
			projection, _, err := OpenProjection(ctx, paths)
			if err != nil {
				errorsByWorker <- err
				return
			}
			objects, err := projection.Objects(ctx)
			closeErr := projection.Close()
			if err != nil {
				errorsByWorker <- err
				return
			}
			if closeErr != nil {
				errorsByWorker <- closeErr
				return
			}
			if len(objects) != 3 {
				errorsByWorker <- errors.New("concurrent projection observed incomplete snapshot")
			}
		}()
	}
	close(start)
	group.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		t.Errorf("concurrent OpenProjection() error = %v", err)
	}

	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	defer projection.Close()
	objects, err := projection.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 3 {
		t.Fatalf("converged object count = %d, want 3", len(objects))
	}
}

func openRawProjection(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open(projectionDriverName, "file:"+filepath.ToSlash(path)+"?mode=rw")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return db
}

// TestOpenProjectionRecoversPageCorruptionDetectableOnlyByQuickCheck pins the
// production effect of the PRAGMA quick_check gate in verifyProjectionIntegrity.
//
// Every other corruption case in this file is caught strictly earlier:
// verifyProjectionHeader rejects a non-"SQLite format 3" prefix, and
// PingContext raises SQLITE_NOTADB on a header-only file. This case damages a
// b-tree page body past page 1, so the header check and the ping both succeed
// and requireProjectionQuickCheckResult is the only clause that can refuse.
func TestOpenProjectionRecoversPageCorruptionDetectableOnlyByQuickCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	digests := make(map[scalar.Digest][]byte)
	for index := 0; index < 5; index++ {
		content := []byte(fmt.Sprintf("quick check page corruption payload %d", index))
		digest := scalar.SHA256Digest(content)
		if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
		digests[digest] = content
	}
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	const pageSize = 4096
	if len(raw) < 2*pageSize {
		t.Fatalf("projection size = %d bytes, want at least two %d-byte pages to corrupt", len(raw), pageSize)
	}
	// Page 2 body, well past page 1 and past page 2's own b-tree header.
	for offset := pageSize + 8; offset < pageSize+64; offset++ {
		raw[offset] ^= 0xff
	}
	if err := os.WriteFile(indexPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	corrupted := make([]byte, len(raw))
	copy(corrupted, raw)

	// The header gate must not fire: this corruption is invisible to it.
	info, err := os.Lstat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyProjectionHeader(indexPath, info.Size()); err != nil {
		t.Fatalf("verifyProjectionHeader(page-corrupt index) error = %v, want nil so quick_check owns the refusal", err)
	}
	// Naming the production call site: openProjectionDatabase is what
	// openProjection calls, and quick_check is the clause that refuses here.
	opened, openErr := openProjectionDatabase(ctx, indexPath, projectionHooks{})
	if opened != nil {
		_ = opened.db.Close()
	}
	if !errors.Is(openErr, ErrProjectionCorrupt) {
		t.Fatalf("openProjectionDatabase(page-corrupt index) error = %v, want %v", openErr, ErrProjectionCorrupt)
	}
	if !strings.Contains(openErr.Error(), "quick_check:") {
		t.Fatalf("openProjectionDatabase(page-corrupt index) error = %v, want the quick_check refusal clause", openErr)
	}

	restored, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(page-corrupt index) error = %v", err)
	}
	t.Cleanup(func() { _ = restored.Close() })
	if !recovery.RecoveredCorruption {
		t.Fatalf("recovery = %+v, want quick_check corruption quarantined and rebuilt", recovery)
	}
	if recovery.RecoveryDirectory == "" || recovery.ObjectCount != len(digests) {
		t.Fatalf("recovery = %+v, want a quarantine directory and %d rebuilt objects", recovery, len(digests))
	}
	quarantined, err := os.ReadFile(filepath.Join(recovery.RecoveryDirectory, projectionFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(quarantined, corrupted) {
		t.Fatal("quarantined index bytes differ from the corrupt index that was refused")
	}
	objects, err := restored.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != len(digests) {
		t.Fatalf("rebuilt object count = %d, want %d", len(objects), len(digests))
	}
	for _, object := range objects {
		if _, ok := digests[object.Digest]; !ok {
			t.Errorf("rebuilt object %s is not an authoritative blob", object.Digest)
		}
	}
}

// TestOpenProjectionRecoversDefinitionOnlyDriftPreservingInventoryShape pins
// the equalProjectionSchemaEntry definition comparison inside
// verifyProjectionSchema, driven through the production OpenProjection entry.
//
// The neighbouring drift tests cannot reach that clause:
// TestOpenProjectionRecoversDeclaredSchemaDefinitionDrift drops
// UNIQUE(relative_path), so the autoindex disappears and the "missing schema
// objects" branch fires first, and
// TestVerifyProjectionSchemaRejectsKindDriftWithUnchangedNameAndSQL rewrites
// the row type, so the inventory key changes and the "unexpected" branch fires
// on lookup. Both subtests below keep kind, name, tbl_name and the complete
// autoindex set identical, so only the SQL definition comparison can refuse.
func TestOpenProjectionRecoversDefinitionOnlyDriftPreservingInventoryShape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	drifts := []struct {
		name       string
		kind       string
		object     string
		statements []string
	}{
		{
			name:   "declared table loses its uint53 bound and NOT NULL",
			kind:   "table",
			object: "immutable_objects",
			statements: []string{
				"DROP TABLE immutable_objects",
				// Identical kind, name, tbl_name and autoindex set: the
				// WITHOUT ROWID primary key stays the table itself and
				// UNIQUE(relative_path) still yields
				// sqlite_autoindex_immutable_objects_1. Only the CHECK bound
				// and the NOT NULL on size are gone.
				`CREATE TABLE immutable_objects (
					storage_class TEXT NOT NULL,
					representation TEXT NOT NULL,
					digest TEXT NOT NULL,
					size INTEGER,
					relative_path TEXT NOT NULL,
					PRIMARY KEY(storage_class, representation, digest),
					UNIQUE(relative_path)
				) STRICT, WITHOUT ROWID`,
				"CREATE INDEX idx_immutable_objects_digest ON immutable_objects(digest)",
				"CREATE INDEX idx_immutable_objects_storage ON immutable_objects(storage_class, representation, digest)",
			},
		},
		{
			name:   "declared index keeps its name but indexes another column",
			kind:   "index",
			object: "idx_immutable_objects_digest",
			statements: []string{
				"DROP INDEX idx_immutable_objects_digest",
				"CREATE INDEX idx_immutable_objects_digest ON immutable_objects(relative_path)",
			},
		},
	}

	for _, drift := range drifts {
		t.Run(drift.name, func(t *testing.T) {
			ctx := context.Background()
			store := openTestStore(t)
			content := []byte("definition drift authoritative blob")
			digest := scalar.SHA256Digest(content)
			if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
				t.Fatal(err)
			}
			paths := store.resolvedPathsForTest(t)
			projection, _, err := OpenProjection(ctx, paths)
			if err != nil {
				t.Fatal(err)
			}
			indexPath := projection.Path()
			if err := projection.Close(); err != nil {
				t.Fatal(err)
			}

			before := readProjectionSchemaShape(t, ctx, indexPath)
			db := openRawProjection(t, indexPath)
			for _, statement := range drift.statements {
				if _, err := db.ExecContext(ctx, statement); err != nil {
					t.Fatalf("apply drift statement %q: %v", statement, err)
				}
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			after := readProjectionSchemaShape(t, ctx, indexPath)

			// Without this the subtest would silently degrade into one of the
			// already-pinned branches instead of exercising the definition
			// comparison.
			if !slices.Equal(before.keys, after.keys) {
				t.Fatalf("drift changed the sqlite_master inventory shape:\nbefore %v\nafter  %v", before.keys, after.keys)
			}
			driftKey := drift.kind + " " + drift.object
			if before.statements[driftKey] == after.statements[driftKey] {
				t.Fatalf("drift did not change the %s definition", driftKey)
			}
			for key, statement := range before.statements {
				if key == driftKey {
					continue
				}
				if after.statements[key] != statement {
					t.Fatalf("drift changed unrelated object %s definition", key)
				}
			}

			recovered, recovery, err := OpenProjection(ctx, paths)
			if err != nil {
				t.Fatalf("OpenProjection(definition-only drift) error = %v", err)
			}
			t.Cleanup(func() { _ = recovered.Close() })
			if !recovery.RecoveredCorruption || recovery.RecoveryDirectory == "" {
				t.Fatalf("recovery = %+v, want the drifted index quarantined and rebuilt", recovery)
			}
			restored := readProjectionSchemaShape(t, ctx, recovered.Path())
			if restored.statements[driftKey] != before.statements[driftKey] {
				t.Fatalf("rebuilt %s definition = %q, want the declared definition %q",
					driftKey, restored.statements[driftKey], before.statements[driftKey])
			}
			objects, err := recovered.Objects(ctx)
			if err != nil || len(objects) != 1 || objects[0].Digest != digest {
				t.Fatalf("Objects() = %+v, %v, want the authoritative digest %s rebuilt", objects, err, digest)
			}
		})
	}
}

type projectionSchemaShape struct {
	keys         []string
	names        []string
	tables       map[string]string
	statements   map[string]string
	hasStatement map[string]bool
}

// readProjectionSchemaShape reads the raw sqlite_master inventory so a drift
// case can prove it changed only a definition and left kind, name and the
// SQLite-created autoindex set untouched.
//
// keys carry "kind name on table" and therefore move when tbl_name drifts;
// names, tables, statements and hasStatement split the same row into the four
// fields equalProjectionSchemaEntry compares, so a case that targets exactly
// one of those terms can prove the other three still match.
func readProjectionSchemaShape(t *testing.T, ctx context.Context, indexPath string) projectionSchemaShape {
	t.Helper()
	db := openRawProjection(t, indexPath)
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT type, name, tbl_name, sql FROM sqlite_master ORDER BY type, name")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	shape := projectionSchemaShape{
		tables:       make(map[string]string),
		statements:   make(map[string]string),
		hasStatement: make(map[string]bool),
	}
	for rows.Next() {
		var kind, name, table string
		var statement sql.NullString
		if err := rows.Scan(&kind, &name, &table, &statement); err != nil {
			t.Fatal(err)
		}
		shape.keys = append(shape.keys, kind+" "+name+" on "+table)
		shape.names = append(shape.names, kind+" "+name)
		shape.tables[kind+" "+name] = table
		shape.statements[kind+" "+name] = normalizeSchemaSQL(statement.String)
		shape.hasStatement[kind+" "+name] = statement.Valid
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return shape
}

// TestOpenProjectionRecoversAutoindexTableDriftPreservingKeysAndDefinitions
// pins the actual.table != expected.table term of equalProjectionSchemaEntry
// through the production OpenProjection entry.
//
// That term is the only one that can observe this drift, and the fixture below
// asserts the isolation rather than assuming it. A writable_schema UPDATE moves
// the tbl_name of the SQLite-created autoindex from immutable_objects to
// projection_metadata; SQLite accepts the mutation and reopens the database,
// unlike every other sqlite_master rewrite in this file, which fails as a
// malformed schema. Afterwards the (type, name) key set is unchanged, so the
// projectionSchemaObjectKey lookup still resolves the declared entry instead of
// taking the !ok branch; every sql text is unchanged, so the definition
// comparison matches; and sql stays NULL on both sides, so the hasStatement
// term matches too. Only tbl_name moved.
//
// The neighbouring drift tests cannot reach it either:
// TestOpenProjectionRecoversDefinitionOnlyDriftPreservingInventoryShape keeps
// tbl_name identical by construction, and the kind and missing-object cases
// change the key set.
func TestOpenProjectionRecoversAutoindexTableDriftPreservingKeysAndDefinitions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	content := []byte("autoindex table drift authoritative blob")
	digest := scalar.SHA256Digest(content)
	if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	projection, initial, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := projection.Path()
	if err := projection.Close(); err != nil {
		t.Fatal(err)
	}
	if initial.RecoveredCorruption {
		t.Fatalf("initial recovery = %+v, want a clean first open before the drift is planted", initial)
	}

	before := readProjectionSchemaShape(t, ctx, indexPath)

	const driftedTable = "projection_metadata"
	db := openRawProjection(t, indexPath)
	for _, statement := range []string{
		"PRAGMA writable_schema = ON",
		"UPDATE sqlite_master SET tbl_name = '" + driftedTable + "' WHERE type = 'index' AND sql IS NULL",
		"PRAGMA writable_schema = OFF",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("apply drift statement %q: %v", statement, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	after := readProjectionSchemaShape(t, ctx, indexPath)

	// Anti-vacuity guard. Without these the case would silently degrade into
	// one of the already-pinned branches — the !ok lookup, the definition
	// comparison or the hasStatement term — instead of exercising the tbl_name
	// comparison.
	if !slices.Equal(before.names, after.names) {
		t.Fatalf("drift changed the (type, name) key set:\nbefore %v\nafter  %v", before.names, after.names)
	}
	for key, statement := range before.statements {
		if after.statements[key] != statement {
			t.Fatalf("drift changed the %s definition; the SQL comparison, not the tbl_name term, would refuse it", key)
		}
	}
	for key, has := range before.hasStatement {
		if after.hasStatement[key] != has {
			t.Fatalf("drift changed whether %s carries a SQL definition; the hasStatement term, not the tbl_name term, would refuse it", key)
		}
	}
	moved := make([]string, 0, len(before.tables))
	for key, table := range before.tables {
		if after.tables[key] == table {
			continue
		}
		if after.tables[key] != driftedTable {
			t.Fatalf("%s tbl_name = %q, want the drift to move it to %q", key, after.tables[key], driftedTable)
		}
		if before.hasStatement[key] {
			t.Fatalf("%s carries a SQL definition, so moving its tbl_name is observable by the definition comparison too", key)
		}
		moved = append(moved, key)
	}
	if len(moved) == 0 {
		t.Fatal("drift moved no tbl_name, so the tbl_name comparison has nothing to refuse")
	}

	recovered, recovery, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection(autoindex tbl_name drift) error = %v", err)
	}
	t.Cleanup(func() { _ = recovered.Close() })
	if !recovery.RecoveredCorruption || recovery.RecoveryDirectory == "" {
		t.Fatalf("recovery = %+v, want the drifted index quarantined and rebuilt", recovery)
	}
	restored := readProjectionSchemaShape(t, ctx, recovered.Path())
	if !slices.Equal(before.keys, restored.keys) {
		t.Fatalf("rebuilt inventory shape:\nwant %v\ngot  %v", before.keys, restored.keys)
	}
	for _, key := range moved {
		if restored.tables[key] != before.tables[key] {
			t.Fatalf("rebuilt %s tbl_name = %q, want the declared %q", key, restored.tables[key], before.tables[key])
		}
	}
	objects, err := recovered.Objects(ctx)
	if err != nil || len(objects) != 1 || objects[0].Digest != digest {
		t.Fatalf("Objects() = %+v, %v, want the authoritative digest %s rebuilt", objects, err, digest)
	}
}

// TestProjectionSchemaSizeCheckIsDerivedFromMaxBlobSizeInBothDirections drives
// the declared CHECK constraint rather than reading it. The bound is derived
// from MaxBlobSize, the same constant validateProjectionBlobSize enforces on the
// scan path, so the defence-in-depth column constraint cannot drift away from
// the production bound; this proves the derivation produced the real limit and
// pins it at the limit in both directions.
func TestProjectionSchemaSizeCheckIsDerivedFromMaxBlobSizeInBothDirections(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open(projectionDriverName, "file:ax-projection-size-check?mode=memory&cache=private")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, object := range projectionSchemaV1 {
		if _, err := db.ExecContext(ctx, object.sql); err != nil {
			t.Fatalf("create %s %q: %v", object.kind, object.name, err)
		}
	}
	insert := func(digest string, size uint64, path string) error {
		_, err := db.ExecContext(ctx, `
			INSERT INTO immutable_objects(storage_class, representation, digest, size, relative_path)
			VALUES('blob', 'raw', ?, ?, ?)
		`, digest, int64(size), path)
		return err
	}

	if err := insert(strings.Repeat("a", 64), MaxBlobSize, "objects/sha256/at/limit"); err != nil {
		t.Fatalf("insert at MaxBlobSize = %v, want accepted at the limit", err)
	}
	if err := insert(strings.Repeat("b", 64), MaxBlobSize+1, "objects/sha256/past/limit"); err == nil {
		t.Fatalf("insert at MaxBlobSize+1 = nil, want the derived CHECK to refuse past the limit")
	} else if !strings.Contains(err.Error(), "CHECK") {
		t.Fatalf("insert at MaxBlobSize+1 = %v, want the size CHECK constraint to refuse it", err)
	}
	var stored int64
	if err := db.QueryRowContext(ctx, "SELECT size FROM immutable_objects").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if uint64(stored) != MaxBlobSize {
		t.Fatalf("stored size = %d, want exactly MaxBlobSize %d", stored, MaxBlobSize)
	}
}

// projectionFingerprintFixture is the three-blob source set that pins the
// deterministic rebuild. Its digests, sizes and relative paths are written out
// here rather than read back from the production scan, so the expectation is
// independent of both projectionSourceFingerprint and the scan's own ordering.
var projectionFingerprintFixture = []struct {
	content   string
	digestHex string
}{
	{"zeta", "5cc10d9143b2cff082cf5fb373073b13d02d12c9a4d24a97d822d701404fb421"},
	{"alpha", "8ed3f6ad685b959ead7022518e1af76cd816f8e8ec7ccdda1ed4018e8f2223f8"},
	{"middle", "a4888af4e46c129c695ee32775a8c233f113c82e7cd4e6fd3cbb1fda5659f36a"},
}

// projectionFingerprintGolden is the fingerprint of projectionFingerprintFixture
// in ascending digest order, derived outside this module (see
// .temp/TASK-260830-3amrl9/fingerprint-derivation.py) so that copying the
// production value into the test cannot satisfy it. Reversing the scan order
// yields sha256:c17cf49a8881a704934a7ab2f0d9999cbb9b40a3bd54b20076cd1a6c809d80c4
// instead, which is why this constant refuses both a constant fingerprint and a
// re-ordered scan.
const projectionFingerprintGolden = "sha256:daa73c8acff03cbf1d75b9ec94a225309b825d6661e7d0c43b5d57ada749efce"

// expectedProjectionFingerprintObjects rebuilds the fixture as the scan must
// report it: ascending by digest, with the digest_path_v1 shard split spelled
// out locally instead of delegating to digestPathComponents.
func expectedProjectionFingerprintObjects(t *testing.T) []IndexedObject {
	t.Helper()
	expected := make([]IndexedObject, 0, len(projectionFingerprintFixture))
	for _, entry := range projectionFingerprintFixture {
		digest, err := scalar.ParseDigest("sha256:" + entry.digestHex)
		if err != nil {
			t.Fatalf("fixture digest %q is malformed: %v", entry.digestHex, err)
		}
		if got := scalar.SHA256Digest([]byte(entry.content)); got != digest {
			t.Fatalf("fixture content %q hashes to %s, want the pinned %s", entry.content, got, digest)
		}
		expected = append(expected, IndexedObject{
			StorageClass:   "blob",
			Representation: "raw",
			Digest:         digest,
			Size:           uint64(len(entry.content)),
			RelativePath:   "objects/sha256/" + entry.digestHex[:2] + "/" + entry.digestHex[2:],
		})
	}
	for index := 1; index < len(expected); index++ {
		if expected[index-1].Digest.String() >= expected[index].Digest.String() {
			t.Fatalf("fixture is not written in ascending digest order: %+v", expected)
		}
	}
	return expected
}

// independentSourceFingerprint re-derives the fingerprint from the field order
// the schema commits to. It is deliberately a second implementation: it exists
// so a change to projectionSourceFingerprint alone cannot move the expectation
// with it.
func independentSourceFingerprint(objects []IndexedObject) string {
	buffer := &bytes.Buffer{}
	for _, object := range objects {
		fmt.Fprintf(buffer, "%s\x00%s\x00%s\x00%d\x00%s\n",
			object.StorageClass, object.Representation, object.Digest.String(),
			object.Size, object.RelativePath)
	}
	sum := sha256.Sum256(buffer.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestScanAuthoritativeBlobsOrdersSourceObjectsByAscendingDigest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	store := openTestStore(t)
	// Insert in an order that is neither ascending nor descending by digest, so
	// a scan that simply preserves or reverses discovery order cannot pass.
	for _, index := range []int{2, 0, 1} {
		content := []byte(projectionFingerprintFixture[index].content)
		digest := scalar.SHA256Digest(content)
		if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
	}
	paths := store.resolvedPathsForTest(t)
	data, _ := paths.Path(PathData)

	// Asserted against scanAuthoritativeBlobs directly, not through Objects():
	// Objects() carries its own ORDER BY, so reading the order back through it
	// would be decided by the read query rather than by the production scan.
	scanned, err := scanAuthoritativeBlobs(data.Value.String(), projectionHooks{})
	if err != nil {
		t.Fatalf("scanAuthoritativeBlobs() error = %v", err)
	}
	expected := expectedProjectionFingerprintObjects(t)
	if !slices.Equal(scanned, expected) {
		t.Fatalf("scanAuthoritativeBlobs() = %+v, want ascending digest order %+v", scanned, expected)
	}
}

func TestOpenProjectionPinsSourceFingerprintToScannedContentAndOrder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	for _, index := range []int{2, 0, 1} {
		content := []byte(projectionFingerprintFixture[index].content)
		digest := scalar.SHA256Digest(content)
		if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
			t.Fatal(err)
		}
	}

	expected := expectedProjectionFingerprintObjects(t)
	if independent := independentSourceFingerprint(expected); independent != projectionFingerprintGolden {
		t.Fatalf("independent fingerprint of the fixture = %q, want the externally derived golden %q",
			independent, projectionFingerprintGolden)
	}

	projection, _, err := OpenProjection(ctx, store.resolvedPathsForTest(t))
	if err != nil {
		t.Fatalf("OpenProjection() error = %v", err)
	}
	t.Cleanup(func() { _ = projection.Close() })
	metadata, err := projection.Metadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// An exact expected value, not a self-comparison across opens: a constant
	// fingerprint is maximally self-consistent and would satisfy any
	// first-open-equals-second-open check.
	if metadata.SourceFingerprint != projectionFingerprintGolden {
		t.Fatalf("ProjectionMetadata.SourceFingerprint = %q, want %q",
			metadata.SourceFingerprint, projectionFingerprintGolden)
	}
	if metadata.ObjectCount != len(expected) {
		t.Fatalf("ProjectionMetadata.ObjectCount = %d, want %d", metadata.ObjectCount, len(expected))
	}

	// A different source set must move the fingerprint, so the value is proven
	// to depend on the scanned content and not only to be stable.
	other := openTestStore(t)
	otherContent := []byte("alpha")
	otherDigest := scalar.SHA256Digest(otherContent)
	if _, err := other.PutBlob(otherDigest, uint64(len(otherContent)), bytes.NewReader(otherContent)); err != nil {
		t.Fatal(err)
	}
	otherProjection, _, err := OpenProjection(ctx, other.resolvedPathsForTest(t))
	if err != nil {
		t.Fatalf("OpenProjection(other) error = %v", err)
	}
	t.Cleanup(func() { _ = otherProjection.Close() })
	otherMetadata, err := otherProjection.Metadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if otherMetadata.SourceFingerprint == metadata.SourceFingerprint {
		t.Fatalf("a one-object source produced the same fingerprint %q as the three-object source",
			otherMetadata.SourceFingerprint)
	}
	if want := independentSourceFingerprint([]IndexedObject{{
		StorageClass:   "blob",
		Representation: "raw",
		Digest:         otherDigest,
		Size:           uint64(len(otherContent)),
		RelativePath:   "objects/sha256/" + otherDigest.Hex()[:2] + "/" + otherDigest.Hex()[2:],
	}}); otherMetadata.SourceFingerprint != want {
		t.Fatalf("one-object SourceFingerprint = %q, want %q", otherMetadata.SourceFingerprint, want)
	}
}

func TestOpenProjectionConnectionRefusesSchemaRewrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native Windows owner-DACL implementation is a later platform task")
	}

	ctx := context.Background()
	store := openTestStore(t)
	content := []byte("defended")
	digest := scalar.SHA256Digest(content)
	if _, err := store.PutBlob(digest, uint64(len(content)), bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	paths := store.resolvedPathsForTest(t)
	projection, _, err := OpenProjection(ctx, paths)
	if err != nil {
		t.Fatalf("OpenProjection() error = %v", err)
	}
	t.Cleanup(func() { _ = projection.Close() })

	// SQLITE_DBCONFIG_DEFENSIVE pins writable_schema off: the PRAGMA is accepted
	// as a statement but must not take effect on the production connection.
	if _, err := projection.db.ExecContext(ctx, "PRAGMA writable_schema = ON"); err != nil {
		t.Fatalf("PRAGMA writable_schema = ON returned %v, want the statement to be accepted and ignored", err)
	}
	var writableSchema int
	if err := projection.db.QueryRowContext(ctx, "PRAGMA writable_schema").Scan(&writableSchema); err != nil {
		t.Fatal(err)
	}
	if writableSchema != 0 {
		t.Fatalf("writable_schema = %d on the production connection, want 0", writableSchema)
	}
	if _, err := projection.db.ExecContext(ctx, "UPDATE sqlite_master SET sql = sql"); err == nil {
		t.Fatal("UPDATE sqlite_master on the production connection = nil, want the write refused")
	} else if !strings.Contains(err.Error(), "sqlite_master may not be modified") {
		t.Fatalf("UPDATE sqlite_master error = %v, want a defensive-mode refusal", err)
	}

	// Control: the same statements on a connection opened without the defensive
	// parameter do succeed, so the refusal above is the projection's own
	// configuration and not a SQLite baseline every connection would give.
	state, _ := paths.Path(PathState)
	controlPath := filepath.Join(t.TempDir(), "control.sqlite")
	control, err := sql.Open(projectionDriverName, "file:"+filepath.ToSlash(controlPath)+"?mode=rwc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = control.Close() })
	control.SetMaxOpenConns(1)
	if _, err := control.ExecContext(ctx, "CREATE TABLE control_rows(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err := control.ExecContext(ctx, "PRAGMA writable_schema = ON"); err != nil {
		t.Fatal(err)
	}
	var controlWritableSchema int
	if err := control.QueryRowContext(ctx, "PRAGMA writable_schema").Scan(&controlWritableSchema); err != nil {
		t.Fatal(err)
	}
	if controlWritableSchema != 1 {
		t.Fatalf("control writable_schema = %d, want 1; the control no longer demonstrates an undefended connection", controlWritableSchema)
	}
	if _, err := control.ExecContext(ctx, "UPDATE sqlite_master SET sql = sql"); err != nil {
		t.Fatalf("control UPDATE sqlite_master = %v, want an undefended connection to accept it", err)
	}

	// The projection index itself is untouched: the refused write left the
	// on-disk schema exactly as the rebuild wrote it.
	indexPath := filepath.Join(state.Value.String(), projectionFilename)
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatal(err)
	}
	objects, err := projection.Objects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0].Digest != digest {
		t.Fatalf("Objects() after the refused schema write = %+v, want the single authoritative blob", objects)
	}
}

func TestSortIndexedObjectsTotallyOrdersByDigestThenRelativePath(t *testing.T) {
	digestFor := func(hexDigest string) scalar.Digest {
		t.Helper()
		digest, err := scalar.ParseDigest("sha256:" + hexDigest)
		if err != nil {
			t.Fatalf("ParseDigest(%q) error = %v", hexDigest, err)
		}
		return digest
	}
	low := digestFor(strings.Repeat("1", 64))
	high := digestFor(strings.Repeat("9", 64))

	// A same-digest pair exercises the RelativePath tie-break, which the scan's
	// own discovery order can never produce: one digest owns one path there.
	unsorted := []IndexedObject{
		{StorageClass: "blob", Representation: "raw", Digest: high, Size: 2, RelativePath: "objects/sha256/99/b"},
		{StorageClass: "blob", Representation: "raw", Digest: low, Size: 1, RelativePath: "objects/sha256/11/z"},
		{StorageClass: "blob", Representation: "raw", Digest: high, Size: 2, RelativePath: "objects/sha256/99/a"},
		{StorageClass: "blob", Representation: "raw", Digest: low, Size: 1, RelativePath: "objects/sha256/11/a"},
	}
	want := []IndexedObject{unsorted[3], unsorted[1], unsorted[2], unsorted[0]}

	sortIndexedObjects(unsorted)
	if !slices.Equal(unsorted, want) {
		t.Fatalf("sortIndexedObjects() = %+v, want ascending digest then relative path %+v", unsorted, want)
	}

	// Already-ordered input must be left alone, so the order is a fixed point
	// rather than any transformation that merely rearranges.
	stable := slices.Clone(want)
	sortIndexedObjects(stable)
	if !slices.Equal(stable, want) {
		t.Fatalf("sortIndexedObjects() on ordered input = %+v, want unchanged %+v", stable, want)
	}
}
