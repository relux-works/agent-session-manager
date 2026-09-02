package localstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	modernsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	projectionDriverName    = "sqlite"
	projectionFilename      = "index.sqlite"
	projectionSchemaVersion = 1
	projectionBusyTimeoutMS = 5000
)

var (
	ErrProjection                = errors.New("AX SQLite projection failed")
	ErrProjectionSourceIntegrity = errors.New("AX authoritative object source is invalid")
	ErrProjectionCorrupt         = errors.New("AX SQLite projection is corrupt")
	ErrProjectionIncompatible    = errors.New("AX SQLite projection schema is incompatible")
	ErrProjectionMigration       = errors.New("AX SQLite projection migration failed")
	ErrProjectionRebuild         = errors.New("AX SQLite projection rebuild failed")
)

type projectionSchemaObject struct {
	kind string
	name string
	sql  string
}

type projectionSchemaEntry struct {
	kind         string
	name         string
	table        string
	statement    string
	hasStatement bool
}

// projectionSchemaV1 is the complete declared schema inventory. Migration and
// verification both derive their expected set from this one catalog so a new
// table or index cannot be created without becoming part of the integrity gate.
var projectionSchemaV1 = []projectionSchemaObject{
	{
		kind: "table",
		name: "projection_metadata",
		sql: `CREATE TABLE projection_metadata (
			id INTEGER PRIMARY KEY CHECK(id = 1),
			schema_version INTEGER NOT NULL,
			source_fingerprint TEXT NOT NULL,
			object_count INTEGER NOT NULL CHECK(object_count >= 0)
		) STRICT`,
	},
	{
		kind: "table",
		name: "immutable_objects",
		// The stored-size CHECK is derived from MaxBlobSize, the same bound
		// validateProjectionBlobSize enforces on the production scan path, so
		// the defence-in-depth column constraint cannot drift away from it.
		sql: `CREATE TABLE immutable_objects (
			storage_class TEXT NOT NULL,
			representation TEXT NOT NULL,
			digest TEXT NOT NULL,
			size INTEGER NOT NULL CHECK(size >= 0 AND size <= ` + strconv.FormatUint(MaxBlobSize, 10) + `),
			relative_path TEXT NOT NULL,
			PRIMARY KEY(storage_class, representation, digest),
			UNIQUE(relative_path)
		) STRICT, WITHOUT ROWID`,
	},
	{
		kind: "index",
		name: "idx_immutable_objects_digest",
		sql:  "CREATE INDEX idx_immutable_objects_digest ON immutable_objects(digest)",
	},
	{
		kind: "index",
		name: "idx_immutable_objects_storage",
		sql:  "CREATE INDEX idx_immutable_objects_storage ON immutable_objects(storage_class, representation, digest)",
	},
}

type projectionHooks struct {
	beforeMigrationCommit func() error
	beforeRebuildCommit   func() error
	afterBlobStat         func(string) error
	afterRebuildCommit    func(string) error
	// openBlob is the injectable read seam for an authoritative source blob.
	// It exists so the "read did not complete" classification can be driven on
	// this path exactly as it is on the object-store path, instead of being
	// argued from the shared classifier alone.
	openBlob blobOpener
}

// projectionRefusal and projectionOwnershipRefusal are the two refusal funnels
// of the projection. Every refusal decision in a projection-owned production
// source must be raised through one of them, because both record the refusal at
// their immediate call site and the derived refusal inventory in
// projection_refusal_inventory_test.go requires an exercised negative path for
// every such site. A hand-rolled sentinel wrap or an unrouted owner-only guard
// would be invisible to that inventory, so the same gate refuses those shapes.
var projectionRefusal = func(sentinel error, format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, arguments...))
}

// projectionOwnershipRefusal keeps the refusal decision point at its caller
// while preserving the cause raised by a shared owner-only verifier. The
// sentinel selects the domain the caller refuses in; when the cause already
// carries that sentinel the prefix is not repeated.
var projectionOwnershipRefusal = func(sentinel error, cause error, format string, arguments ...any) error {
	subject := fmt.Sprintf(format, arguments...)
	if errors.Is(cause, sentinel) {
		return fmt.Errorf("%s: %w", subject, cause)
	}
	return fmt.Errorf("%w: %s: %w", sentinel, subject, cause)
}

type IndexedObject struct {
	StorageClass   string
	Representation string
	Digest         scalar.Digest
	Size           uint64
	RelativePath   string
}

type ProjectionMetadata struct {
	SchemaVersion     int
	SourceFingerprint string
	ObjectCount       int
}

type ProjectionRecovery struct {
	Rebuilt             bool
	RecoveredCorruption bool
	RecoveryDirectory   string
	ObjectCount         int
}

type Projection struct {
	db   *sql.DB
	path string
}

// OpenProjection is the production state-engine entry point for the currently
// implemented immutable raw-blob store. It verifies source bytes against their
// digest paths before touching the derived index, configures owner-local SQLite
// for WAL/FULL durability, applies schema migrations transactionally, and
// deterministically replaces all projection rows in one transaction. A corrupt
// index is quarantined and rebuilt; corrupt or unreadable authoritative input is
// refused and never laundered into an empty projection.
//
// The authoritative scan runs before the index lock is taken, so two concurrent
// opens can scan different source snapshots and the later committer may write
// the older one. The index is a derived cache with no advertised freshness
// guarantee and the next open converges on the current source, so this is
// deliberate: refusing invalid source bytes must not require holding a
// cross-process lock for the whole verification pass.
func OpenProjection(ctx context.Context, paths ResolvedPaths) (*Projection, ProjectionRecovery, error) {
	return openProjection(ctx, paths, projectionHooks{})
}

func openProjection(ctx context.Context, paths ResolvedPaths, hooks projectionHooks) (*Projection, ProjectionRecovery, error) {
	if ctx == nil {
		return nil, ProjectionRecovery{}, projectionRefusal(ErrProjection, "nil context")
	}
	if err := InitializeLayout(paths); err != nil {
		return nil, ProjectionRecovery{}, fmt.Errorf("%w: initialize local layout: %v", ErrProjection, err)
	}
	data, _ := paths.Path(PathData)
	state, _ := paths.Path(PathState)

	objects, err := scanAuthoritativeBlobs(data.Value.String(), hooks)
	if err != nil {
		return nil, ProjectionRecovery{}, err
	}
	indexPath := filepath.Join(state.Value.String(), projectionFilename)
	lock, err := acquireProjectionLock(indexPath + ".lock")
	if err != nil {
		return nil, ProjectionRecovery{}, fmt.Errorf("%w: acquire index lock: %w", ErrProjection, err)
	}
	defer lock.release()
	projection, openErr := openProjectionDatabase(ctx, indexPath)
	recovery := ProjectionRecovery{Rebuilt: true, ObjectCount: len(objects)}
	if openErr != nil && errors.Is(openErr, ErrProjectionCorrupt) {
		recoveryDirectory, recoveryErr := quarantineProjectionFiles(state.Value.String(), indexPath)
		if recoveryErr != nil {
			return nil, ProjectionRecovery{}, fmt.Errorf("%w: quarantine corrupt index: %v", ErrProjection, recoveryErr)
		}
		recovery.RecoveredCorruption = true
		recovery.RecoveryDirectory = recoveryDirectory
		projection, openErr = openProjectionDatabase(ctx, indexPath)
	}
	if openErr != nil {
		return nil, ProjectionRecovery{}, openErr
	}
	fail := func(err error) (*Projection, ProjectionRecovery, error) {
		_ = projection.db.Close()
		return nil, ProjectionRecovery{}, err
	}

	if err := migrateProjection(ctx, projection.db, hooks); err != nil {
		return fail(err)
	}
	if err := verifyProjectionSchema(ctx, projection.db); err != nil {
		_ = projection.db.Close()
		if !errors.Is(err, ErrProjectionCorrupt) {
			return nil, ProjectionRecovery{}, err
		}
		recoveryDirectory, recoveryErr := quarantineProjectionFiles(state.Value.String(), indexPath)
		if recoveryErr != nil {
			return nil, ProjectionRecovery{}, fmt.Errorf("%w: quarantine invalid schema: %v", ErrProjection, recoveryErr)
		}
		recovery.RecoveredCorruption = true
		recovery.RecoveryDirectory = recoveryDirectory
		projection, openErr = openProjectionDatabase(ctx, indexPath)
		if openErr != nil {
			return nil, ProjectionRecovery{}, openErr
		}
		if err := migrateProjection(ctx, projection.db, hooks); err != nil {
			return fail(err)
		}
	}
	if err := rebuildProjection(ctx, projection.db, objects, hooks); err != nil {
		return fail(err)
	}
	if hooks.afterRebuildCommit != nil {
		if err := hooks.afterRebuildCommit(indexPath); err != nil {
			return fail(fmt.Errorf("%w: after rebuild commit: %v", ErrProjection, err))
		}
	}
	if err := ensureProjectionFileModes(indexPath); err != nil {
		return fail(projectionRefusal(ErrProjection, "verify index files: %v", err))
	}
	return projection, recovery, nil
}

func openProjectionDatabase(ctx context.Context, path string) (*Projection, error) {
	if err := prepareProjectionFile(path); err != nil {
		return nil, fmt.Errorf("%w: prepare index: %w", ErrProjection, err)
	}
	query := url.Values{}
	query.Set("mode", "rw")
	query.Set("_busy_timeout", strconv.Itoa(projectionBusyTimeoutMS))
	query.Set("_journal_mode", "wal")
	// _synchronous=full is defence for a pool reconnect that bypasses
	// configureProjectionConnection; it is currently indistinguishable from
	// SQLite's own default of FULL, so removing it alone is not observable.
	// The durability property is asserted in the direction that matters:
	// setting both this parameter and the post-open PRAGMA to NORMAL reddens
	// TestOpenProjectionProductionEntryConfiguresWALSchemaIndexesAndOwnerOnlyFiles.
	query.Set("_synchronous", "full")
	query.Set("_foreign_keys", "on")
	// _defensive=1 sets SQLITE_DBCONFIG_DEFENSIVE, which is what stops the live
	// projection connection from rewriting sqlite_master the way the drift
	// fixtures do on a raw connection. Unlike _synchronous above it is
	// individually observable, so it is pinned rather than documented: removing
	// or clearing it reddens TestOpenProjectionConnectionRefusesSchemaRewrites.
	query.Set("_defensive", "1")
	dsn := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: query.Encode()}).String()
	db, err := sql.Open(projectionDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: open index: %v", ErrProjection, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	projection := &Projection{db: db, path: path}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, classifySQLiteOpenError("ping index", err)
	}
	if err := configureProjectionConnection(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := verifyProjectionIntegrity(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	// Defence in depth against SQLite's own umask when it creates -wal/-shm:
	// refusing here fails before the rebuild transaction writes any row. Every
	// state that reaches this call also reaches the post-rebuild
	// ensureProjectionFileModes gate in openProjection, which is the subsuming
	// check, so disabling this call alone leaves the suite green by
	// construction rather than for want of a negative path.
	if err := ensureProjectionFileModes(path); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("%w: owner-only index files: %v", ErrProjection, err)
	}
	return projection, nil
}

func prepareProjectionFile(path string) error {
	if err := verifyOwnerDirectory(filepath.Dir(path)); err != nil {
		return projectionOwnershipRefusal(ErrUnsafeOwnership, err, "index state root %q", filepath.Dir(path)) // projection-refusal-direct: production call site is prepareProjectionFile, reached from openProjection; InitializeLayout ensures and verifies the same owner-only state root at the top of openProjection and refuses first, so no on-disk fixture reaches this re-check through the production entry. Pinned by TestPrepareProjectionFileRefusesUnsafeStateRootDirectly
	}
	if err := verifyProjectionSidecars(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return projectionRefusal(ErrUnsafeOwnership, "index is not a regular file")
		}
		if err := verifyOwnerFileInfo(info, 0o600); err != nil {
			return projectionOwnershipRefusal(ErrUnsafeOwnership, err, "index file %q", path)
		}
		if err := verifyProjectionHeader(path, info.Size()); err != nil {
			return err
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return prepareProjectionFile(path)
		}
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func verifyProjectionHeader(path string, size int64) error {
	if size == 0 {
		return nil
	}
	const sqliteHeader = "SQLite format 3\x00"
	if size < int64(len(sqliteHeader)) {
		return projectionRefusal(ErrProjectionCorrupt, "index header is truncated")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	header := make([]byte, len(sqliteHeader))
	_, readErr := io.ReadFull(file, header)
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if string(header) != sqliteHeader {
		return projectionRefusal(ErrProjectionCorrupt, "index header is not SQLite format 3")
	}
	return nil
}

func verifyProjectionSidecars(indexPath string) error {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		filename := indexPath + suffix
		info, err := os.Lstat(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return projectionRefusal(ErrUnsafeOwnership, "projection sidecar %q is not a regular file", filename)
		}
		if err := verifyOwnerFileInfo(info, 0o600); err != nil {
			return projectionOwnershipRefusal(ErrUnsafeOwnership, err, "projection sidecar %q", filename)
		}
	}
	return nil
}

func configureProjectionConnection(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout = "+strconv.Itoa(projectionBusyTimeoutMS)); err != nil {
		return classifySQLiteOpenError("set busy timeout", err)
	}
	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode = WAL").Scan(&journalMode); err != nil {
		return classifySQLiteOpenError("enable WAL", err)
	}
	if err := requireProjectionJournalMode(journalMode); err != nil {
		return err
	}
	// Reasserted post-open rather than trusted from the DSN. Removing either
	// this statement or the DSN parameter alone is not observable, because
	// SQLite's own default is already FULL; the property is pinned in the
	// direction that matters by weakening both to NORMAL, which reddens the
	// synchronous assertion in
	// TestOpenProjectionProductionEntryConfiguresWALSchemaIndexesAndOwnerOnlyFiles.
	if _, err := db.ExecContext(ctx, "PRAGMA synchronous = FULL"); err != nil {
		return classifySQLiteOpenError("enable FULL synchronous mode", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return classifySQLiteOpenError("enable foreign keys", err)
	}
	return nil
}

func verifyProjectionIntegrity(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "PRAGMA quick_check")
	if err != nil {
		return classifySQLiteOpenError("quick_check", err)
	}
	defer rows.Close()
	ok := false
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return fmt.Errorf("%w: read quick_check: %v", ErrProjectionCorrupt, err)
		}
		if err := requireProjectionQuickCheckResult(result); err != nil {
			return err
		}
		ok = true
	}
	if err := rows.Err(); err != nil {
		return classifySQLiteOpenError("iterate quick_check", err)
	}
	if !ok {
		return projectionRefusal(ErrProjectionCorrupt, "quick_check returned no result") // projection-refusal-subsumed: SQLite PRAGMA quick_check returns at least one row or a query error; requireProjectionQuickCheckResult owns every returned non-ok value
	}
	return nil
}

func requireProjectionJournalMode(mode string) error {
	if mode != "wal" {
		return projectionRefusal(ErrProjection, "journal_mode is %q, want wal", mode) // projection-refusal-direct: production call site is configureProjectionConnection, reached from openProjectionDatabase; SQLite converts every file-backed journal mode to WAL on request, so no index file openProjection creates can reach this clause. Pinned by TestProjectionPreviouslyUnpinnedRefusalClauses/journal_mode_value_class over the real fallback modes SQLite reports, plus the production configureProjectionConnection memory case
	}
	return nil
}

func requireProjectionQuickCheckResult(result string) error {
	if result != "ok" {
		return projectionRefusal(ErrProjectionCorrupt, "quick_check: %s", result)
	}
	return nil
}

func classifySQLiteOpenError(operation string, err error) error {
	if isSQLiteCorruption(err) {
		return fmt.Errorf("%w: %s: %v", ErrProjectionCorrupt, operation, err)
	}
	return fmt.Errorf("%w: %s: %v", ErrProjection, operation, err)
}

func isSQLiteCorruption(err error) bool {
	var sqliteErr *modernsqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code() & 0xff
	return code == sqlite3.SQLITE_CORRUPT || code == sqlite3.SQLITE_NOTADB
}

func migrateProjection(ctx context.Context, db *sql.DB, hooks projectionHooks) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin: %v", ErrProjectionMigration, err)
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("%w: read schema version: %v", ErrProjectionMigration, err)
	}
	if version > projectionSchemaVersion {
		return projectionRefusal(ErrProjectionIncompatible, "index schema version %d is newer than supported %d", version, projectionSchemaVersion)
	}
	if version == projectionSchemaVersion {
		return tx.Commit()
	}
	for _, object := range projectionSchemaV1 {
		if _, err := tx.ExecContext(ctx, object.sql); err != nil {
			return fmt.Errorf("%w: create %s %q: %v", ErrProjectionMigration, object.kind, object.name, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(projectionSchemaVersion)); err != nil {
		return fmt.Errorf("%w: set schema version: %v", ErrProjectionMigration, err)
	}
	if hooks.beforeMigrationCommit != nil {
		if err := hooks.beforeMigrationCommit(); err != nil {
			return fmt.Errorf("%w: before commit: %w", ErrProjectionMigration, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit: %v", ErrProjectionMigration, err)
	}
	return nil
}

func verifyProjectionSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return classifySQLiteOpenError("read schema version", err)
	}
	if version > projectionSchemaVersion {
		return projectionRefusal(ErrProjectionIncompatible, "index schema version %d is newer than supported %d", version, projectionSchemaVersion) // projection-refusal-subsumed: migrateProjection owns the production newer-version refusal before schema verification
	}
	if version != projectionSchemaVersion {
		return projectionRefusal(ErrProjectionCorrupt, "index schema version %d, want %d", version, projectionSchemaVersion) // projection-refusal-direct: production call site is verifyProjectionSchema, reached from openProjection; migrateProjection always normalizes user_version to projectionSchemaVersion immediately before it, so this defends only against an out-of-band writer changing the version between the two statements. Pinned by TestProjectionPreviouslyUnpinnedRefusalClauses/schema_verifier_noncurrent_version
	}

	expected, err := deriveProjectionSchemaInventory(ctx)
	if err != nil {
		return fmt.Errorf("%w: derive declared schema inventory: %v", ErrProjection, err)
	}
	rows, err := db.QueryContext(ctx, `
			SELECT type, name, tbl_name, sql
			FROM sqlite_master
			ORDER BY type, name
		`)
	if err != nil {
		return classifySQLiteOpenError("read schema inventory", err)
	}
	defer rows.Close()
	seen := make(map[string]struct{}, len(expected))
	for rows.Next() {
		var kind, name, table string
		var statement sql.NullString
		if err := rows.Scan(&kind, &name, &table, &statement); err != nil {
			return fmt.Errorf("%w: scan schema inventory: %v", ErrProjectionCorrupt, err)
		}
		key := projectionSchemaObjectKey(kind, name)
		want, ok := expected[key]
		got := projectionSchemaEntry{
			kind: kind, name: name, table: table,
			statement: statement.String, hasStatement: statement.Valid,
		}
		if !ok || !equalProjectionSchemaEntry(got, want) {
			return projectionRefusal(ErrProjectionCorrupt, "unexpected or drifted %s %q", kind, name)
		}
		seen[key] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return classifySQLiteOpenError("iterate schema inventory", err)
	}
	if len(seen) != len(expected) {
		missing := make([]string, 0, len(expected)-len(seen))
		for key, object := range expected {
			if _, ok := seen[key]; !ok {
				missing = append(missing, object.kind+" "+object.name)
			}
		}
		sort.Strings(missing)
		return projectionRefusal(ErrProjectionCorrupt, "missing schema objects %v", missing)
	}
	return nil
}

func deriveProjectionSchemaInventory(ctx context.Context) (map[string]projectionSchemaEntry, error) {
	db, err := sql.Open(projectionDriverName, "file:ax-projection-schema-v1?mode=memory&cache=private")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, object := range projectionSchemaV1 {
		if _, err := db.ExecContext(ctx, object.sql); err != nil {
			return nil, fmt.Errorf("create catalog %s %q: %v", object.kind, object.name, err)
		}
	}
	rows, err := db.QueryContext(ctx, `
		SELECT type, name, tbl_name, sql
		FROM sqlite_master
		ORDER BY type, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	inventory := make(map[string]projectionSchemaEntry)
	for rows.Next() {
		var entry projectionSchemaEntry
		var statement sql.NullString
		if err := rows.Scan(&entry.kind, &entry.name, &entry.table, &statement); err != nil {
			return nil, err
		}
		entry.statement = statement.String
		entry.hasStatement = statement.Valid
		inventory[projectionSchemaObjectKey(entry.kind, entry.name)] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return inventory, nil
}

// equalProjectionSchemaEntry compares one observed sqlite_master row against
// the declared catalog entry. Only two of its four identity terms carry the
// refusal on their own; the other two are stated here so a later sweep does not
// have to re-derive why weakening them leaves the suite green.
//
//   - actual.table is load-bearing and reachable: SQLite accepts a
//     writable_schema move of an autoindex row's tbl_name to another declared
//     table and still opens the database, leaving the (type, name) key set and
//     every SQL text untouched. Pinned through OpenProjection by
//     TestOpenProjectionRecoversAutoindexTableDriftPreservingKeysAndDefinitions.
//   - actual.kind and actual.name are dead with respect to the refusal, and
//     subsumed by the projectionSchemaObjectKey lookup in verifyProjectionSchema:
//     the key is kind + "\x00" + name, so drift in either field misses the map
//     and takes the !ok branch before this comparison decides anything.
//   - actual.hasStatement is subsumed by SQLite's own schema parser. Both
//     reachable drift directions make the database unopenable: nulling a
//     declared index's sql yields "malformed database schema
//     (idx_immutable_objects_digest) - orphan index", and materialising sql on
//     the autoindex yields "... sqlite_autoindex_immutable_objects_2 already
//     exists". Both surface as SQLITE_CORRUPT through classifySQLiteOpenError,
//     which routes to the quarantine-and-rebuild path pinned by
//     TestOpenProjectionRecoversCorruptIndexFromAuthoritativeObjects.
func equalProjectionSchemaEntry(actual, expected projectionSchemaEntry) bool {
	if actual.kind != expected.kind || actual.name != expected.name || actual.table != expected.table || actual.hasStatement != expected.hasStatement {
		return false
	}
	if !actual.hasStatement {
		return true
	}
	return normalizeSchemaSQL(actual.statement) == normalizeSchemaSQL(expected.statement)
}

func projectionSchemaObjectKey(kind, name string) string {
	return kind + "\x00" + name
}

func normalizeSchemaSQL(statement string) string {
	return strings.Join(strings.Fields(strings.TrimSuffix(strings.TrimSpace(statement), ";")), " ")
}

func rebuildProjection(ctx context.Context, db *sql.DB, objects []IndexedObject, hooks projectionHooks) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin: %v", ErrProjectionRebuild, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM immutable_objects"); err != nil {
		return fmt.Errorf("%w: clear objects: %v", ErrProjectionRebuild, err)
	}
	insert, err := tx.PrepareContext(ctx, `
		INSERT INTO immutable_objects(storage_class, representation, digest, size, relative_path)
		VALUES(?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("%w: prepare object insert: %v", ErrProjectionRebuild, err)
	}
	for _, object := range objects {
		if _, err := insert.ExecContext(ctx, object.StorageClass, object.Representation, object.Digest.String(), object.Size, object.RelativePath); err != nil {
			_ = insert.Close()
			return fmt.Errorf("%w: insert %s: %v", ErrProjectionRebuild, object.Digest, err)
		}
	}
	if err := insert.Close(); err != nil {
		return fmt.Errorf("%w: close object insert: %v", ErrProjectionRebuild, err)
	}
	fingerprint := projectionSourceFingerprint(objects)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO projection_metadata(id, schema_version, source_fingerprint, object_count)
		VALUES(1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			schema_version = excluded.schema_version,
			source_fingerprint = excluded.source_fingerprint,
			object_count = excluded.object_count
	`, projectionSchemaVersion, fingerprint, len(objects)); err != nil {
		return fmt.Errorf("%w: write metadata: %v", ErrProjectionRebuild, err)
	}
	if hooks.beforeRebuildCommit != nil {
		if err := hooks.beforeRebuildCommit(); err != nil {
			return fmt.Errorf("%w: before commit: %w", ErrProjectionRebuild, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit: %v", ErrProjectionRebuild, err)
	}
	return nil
}

func projectionSourceFingerprint(objects []IndexedObject) string {
	hasher := sha256.New()
	for _, object := range objects {
		_, _ = io.WriteString(hasher, object.StorageClass)
		_, _ = hasher.Write([]byte{0})
		_, _ = io.WriteString(hasher, object.Representation)
		_, _ = hasher.Write([]byte{0})
		_, _ = io.WriteString(hasher, object.Digest.String())
		_, _ = hasher.Write([]byte{0})
		_, _ = io.WriteString(hasher, strconv.FormatUint(object.Size, 10))
		_, _ = hasher.Write([]byte{0})
		_, _ = io.WriteString(hasher, object.RelativePath)
		_, _ = hasher.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

func scanAuthoritativeBlobs(dataRoot string, hooks projectionHooks) ([]IndexedObject, error) {
	objectsRoot := filepath.Join(dataRoot, "objects")
	digestRoot := filepath.Join(objectsRoot, "sha256")
	if _, err := os.Lstat(objectsRoot); errors.Is(err, os.ErrNotExist) {
		return []IndexedObject{}, nil
	} else if err != nil {
		return nil, sourceFailure("lstat objects root", err)
	}
	if err := verifyOwnerDirectory(objectsRoot); err != nil {
		return nil, projectionOwnershipRefusal(ErrProjectionSourceIntegrity, err, "verify objects root")
	}
	objectMembers, err := os.ReadDir(objectsRoot)
	if err != nil {
		return nil, sourceFailure("read objects root", err)
	}
	allowedMember := filepath.Base(digestRoot)
	for _, member := range objectMembers {
		if member.Name() != allowedMember {
			return nil, projectionRefusal(ErrProjectionSourceIntegrity, "invalid objects root member %q", member.Name())
		}
	}
	if _, err := os.Lstat(digestRoot); errors.Is(err, os.ErrNotExist) {
		return []IndexedObject{}, nil
	} else if err != nil {
		return nil, sourceFailure("lstat blob digest root", err)
	}
	if err := verifyOwnerDirectory(digestRoot); err != nil {
		return nil, projectionOwnershipRefusal(ErrProjectionSourceIntegrity, err, "verify blob digest root")
	}
	shards, err := os.ReadDir(digestRoot)
	if err != nil {
		return nil, sourceFailure("read blob digest root", err)
	}
	openBlob := hooks.openBlob
	if openBlob == nil {
		openBlob = openBlobFile
	}
	objects := make([]IndexedObject, 0)
	for _, shard := range shards {
		if !shard.IsDir() {
			return nil, projectionRefusal(ErrProjectionSourceIntegrity, "blob shard %q is not a directory", shard.Name())
		}
		if !isLowerHex(shard.Name(), 2) {
			return nil, projectionRefusal(ErrProjectionSourceIntegrity, "invalid blob shard %q", shard.Name())
		}
		shardPath := filepath.Join(digestRoot, shard.Name())
		if err := verifyOwnerDirectory(shardPath); err != nil {
			return nil, projectionOwnershipRefusal(ErrProjectionSourceIntegrity, err, "verify blob shard %q", shardPath)
		}
		entries, err := os.ReadDir(shardPath)
		if err != nil {
			return nil, sourceFailure("read blob shard", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), stagedFilePrefix) {
				stagePath := filepath.Join(shardPath, entry.Name())
				info, err := os.Lstat(stagePath)
				if err != nil {
					return nil, sourceFailure("lstat staged blob", err)
				}
				if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
					return nil, projectionRefusal(ErrProjectionSourceIntegrity, "staged blob %q is not a regular file", stagePath)
				}
				if err := verifyOwnerFileInfo(info, 0o600); err != nil {
					return nil, projectionOwnershipRefusal(ErrProjectionSourceIntegrity, err, "verify staged blob %q ownership", stagePath)
				}
				continue
			}
			if entry.IsDir() {
				return nil, projectionRefusal(ErrProjectionSourceIntegrity, "blob leaf %q is a directory", entry.Name())
			}
			filename := filepath.Join(shardPath, entry.Name())
			info, err := os.Lstat(filename)
			if err != nil {
				return nil, sourceFailure("lstat blob", err)
			}
			if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return nil, projectionRefusal(ErrProjectionSourceIntegrity, "blob %q is not a regular file", filename)
			}
			if err := verifyOwnerFileInfo(info, 0o600); err != nil {
				return nil, projectionOwnershipRefusal(ErrProjectionSourceIntegrity, err, "verify blob %q ownership", filename)
			}
			if err := validateProjectionBlobSize(info.Size()); err != nil {
				return nil, err
			}
			if hooks.afterBlobStat != nil {
				if err := hooks.afterBlobStat(filename); err != nil {
					return nil, sourceFailure("after blob stat", err)
				}
			}
			// scalar.ParseDigest owns the complete 64-character length and
			// lowercase-hex alphabet. The exact two-character shard check above
			// independently owns only the digest_path_v1 split position.
			digest, err := scalar.ParseDigest("sha256:" + shard.Name() + entry.Name())
			if err != nil {
				return nil, sourceFailure("parse blob path digest", err)
			}
			// verifyBlobContent is the classifier the object store's
			// digest-path inspection also uses. Routing both through it is what
			// keeps the two paths from disagreeing about whether a failed read
			// is an integrity finding: blobUnreadable proves nothing, so this
			// scan reports a source failure rather than a refusal decision
			// about the bytes, and the store moves nothing.
			inspection := verifyBlobContent(openBlob, filename, digest, uint64(info.Size()))
			if inspection.verdict == blobUnreadable {
				return nil, sourceFailure("inspect blob", inspection.err)
			}
			size := int64(inspection.size)
			if size != info.Size() {
				return nil, projectionRefusal(ErrProjectionSourceIntegrity, "blob %s changed size from %d to %d while reading", digest, info.Size(), size)
			}
			if inspection.digest != digest {
				return nil, projectionRefusal(ErrProjectionSourceIntegrity, "blob path %s contains %s", digest, inspection.digest)
			}
			relative, err := filepath.Rel(dataRoot, filename)
			if err != nil {
				return nil, sourceFailure("derive blob relative path", err)
			}
			objects = append(objects, IndexedObject{
				StorageClass: "blob", Representation: "raw", Digest: digest,
				Size: uint64(size), RelativePath: filepath.ToSlash(relative),
			})
		}
	}
	// Removing this call alone is not observable: os.ReadDir returns entries
	// sorted by filename, and the digest_path_v1 split makes that traversal
	// already ascending by digest for the single blob/raw class this scan
	// produces. It is normalization defence for a future source that does not
	// arrive in path order. The ordering property is asserted in the direction
	// that matters: reversing the comparator reddens
	// TestScanAuthoritativeBlobsOrdersSourceObjectsByAscendingDigest, and the
	// total order itself, including the RelativePath tie-break, is pinned
	// directly on sortIndexedObjects by
	// TestSortIndexedObjectsTotallyOrdersByDigestThenRelativePath.
	sortIndexedObjects(objects)
	return objects, nil
}

// sortIndexedObjects imposes the deterministic rebuild order: ascending by
// digest, then by relative path. It is the single owner of that order, so the
// comparator can be driven directly instead of only through a scan whose
// discovery order already happens to match it.
func sortIndexedObjects(objects []IndexedObject) {
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].Digest != objects[j].Digest {
			return objects[i].Digest.String() < objects[j].Digest.String()
		}
		return objects[i].RelativePath < objects[j].RelativePath
	})
}

func validateProjectionBlobSize(size int64) error {
	if size < 0 {
		return projectionRefusal(ErrProjectionSourceIntegrity, "blob size %d is negative", size) // projection-refusal-direct: production call site is scanAuthoritativeBlobs, reached from openProjection; the argument is os.FileInfo.Size(), which the kernel never reports negative, so no on-disk fixture can drive it. Pinned by TestProjectionPreviouslyUnpinnedRefusalClauses/blob_size_uint53_bounds
	}
	if uint64(size) > MaxBlobSize {
		return projectionRefusal(ErrProjectionSourceIntegrity, "blob size %d exceeds uint53", size) // projection-refusal-direct: production call site is scanAuthoritativeBlobs, reached from openProjection; driving it needs a blob larger than 9007199254740991 bytes, which no test fixture can materialize. Pinned at the limit in both directions by TestProjectionPreviouslyUnpinnedRefusalClauses/blob_size_uint53_bounds
	}
	return nil
}

func sourceFailure(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrProjectionSourceIntegrity, operation, err)
}

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	return isLowerHexCharacters(value)
}

func isLowerHexCharacters(value string) bool {
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func quarantineProjectionFiles(stateRoot, indexPath string) (string, error) {
	identifier, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	if err := ensureOwnerChildTree(stateRoot, "index-recovery", identifier); err != nil {
		return "", err
	}
	directory := filepath.Join(stateRoot, "index-recovery", identifier)
	moved := false
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		source := indexPath + suffix
		info, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", projectionRefusal(ErrUnsafeOwnership, "index recovery source %q is not a regular file", source) // projection-refusal-direct: production call site is quarantineProjectionFiles, reached from openProjection; every quarantined path was proven regular and owner-only by prepareProjectionFile, verifyProjectionSidecars or ensureProjectionFileModes immediately before, under the held index lock, so this TOCTOU re-check has no reachable on-disk fixture. Pinned by TestQuarantineProjectionFilesRefusesUnsafeRecoverySources
		}
		if err := verifyOwnerFileInfo(info, 0o600); err != nil {
			return "", projectionOwnershipRefusal(ErrUnsafeOwnership, err, "index recovery source %q", source) // projection-refusal-direct: production call site is quarantineProjectionFiles, reached from openProjection; same TOCTOU re-check as the regularity clause above. Pinned by TestQuarantineProjectionFilesRefusesUnsafeRecoverySources
		}
		destination := filepath.Join(directory, filepath.Base(source))
		if err := atomicRenameNoReplace(source, destination); err != nil {
			return "", err
		}
		moved = true
	}
	if !moved {
		return "", fmt.Errorf("no corrupt index files found")
	}
	if err := syncDirectory(directory); err != nil {
		return "", err
	}
	if err := syncDirectory(stateRoot); err != nil {
		return "", err
	}
	return directory, nil
}

func ensureProjectionFileModes(indexPath string) error {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		filename := indexPath + suffix
		info, err := os.Lstat(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return projectionRefusal(ErrUnsafeOwnership, "%q is not a regular file", filename)
		}
		if err := verifyOwnerFileInfo(info, 0o600); err != nil {
			return projectionOwnershipRefusal(ErrUnsafeOwnership, err, "%q", filename)
		}
	}
	return nil
}

func (projection *Projection) Path() string {
	if projection == nil {
		return ""
	}
	return projection.path
}

func (projection *Projection) Objects(ctx context.Context) ([]IndexedObject, error) {
	if projection == nil || projection.db == nil {
		return nil, fmt.Errorf("%w: projection is not open", ErrProjection)
	}
	rows, err := projection.db.QueryContext(ctx, `
		SELECT storage_class, representation, digest, size, relative_path
		FROM immutable_objects
		ORDER BY storage_class, representation, digest
	`)
	if err != nil {
		return nil, fmt.Errorf("%w: query objects: %v", ErrProjection, err)
	}
	defer rows.Close()
	objects := make([]IndexedObject, 0)
	for rows.Next() {
		var storageClass, representation, digestText, relativePath string
		var size int64
		if err := rows.Scan(&storageClass, &representation, &digestText, &size, &relativePath); err != nil {
			return nil, fmt.Errorf("%w: scan object: %v", ErrProjection, err)
		}
		digest, err := scalar.ParseDigest(digestText)
		// The two size terms are defence in depth and are subsumed on the
		// production read path: verifyProjectionSchema runs before any Objects
		// call and refuses a schema whose immutable_objects definition drifts
		// from the catalog, and that catalog carries
		// CHECK(size >= 0 AND size <= MaxBlobSize) derived from the same
		// constant. The CHECK is pinned in both directions by
		// TestProjectionSchemaSizeCheckIsDerivedFromMaxBlobSizeInBothDirections,
		// so a stored row outside the bound cannot reach here without the
		// schema comparison refusing first. Only the digest term carries the
		// refusal on its own.
		if err != nil || size < 0 || uint64(size) > MaxBlobSize {
			return nil, fmt.Errorf("%w: invalid indexed object", ErrProjectionCorrupt)
		}
		objects = append(objects, IndexedObject{
			StorageClass: storageClass, Representation: representation, Digest: digest,
			Size: uint64(size), RelativePath: relativePath,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate objects: %v", ErrProjection, err)
	}
	return objects, nil
}

func (projection *Projection) Metadata(ctx context.Context) (ProjectionMetadata, error) {
	if projection == nil || projection.db == nil {
		return ProjectionMetadata{}, fmt.Errorf("%w: projection is not open", ErrProjection)
	}
	var metadata ProjectionMetadata
	if err := projection.db.QueryRowContext(ctx, `
		SELECT schema_version, source_fingerprint, object_count
		FROM projection_metadata WHERE id = 1
	`).Scan(&metadata.SchemaVersion, &metadata.SourceFingerprint, &metadata.ObjectCount); err != nil {
		return ProjectionMetadata{}, fmt.Errorf("%w: read metadata: %v", ErrProjection, err)
	}
	return metadata, nil
}

func (projection *Projection) Close() error {
	if projection == nil || projection.db == nil {
		return nil
	}
	db := projection.db
	projection.db = nil
	_, checkpointErr := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	closeErr := db.Close()
	return errors.Join(checkpointErr, closeErr)
}
