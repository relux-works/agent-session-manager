package merkleinventory

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessquery"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

var (
	ErrDurableStore   = errors.New("durable inventory operation failed")
	ErrDurableCorrupt = errors.New("durable inventory contains invalid state")
)

var durableJSONNamespaces = []string{"event", "manifest", "record", "tombstone", "tombstone_ack"}

// DurableIndex persists validated JSON objects under their semantic identity.
// Its Merkle Index is rebuilt from those objects on every open.
type DurableIndex struct {
	root  string
	index *Index
	mu    sync.Mutex
	ops   durableOperations
}

type durableOperations struct {
	afterObjectInstall       func(string)
	afterQuarantineCandidate func(string)
	afterActiveRemoval       func(string)
}

// OpenDurable creates or loads an owner-local immutable JSON union. RPC 2 raw
// blobs and chunks remain owned by the Section 11.5 transfer and staging path.
func OpenDurable(root string) (*DurableIndex, error) {
	if root == "" {
		return nil, fmt.Errorf("%w: empty storage root", ErrDurableStore)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve storage root: %v", ErrDurableStore, err)
	}
	for _, path := range []string{abs, filepath.Join(abs, "objects"), filepath.Join(abs, "quarantine")} {
		if err := ensureDurableDirectory(path); err != nil {
			return nil, err
		}
	}
	index, err := New(RPC2Version)
	if err != nil {
		return nil, err
	}
	store := &DurableIndex{root: abs, index: index}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// AddJSON validates schema and canonical identity before its atomic durable
// install. Replaying identical bytes is idempotent. A byte conflict is
// persisted in quarantine, removed from the active inventory, and refused.
// Tombstone/Acknowledgement links are checked after the complete union arrives
// so either immutable record may arrive first.
func (store *DurableIndex) AddJSON(data []byte) error {
	if store == nil || store.index == nil {
		return fmt.Errorf("%w: store is not open", ErrDurableStore)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.addJSONLocked(data)
}

func (store *DurableIndex) addJSONLocked(data []byte) error {
	membership, err := ClassifyJSON(data)
	if err != nil {
		return err
	}
	if membership.Excluded {
		return ErrExcluded
	}
	if !slices.Contains(durableJSONNamespaces, membership.Namespace) {
		return ErrInvalidRequest
	}
	id := membership.ID.String()
	store.index.mu.RLock()
	priorNamespace, exists := store.index.locations[id]
	prior, hasPrior := store.index.objects[priorNamespace][id]
	_, quarantined := store.index.quarantine[id]
	store.index.mu.RUnlock()
	if quarantined {
		if err := store.persistQuarantineCandidate(membership.Namespace, id, data); err != nil {
			return err
		}
		return store.index.AddUnionJSON(data)
	}
	if exists {
		if priorNamespace == membership.Namespace && hasPrior && bytes.Equal(prior.data, data) {
			return nil
		}
		if hasPrior {
			if err := store.persistQuarantineCandidate(membership.Namespace, id, data); err != nil {
				return err
			}
			if err := store.persistQuarantineCandidate(priorNamespace, id, prior.data); err != nil {
				return err
			}
		}
		if err := store.removeActive(priorNamespace, id); err != nil {
			return err
		}
		return store.index.AddUnionJSON(data)
	}

	path, err := store.objectPath(membership.Namespace, id)
	if err != nil {
		return err
	}
	if err := ensureDurableDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	installed, existing, err := durableInstallNoReplace(path, data)
	if err != nil {
		return err
	}
	if existing != nil && !bytes.Equal(existing, data) {
		other, classifyErr := ClassifyJSON(existing)
		if classifyErr != nil || other.ID.String() != id || other.Namespace != membership.Namespace {
			return fmt.Errorf("%w: occupied semantic identity path %s does not validate", ErrDurableCorrupt, path)
		}
		if err := store.persistQuarantineCandidate(membership.Namespace, id, data); err != nil {
			return err
		}
		if err := store.persistQuarantineCandidate(membership.Namespace, id, existing); err != nil {
			return err
		}
		if err := store.removeActive(membership.Namespace, id); err != nil {
			return err
		}
		store.index.mu.Lock()
		store.index.quarantine[id] = append(store.index.quarantine[id], bytes.Clone(existing))
		store.index.mu.Unlock()
		return store.index.AddUnionJSON(data)
	}
	if installed && store.ops.afterObjectInstall != nil {
		store.ops.afterObjectInstall(id)
	}
	if err := store.index.AddUnionJSON(data); err != nil {
		return fmt.Errorf("%w: inventory refused an installed validated object: %v", ErrDurableStore, err)
	}
	return nil
}

// SyncFrom audits common identity bytes and transfers missing JSON objects by
// digest. Merkle roots contain identities only, so common IDs must also be
// fetched to detect same-digest/different-byte conflicts.
func (store *DurableIndex) SyncFrom(peer *DurableIndex, nextRequestID RequestIDFactory) error {
	if store == nil || peer == nil || nextRequestID == nil {
		return ErrInvalidRequest
	}
	if store == peer {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, namespace := range durableJSONNamespaces {
		peerIDs := peer.index.objectIDs(namespace)
		localIDs := store.index.objectIDs(namespace)
		localSet := make(map[string]struct{}, len(localIDs))
		for _, id := range localIDs {
			localSet[id] = struct{}{}
		}
		common := make([]string, 0)
		for _, id := range peerIDs {
			if _, exists := localSet[id]; exists {
				common = append(common, id)
			}
		}
		if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}
		missing, err := MissingObjectIDs(store.index, peer.index, namespace)
		if err != nil {
			return err
		}
		if err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil {
			return err
		}
	}
	return store.validateTombstoneAckClosureLocked()
}

func (store *DurableIndex) fetchAndAdd(peer *DurableIndex, namespace string, ids []string, nextRequestID RequestIDFactory) error {
	for start := 0; start < len(ids); start += 4096 {
		end := min(start+4096, len(ids))
		objects, err := FetchObjects(peer, namespace, ids[start:end], rpcwire.MaxLineBytes, nextRequestID)
		if err != nil {
			return err
		}
		for _, object := range objects {
			data, err := base64.RawURLEncoding.Strict().DecodeString(object.Data)
			if err != nil || base64.RawURLEncoding.EncodeToString(data) != object.Data {
				return ErrInvalidObject
			}
			if err := store.addJSONLocked(data); err != nil {
				return err
			}
		}
	}
	return nil
}

// Root returns a Merkle root over the current validated union.
func (store *DurableIndex) Root(namespace string) (Root, error) {
	if store == nil || store.index == nil {
		return Root{}, ErrInvalidRequest
	}
	return store.index.Root(namespace)
}

// Object returns exact validated bytes addressed by namespace and digest.
func (store *DurableIndex) Object(namespace, id string) ([]byte, error) {
	if store == nil || store.index == nil || !slices.Contains(durableJSONNamespaces, namespace) {
		return nil, ErrInvalidRequest
	}
	digest, err := scalar.ParseDigest(id)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	store.index.mu.RLock()
	defer store.index.mu.RUnlock()
	if _, quarantined := store.index.quarantine[digest.String()]; quarantined {
		return nil, refusal("integrity_failure", ErrIntegrity)
	}
	if current, ok := store.index.objects[namespace][digest.String()]; ok {
		return bytes.Clone(current.data), nil
	}
	return nil, ErrNotFound
}

// ObjectsGet implements the read-only object-source boundary for FetchObjects.
func (store *DurableIndex) ObjectsGet(line []byte, namespace string, maxLineBytes uint64) ([]byte, error) {
	if store == nil || store.index == nil {
		return nil, ErrInvalidRequest
	}
	return store.index.ObjectsGet(line, namespace, maxLineBytes)
}

// ValidateUnionClosure verifies cross-object Tombstone/Acknowledgement links
// after exchange. Union itself retains records and performs no action.
func (store *DurableIndex) ValidateUnionClosure() error {
	if store == nil || store.index == nil {
		return ErrInvalidRequest
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.validateTombstoneAckClosureLocked()
}

// RebuildProjection verifies that the authoritative repository record and
// chained events are present byte-for-byte in this union, derives all
// validated lease heads only after that check, and delegates the pure fold to
// sessstate.Projector. Divergent event objects remain stored in the union but
// do not enter the repository's authoritative chain or the reducer input.
func (store *DurableIndex) RebuildProjection(repo *sessrepo.Repository, sessionID string) (sessstate.Projection, error) {
	if store == nil || store.index == nil || repo == nil {
		return sessstate.Projection{}, ErrInvalidRequest
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.index.RebuildProjection(repo, sessionID)
}

func (store *DurableIndex) validateTombstoneAckClosureLocked() error {
	return store.index.validateTombstoneAckClosure()
}

func (index *Index) validateTombstoneAckClosure() error {
	index.mu.RLock()
	defer index.mu.RUnlock()
	for _, acknowledgement := range index.objects["tombstone_ack"] {
		members, err := strictObject(acknowledgement.data)
		if err != nil {
			return refusal("integrity_failure", err)
		}
		subjectID, subjectOK := requiredJSONString(members, "subject_id")
		tombstoneID, tombstoneOK := requiredJSONString(members, "tombstone_id")
		if !subjectOK || !tombstoneOK {
			return refusal("integrity_failure", ErrInvalidObject)
		}
		tombstone, exists := index.objects["tombstone"][tombstoneID]
		if !exists {
			return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement references no unioned Tombstone", ErrInvalidObject))
		}
		tombstoneMembers, err := strictObject(tombstone.data)
		if err != nil {
			return refusal("integrity_failure", err)
		}
		referencedSubject, ok := requiredJSONString(tombstoneMembers, "subject_id")
		if !ok || referencedSubject != subjectID {
			return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement subject does not match its Tombstone", ErrInvalidObject))
		}
	}
	return nil
}

func (index *Index) objectIDs(namespace string) []string {
	index.mu.RLock()
	defer index.mu.RUnlock()
	return index.objectIDsLocked(namespace)
}

// RebuildProjection validates the persisted-authority bytes represented in an
// inventory snapshot, derives every validated lease head from the full record
// union, then delegates state derivation to the pure sessstate projector.
func (index *Index) RebuildProjection(repo *sessrepo.Repository, sessionID string) (sessstate.Projection, error) {
	if index == nil || repo == nil {
		return sessstate.Projection{}, ErrInvalidRequest
	}
	if err := index.validateTombstoneAckClosure(); err != nil {
		return sessstate.Projection{}, err
	}
	recordBytes, err := repo.GetRecord(sessionID)
	if err != nil {
		return sessstate.Projection{}, err
	}
	recordMembership, err := ClassifyJSON(recordBytes)
	if err != nil || recordMembership.Namespace != "record" {
		return sessstate.Projection{}, refusal("integrity_failure", fmt.Errorf("%w: repository record failed inventory validation", ErrIntegrity))
	}
	index.mu.RLock()
	storedRecord, recordPresent := index.objects["record"][recordMembership.ID.String()]
	index.mu.RUnlock()
	if !recordPresent || !bytes.Equal(storedRecord.data, recordBytes) {
		return sessstate.Projection{}, refusal("integrity_failure", fmt.Errorf("%w: authoritative repository record is absent from the union", ErrIntegrity))
	}
	events, err := repo.ListEvents(sessionID)
	if err != nil {
		return sessstate.Projection{}, err
	}
	for _, event := range events {
		eventBytes, err := repo.GetEvent(sessionID, event.EventID)
		if err != nil {
			return sessstate.Projection{}, err
		}
		membership, err := ClassifyJSON(eventBytes)
		if err != nil || membership.Namespace != "event" || membership.ID.String() != event.EventID {
			return sessstate.Projection{}, refusal("integrity_failure", fmt.Errorf("%w: authoritative event failed inventory validation", ErrIntegrity))
		}
		index.mu.RLock()
		stored, present := index.objects["event"][event.EventID]
		index.mu.RUnlock()
		if !present || !bytes.Equal(stored.data, eventBytes) {
			return sessstate.Projection{}, refusal("integrity_failure", fmt.Errorf("%w: authoritative event %s is absent from the union", ErrIntegrity, event.EventID))
		}
	}
	index.mu.RLock()
	leaseRecords := make([][]byte, 0)
	for _, id := range index.objectIDsLocked("record") {
		object := index.objects["record"][id]
		membership, err := ClassifyJSON(object.data)
		if err == nil && membership.Schema == "urn:ax:schema:lease" {
			leaseRecords = append(leaseRecords, bytes.Clone(object.data))
		}
	}
	index.mu.RUnlock()
	reader := &sessquery.Reader{LeaseRecords: leaseRecords}
	leaseHeads, err := reader.LeaseHeadsForSession(sessionID)
	if err != nil {
		return sessstate.Projection{}, err
	}
	return (&sessstate.Projector{Repo: repo, Union: leaseHeads}).Project(sessionID)
}

func (index *Index) objectIDsLocked(namespace string) []string {
	ids := make([]string, 0, len(index.objects[namespace]))
	for id := range index.objects[namespace] {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (store *DurableIndex) load() error {
	quarantined, err := store.loadQuarantine()
	if err != nil {
		return err
	}
	active, err := store.loadActive()
	if err != nil {
		return err
	}
	for _, object := range active {
		membership, err := ClassifyJSON(object.data)
		if err != nil || membership.Excluded || membership.Namespace != object.namespace || membership.ID.Hex()+".json" != object.name {
			return fmt.Errorf("%w: active object %s/%s failed schema, digest, or path validation", ErrDurableCorrupt, object.namespace, object.name)
		}
		if candidates := quarantined[membership.ID.String()]; len(candidates) > 0 {
			if err := store.persistQuarantineCandidate(object.namespace, membership.ID.String(), object.data); err != nil {
				return err
			}
			if err := store.removeActive(object.namespace, membership.ID.String()); err != nil {
				return err
			}
			quarantined[membership.ID.String()] = append(candidates, bytes.Clone(object.data))
			continue
		}
		if err := store.index.AddUnionJSON(object.data); err != nil {
			return fmt.Errorf("%w: rebuild inventory from %s/%s: %v", ErrDurableCorrupt, object.namespace, object.name, err)
		}
	}
	store.index.mu.Lock()
	for id, candidates := range quarantined {
		store.index.quarantine[id] = cloneByteSlices(candidates)
	}
	store.index.mu.Unlock()
	return nil
}

type durableObjectFile struct {
	namespace string
	name      string
	data      []byte
}

func (store *DurableIndex) loadActive() ([]durableObjectFile, error) {
	root := filepath.Join(store.root, "objects")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("%w: read object root: %v", ErrDurableStore, err)
	}
	var objects []durableObjectFile
	for _, namespaceEntry := range entries {
		if !namespaceEntry.IsDir() || !slices.Contains(durableJSONNamespaces, namespaceEntry.Name()) {
			return nil, fmt.Errorf("%w: unexpected object namespace entry %q", ErrDurableCorrupt, namespaceEntry.Name())
		}
		directory := filepath.Join(root, namespaceEntry.Name())
		if err := rejectSymlinkDirectory(directory); err != nil {
			return nil, err
		}
		if err := cleanAbandonedStages(directory); err != nil {
			return nil, err
		}
		files, err := os.ReadDir(directory)
		if err != nil {
			return nil, fmt.Errorf("%w: read object namespace %s: %v", ErrDurableStore, namespaceEntry.Name(), err)
		}
		for _, entry := range files {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				return nil, fmt.Errorf("%w: unexpected active object entry %s/%s", ErrDurableCorrupt, namespaceEntry.Name(), entry.Name())
			}
			data, err := readRegularFile(filepath.Join(directory, entry.Name()))
			if err != nil {
				return nil, err
			}
			objects = append(objects, durableObjectFile{namespace: namespaceEntry.Name(), name: entry.Name(), data: data})
		}
	}
	slices.SortFunc(objects, func(a, b durableObjectFile) int {
		if a.namespace != b.namespace {
			return strings.Compare(a.namespace, b.namespace)
		}
		return strings.Compare(a.name, b.name)
	})
	return objects, nil
}

func (store *DurableIndex) loadQuarantine() (map[string][][]byte, error) {
	root := filepath.Join(store.root, "quarantine")
	namespaces, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("%w: read quarantine root: %v", ErrDurableStore, err)
	}
	quarantined := map[string][][]byte{}
	for _, namespaceEntry := range namespaces {
		if !namespaceEntry.IsDir() || !slices.Contains(durableJSONNamespaces, namespaceEntry.Name()) {
			return nil, fmt.Errorf("%w: unexpected quarantine namespace entry %q", ErrDurableCorrupt, namespaceEntry.Name())
		}
		namespaceDir := filepath.Join(root, namespaceEntry.Name())
		if err := rejectSymlinkDirectory(namespaceDir); err != nil {
			return nil, err
		}
		ids, err := os.ReadDir(namespaceDir)
		if err != nil {
			return nil, fmt.Errorf("%w: read quarantine namespace %s: %v", ErrDurableStore, namespaceEntry.Name(), err)
		}
		for _, idEntry := range ids {
			if !idEntry.IsDir() || len(idEntry.Name()) != 64 {
				return nil, fmt.Errorf("%w: unexpected quarantined ID entry %s/%s", ErrDurableCorrupt, namespaceEntry.Name(), idEntry.Name())
			}
			id, err := scalar.ParseDigest("sha256:" + idEntry.Name())
			if err != nil || id.Hex() != idEntry.Name() {
				return nil, fmt.Errorf("%w: malformed quarantined digest directory %q", ErrDurableCorrupt, idEntry.Name())
			}
			idDir := filepath.Join(namespaceDir, idEntry.Name())
			if err := rejectSymlinkDirectory(idDir); err != nil {
				return nil, err
			}
			if err := cleanAbandonedStages(idDir); err != nil {
				return nil, err
			}
			candidates, err := os.ReadDir(idDir)
			if err != nil {
				return nil, fmt.Errorf("%w: read quarantined identity %s: %v", ErrDurableStore, id.String(), err)
			}
			for _, candidate := range candidates {
				if candidate.IsDir() || len(candidate.Name()) != 64 {
					return nil, fmt.Errorf("%w: unexpected quarantine candidate %s", ErrDurableCorrupt, candidate.Name())
				}
				data, err := readRegularFile(filepath.Join(idDir, candidate.Name()))
				if err != nil {
					return nil, err
				}
				sum := sha256.Sum256(data)
				membership, classifyErr := ClassifyJSON(data)
				if hex.EncodeToString(sum[:]) != candidate.Name() || classifyErr != nil || membership.ID.String() != id.String() || membership.Namespace != namespaceEntry.Name() {
					return nil, fmt.Errorf("%w: quarantined bytes do not match identity %s", ErrDurableCorrupt, id.String())
				}
				quarantined[id.String()] = append(quarantined[id.String()], bytes.Clone(data))
			}
		}
	}
	return quarantined, nil
}

func (store *DurableIndex) objectPath(namespace, id string) (string, error) {
	digest, err := scalar.ParseDigest(id)
	if err != nil || !slices.Contains(durableJSONNamespaces, namespace) {
		return "", ErrInvalidRequest
	}
	return filepath.Join(store.root, "objects", namespace, digest.Hex()+".json"), nil
}

func (store *DurableIndex) quarantinePath(namespace, id string, data []byte) (string, error) {
	digest, err := scalar.ParseDigest(id)
	if err != nil || !slices.Contains(durableJSONNamespaces, namespace) {
		return "", ErrInvalidRequest
	}
	sum := sha256.Sum256(data)
	return filepath.Join(store.root, "quarantine", namespace, digest.Hex(), hex.EncodeToString(sum[:])), nil
}

func (store *DurableIndex) persistQuarantineCandidate(namespace, id string, data []byte) error {
	path, err := store.quarantinePath(namespace, id, data)
	if err != nil {
		return err
	}
	if err := ensureDurableDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	installed, existing, err := durableInstallNoReplace(path, data)
	if err != nil {
		return err
	}
	if existing != nil && !bytes.Equal(existing, data) {
		return fmt.Errorf("%w: quarantine content hash path contains different bytes", ErrDurableCorrupt)
	}
	if installed && store.ops.afterQuarantineCandidate != nil {
		store.ops.afterQuarantineCandidate(id)
	}
	return nil
}

func (store *DurableIndex) removeActive(namespace, id string) error {
	path, err := store.objectPath(namespace, id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: remove quarantined active object: %v", ErrDurableStore, err)
	}
	if err := syncDurableDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	if store.ops.afterActiveRemoval != nil {
		store.ops.afterActiveRemoval(id)
	}
	return nil
}

func durableInstallNoReplace(path string, data []byte) (installed bool, prior []byte, resultErr error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".ax-union-stage-")
	if err != nil {
		return false, nil, fmt.Errorf("%w: create staged object: %v", ErrDurableStore, err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if cleanupErr := os.Remove(temporaryPath); cleanupErr != nil && !errors.Is(cleanupErr, fs.ErrNotExist) {
			resultErr = errors.Join(resultErr, fmt.Errorf("%w: remove staged object: %v", ErrDurableStore, cleanupErr))
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return false, nil, fmt.Errorf("%w: set staged object mode: %v", ErrDurableStore, err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return false, nil, fmt.Errorf("%w: write staged object: %v", ErrDurableStore, err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return false, nil, fmt.Errorf("%w: sync staged object: %v", ErrDurableStore, err)
	}
	if err := temporary.Close(); err != nil {
		return false, nil, fmt.Errorf("%w: close staged object: %v", ErrDurableStore, err)
	}
	if err := os.Link(temporaryPath, path); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return false, nil, fmt.Errorf("%w: atomically install object: %v", ErrDurableStore, err)
		}
		prior, readErr := readRegularFile(path)
		if readErr != nil {
			return false, nil, readErr
		}
		return false, prior, nil
	}
	if err := os.Remove(temporaryPath); err != nil {
		return true, nil, fmt.Errorf("%w: remove installed staging name: %v", ErrDurableStore, err)
	}
	if err := syncDurableDirectory(directory); err != nil {
		return true, nil, err
	}
	return true, nil, nil
}

func ensureDurableDirectory(path string) error {
	var missing []string
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: path is not a plain directory", ErrDurableCorrupt)
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: inspect directory path: %v", ErrDurableStore, err)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("%w: no existing ancestor for directory", ErrDurableStore)
		}
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("%w: create directory: %v", ErrDurableStore, err)
	}
	for index := len(missing) - 1; index >= 0; index-- {
		created := missing[index]
		if err := rejectSymlinkDirectory(created); err != nil {
			return err
		}
		if err := syncDurableDirectory(filepath.Dir(created)); err != nil {
			return err
		}
	}
	return rejectSymlinkDirectory(path)
}

func cleanAbandonedStages(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("%w: read staging directory: %v", ErrDurableStore, err)
	}
	removed := false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".ax-union-stage-") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("%w: inspect abandoned stage: %v", ErrDurableStore, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: abandoned staging path is not a regular file", ErrDurableCorrupt)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("%w: remove abandoned stage: %v", ErrDurableStore, err)
		}
		removed = true
	}
	if removed {
		return syncDurableDirectory(directory)
	}
	return nil
}

func rejectSymlinkDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: inspect directory: %v", ErrDurableStore, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: path is not a plain directory", ErrDurableCorrupt)
	}
	return nil
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: inspect object file: %v", ErrDurableStore, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: object path is not a regular file", ErrDurableCorrupt)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read object file: %v", ErrDurableStore, err)
	}
	return data, nil
}

func syncDurableDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open directory for sync: %v", ErrDurableStore, err)
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		return fmt.Errorf("%w: sync directory: %v", ErrDurableStore, syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%w: close synced directory: %v", ErrDurableStore, closeErr)
	}
	return nil
}

func cloneByteSlices(values [][]byte) [][]byte {
	cloned := make([][]byte, len(values))
	for index := range values {
		cloned[index] = bytes.Clone(values[index])
	}
	return cloned
}
