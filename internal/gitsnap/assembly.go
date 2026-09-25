package gitsnap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// WorkspaceSource supplies workspace-record facts and the existing content
// policy. ProjectConfigPaths is keyed by repository-relative submodule path;
// "." selects the containing repository. Ignored files still require an
// explicit non-secret IncludeIgnored classification in Content.
type WorkspaceSource struct {
	WorkspaceID       string
	GroupRelativePath string
	// Directory locates the source repository; the Group Record selects resume cwd.
	Directory             string
	MaterializationPolicy string
	Content               ContentOptions
	ProjectConfigPaths    map[string][]string
}

type AssemblyOptions struct {
	// ScratchRoot is the caller-owned temporary directory for disposable Git
	// verifier repositories. It must be absolute and already exist; capture
	// never chooses a process-global temporary directory implicitly.
	ScratchRoot string
	// GroupRecord is the exact immutable topology selected by the caller.
	// Resolving conflicting group histories belongs to the record owner.
	GroupRecord      json.RawMessage
	WorkspaceGroupID string
	CreatedByHostID  string
	CreatedAt        string
	BaseCheckpointID *string
	Members          []WorkspaceSource
}

// ProvisionalAssembly contains immutable transfer objects produced below the
// coordinated-capture boundary. It is NOT a checkpoint or proof of held runtime
// quiescence. No publish/doctor capability consumes this value yet.
// Manifest and descriptor maps are keyed by canonical self-identity; their
// complete JSON representations have separate byte-digest storage addresses.
type ProvisionalAssembly struct {
	GroupRecord     json.RawMessage
	RootID          string
	Manifests       map[string]json.RawMessage
	Descriptors     map[string]BlobDescriptor
	ObjectAddresses map[string]string
}

type assemblyState struct {
	runner  AssemblyRunner
	options AssemblyOptions
	result  *ProvisionalAssembly
	content *contentState
	source  WorkspaceSource
	cwd     string
}

// AssembleProvisional drives Capture, repository-local pack/index construction,
// recursive wire assembly and closing observations. Its caller must eventually
// be the real runtime coordinator that holds input/provider quiescence; a stable
// observation here cannot establish that lifetime or detect change-and-revert.
func AssembleProvisional(ctx context.Context, runner AssemblyRunner, options AssemblyOptions) (*ProvisionalAssembly, error) {
	if runner == nil || len(options.Members) == 0 || len(options.Members) > 256 {
		return nil, refuse(GateCapturePolicy, "assembly requires a runner and 1..256 workspace members")
	}
	if !filepath.IsAbs(options.ScratchRoot) {
		return nil, refuse(GateCapturePolicy, "assembly requires an absolute scratch root")
	}
	if info, err := os.Stat(options.ScratchRoot); err != nil || !info.IsDir() {
		return nil, refuse(GateCapturePolicy, "assembly scratch root must be an existing directory")
	}
	if _, err := scalar.ParseUUIDv7(options.WorkspaceGroupID); err != nil {
		return nil, refuse(GateCapturePolicy, "invalid workspace group ID")
	}
	if _, err := scalar.ParseUUIDv7(options.CreatedByHostID); err != nil {
		return nil, refuse(GateCapturePolicy, "invalid capturing host ID")
	}
	if _, err := scalar.ParseTimestamp(options.CreatedAt); err != nil {
		return nil, refuse(GateCapturePolicy, "invalid capture timestamp")
	}
	if options.BaseCheckpointID != nil {
		if _, err := scalar.ParseDigest(*options.BaseCheckpointID); err != nil {
			return nil, refuse(GateCapturePolicy, "invalid predecessor checkpoint ID")
		}
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(options.GroupRecord); err != nil {
		return nil, refuse(GateCapturePolicy, "valid immutable workspace group record is required")
	}
	var group map[string]any
	if err := json.Unmarshal(options.GroupRecord, &group); err != nil {
		return nil, refuse(GateCapturePolicy, "invalid workspace group record")
	}
	if group["schema"] != "urn:ax:schema:workspace-group" || group["workspace_group_id"] != options.WorkspaceGroupID {
		return nil, refuse(GateCapturePolicy, "workspace group record binding mismatch")
	}
	if len(group["members"].([]any)) != len(options.Members) {
		return nil, refuse(GateCapturePolicy, "assembly member count differs from group record")
	}
	state := &assemblyState{runner: runner, options: options, result: &ProvisionalAssembly{GroupRecord: append(json.RawMessage(nil), options.GroupRecord...), Manifests: map[string]json.RawMessage{}, Descriptors: map[string]BlobDescriptor{}, ObjectAddresses: map[string]string{}}}
	members := append([]WorkspaceSource(nil), options.Members...)
	sort.Slice(members, func(i, j int) bool { return members[i].WorkspaceID < members[j].WorkspaceID })
	paths := []string{}
	for i, member := range members {
		if _, err := scalar.ParseUUIDv7(member.WorkspaceID); err != nil {
			return nil, refuse(GateCapturePolicy, "invalid workspace ID")
		}
		if member.MaterializationPolicy != "shared_checkout" && member.MaterializationPolicy != "separate_worktree" {
			return nil, refuse(GateCapturePolicy, "invalid Git materialization policy")
		}
		if member.Content.Store == nil || member.Content.Store.DataRoot() != members[0].Content.Store.DataRoot() {
			return nil, refuse(GateCapturePolicy, "all assembly members require the same immutable store")
		}

		if _, err := scalar.ParseRelativePath(member.GroupRelativePath); err != nil {
			return nil, refuse(GateContentPath, "invalid workspace group path")
		}
		if i > 0 && member.WorkspaceID == members[i-1].WorkspaceID {
			return nil, refuse(GateCapturePolicy, "duplicate workspace ID")
		}
		for _, previous := range paths {
			if workspacePathsOverlap(previous, member.GroupRelativePath) {
				return nil, refuse(GateContentPath, "workspace paths overlap or case-collide")
			}
		}
		paths = append(paths, member.GroupRelativePath)
	}
	wireMembers := []any{}
	classes := map[string]bool{}
	snapshots := make([]*Snapshot, 0, len(members))
	for memberIndex, member := range members {
		state.source = member
		state.cwd = group["members"].([]any)[memberIndex].(map[string]any)["repo_relative_cwd"].(string)
		state.content = &contentState{options: member.Content, active: map[string]bool{}}
		if err := state.content.validate(); err != nil {
			return nil, err
		}
		snapshot, err := captureAssembly(ctx, runner, member.Directory, member.Content)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
		known := map[string]bool{}
		var collect func(*Snapshot, string)
		collect = func(current *Snapshot, prefix string) {
			known[prefix] = true
			for _, sub := range current.Content.Submodules {
				if sub.Initialized {
					collect(sub.State, path.Join(prefix, sub.Path))
				}
			}
		}
		collect(snapshot, ".")
		for prefix := range member.ProjectConfigPaths {
			if !known[prefix] {
				return nil, refuse(GateContentPath, "project configuration names an uncaptured repository")
			}
		}

		wire, treeID, err := state.repository(ctx, snapshot, ".", member.WorkspaceID)
		if err != nil {
			return nil, err
		}
		wire["workspace_id"] = member.WorkspaceID
		wire["kind"] = "git"
		wire["group_relative_path"] = member.GroupRelativePath
		wire["materialization_policy"] = member.MaterializationPolicy
		wireMembers = append(wireMembers, wire)
		_ = treeID
		for _, class := range snapshot.Content.ExcludedClasses {
			classes[class] = true
		}
	}
	// Re-scan every member after all repository assembly, including recursively
	// observed files, refs, features, directory entries, symlinks and config paths.
	// This closes the observation interval across siblings; it is not a lock.
	for i, member := range members {
		again, err := captureAssembly(ctx, runner, member.Directory, member.Content)
		if err != nil {
			return nil, err
		}
		if !sameObservation(snapshots[i], again) {
			return nil, refuse(GateConsistency, "workspace changed during provisional assembly")
		}
	}
	children, err := state.rootChildren(ctx)
	if err != nil {
		return nil, err
	}
	root := state.manifest("workspace_group", options.WorkspaceGroupID, []map[string]any{}, children, sortedKeys(classes))
	root["workspace_snapshot"] = map[string]any{"workspace_group_id": options.WorkspaceGroupID, "members": wireMembers}
	id, err := state.seal(ctx, root)
	if err != nil {
		return nil, err
	}
	state.result.RootID = id
	if err := ValidateProvisional(state.result); err != nil {
		return nil, err
	}
	return state.result, nil
}

func sameObservation(a, b *Snapshot) bool {
	if !a.observedIndex.equal(b.observedIndex) {
		return false
	}
	if len(a.Content.Submodules) != len(b.Content.Submodules) {
		return false
	}
	for i, sub := range a.Content.Submodules {
		other := b.Content.Submodules[i]
		if sub.Initialized && other.Initialized && !sameObservation(sub.State, other.State) {
			return false
		}
	}
	// FileInfo is not part of Snapshot. JSON normalizes pointer identities and
	// retains the actual content digests and exact recursive wire observations.
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}

func (s *assemblyState) repository(ctx context.Context, snapshot *Snapshot, prefix, workspaceID string) (map[string]any, string, error) {
	objects, err := s.objects(ctx, snapshot)
	if err != nil {
		return nil, "", err
	}
	content := snapshot.Content
	for _, descriptor := range content.Blobs {
		s.recordDescriptor(descriptor)
	}
	if content.SparsePatterns != nil {
		s.recordDescriptor(*content.SparsePatterns)
	}
	submodules := []any{}
	children := []string{}
	for _, sub := range content.Submodules {
		wire := map[string]any{"path": sub.Path, "repository_identity": sub.RepositoryIdentity, "sanitized_url": sub.SanitizedURL, "gitlink_oid": sub.GitlinkOID, "initialized": sub.Initialized, "head": nil, "upstream_ref": nil, "object_pack": nil, "index": nil, "working_tree_manifest_id": nil, "submodules": nil, "features": nil, "repo_relative_cwd": nil, "agent_project_config_paths": nil}
		if sub.Initialized {
			child, id, err := s.repository(ctx, sub.State, path.Join(prefix, sub.Path), workspaceID)
			if err != nil {
				return nil, "", err
			}
			for _, key := range []string{"head", "upstream_ref", "object_pack", "index", "working_tree_manifest_id", "submodules", "features", "repo_relative_cwd", "agent_project_config_paths"} {
				wire[key] = child[key]
			}
			_ = id
		}
		submodules = append(submodules, wire)
	}
	configs := append([]string{}, s.source.ProjectConfigPaths[prefix]...)
	sort.Strings(configs)
	cwd := snapshot.Worktree.CWD
	if prefix == "." {
		cwd = s.cwd
	}
	if err := checkCapturedPaths(snapshotEntryClosure(snapshot), cwd, configs); err != nil {
		return nil, "", err
	}
	tree := s.manifest("workspace_tree", workspaceID, content.Entries, children, content.ExcludedClasses)
	treeID, err := s.seal(ctx, tree)
	if err != nil {
		return nil, "", err
	}
	features := map[string]any{"object_format": snapshot.Features.ObjectFormat, "filemode": snapshot.Features.FileMode, "symlinks": snapshot.Features.Symlinks, "case_sensitive": snapshot.Features.CaseSensitive, "precompose_unicode": snapshot.Features.PrecomposeUnicode, "sparse_checkout": snapshot.Features.SparseCheckout, "sparse_patterns_blob_id": nil, "sparse_patterns_blob_descriptor_id": nil, "required_filter_names": append([]string{}, snapshot.Features.RequiredFilters...), "lfs_required": snapshot.Features.LFSRequired}
	if content.SparsePatterns != nil {
		features["sparse_patterns_blob_id"] = content.SparsePatterns.BlobID
		features["sparse_patterns_blob_descriptor_id"] = content.SparsePatterns.DescriptorID
	}
	return map[string]any{"repository_identity": snapshot.RepositoryIdentity, "remotes": snapshot.Remotes, "head": snapshot.Head, "upstream_ref": snapshot.UpstreamRef, "object_pack": packWire(objects, snapshot.ObjectFormat), "index": indexWire(snapshot.Index, objects.index), "working_tree_manifest_id": treeID, "submodules": submodules, "features": features, "repo_relative_cwd": cwd, "agent_project_config_paths": configs}, treeID, nil
}

func sortedKeys(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (s *assemblyState) manifest(kind, subject string, entries []map[string]any, children, classes []string) map[string]any {
	sort.Strings(children)
	return map[string]any{"schema": "urn:ax:schema:transfer-manifest", "schema_version": "1.0.0", "manifest_id": "sha256:" + strings.Repeat("0", 64), "kind": kind, "subject_id": subject, "base_checkpoint_id": s.options.BaseCheckpointID, "entries": entries, "child_manifest_ids": children, "workspace_snapshot": nil, "provider_identity_record_id": nil, "task_board_bundle_id": nil, "excluded_classes": classes, "created_by_host_id": s.options.CreatedByHostID, "created_at": s.options.CreatedAt, "extensions": map[string]any{}}
}

func (s *assemblyState) seal(ctx context.Context, value map[string]any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", refuse(GateManifestClosure, "cannot encode transfer manifest")
	}
	id, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		return "", refuse(GateManifestClosure, "transfer manifest violates closed schema: "+err.Error())
	}
	value["manifest_id"] = id.String()
	raw, err = json.Marshal(value)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	address, _ := scalar.ParseDigest(digestHex(sum[:]))
	if err := s.content.installBlob(address, uint64(len(raw)), bytes.NewReader(raw)); err != nil {
		return "", err
	}
	s.result.Manifests[id.String()] = raw
	s.result.ObjectAddresses[id.String()] = address.String()
	return id.String(), nil
}

func (s *assemblyState) addBlob(ctx context.Context, raw []byte, media string) (BlobDescriptor, error) {
	descriptor, err := s.content.blob(ctx, bytes.NewReader(raw), int64(len(raw)), media)
	if err == nil {
		s.recordDescriptor(descriptor)
	}
	return descriptor, err
}

func readAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	return io.ReadAll(&contextReader{ctx: ctx, reader: reader})
}

func checkCapturedPaths(entries []map[string]any, cwd string, configs []string) error {
	byPath := map[string]string{}
	for _, entry := range entries {
		byPath[entry["path"].(string)] = entry["type"].(string)
	}
	if cwd != "." && byPath[cwd] != "directory" {
		return refuse(GateContentPath, "cwd is absent from the captured tree")
	}
	for i, name := range configs {
		if _, err := scalar.ParseRelativePath(name); err != nil || (i > 0 && name == configs[i-1]) {
			return refuse(GateContentPath, "invalid or duplicate project configuration path")
		}
		if byPath[name] != "file" {
			return refuse(GateContentPath, "project configuration is absent or not a captured file")
		}
	}
	return nil
}

// ValidateProvisional validates the immutable object graph below capture. It
// accepts no runtime evidence and cannot authorize a checkpoint. Canonical shape
// validation belongs to canonicaljson; this layer checks transitive availability,
// descriptor agreement, repository tree bindings and cwd/config existence.
func ValidateProvisional(value *ProvisionalAssembly) error {
	if value == nil {
		return refuse(GateManifestClosure, "missing provisional assembly")
	}
	manifests := map[string]map[string]any{}
	for key, raw := range value.Manifests {
		id, _, err := canonicaljson.VerifyObjectIdentity(raw)
		if err != nil || id.String() != key {
			return refuse(GateManifestClosure, "manifest identity mismatch")
		}
		var object map[string]any
		if err := json.Unmarshal(raw, &object); err != nil {
			return refuse(GateManifestClosure, "invalid manifest JSON")
		}
		manifests[key] = object
	}
	root, ok := manifests[value.RootID]
	if !ok || root["kind"] != "workspace_group" {
		return refuse(GateManifestClosure, "workspace root missing")
	}
	reached := map[string]bool{}
	active := map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if active[id] {
			return refuse(GateManifestClosure, "cyclic manifest closure")
		}
		if reached[id] {
			return nil
		}
		object, ok := manifests[id]
		if !ok {
			return refuse(GateManifestClosure, "child manifest missing from closure")
		}
		active[id] = true
		for _, child := range object["child_manifest_ids"].([]any) {
			if err := visit(child.(string)); err != nil {
				return err
			}
		}
		delete(active, id)
		reached[id] = true
		for _, item := range object["entries"].([]any) {
			entry := item.(map[string]any)
			if entry["type"] == "file" {
				if err := checkDescriptor(value, entry["blob_descriptor_id"], entry["blob_id"], entry["size"]); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := checkGroupRecord(value.GroupRecord, root); err != nil {
		return err
	}
	if err := visit(value.RootID); err != nil {
		return err
	}
	if len(reached) != len(manifests) {
		return refuse(GateManifestClosure, "orphan manifest outside root closure")
	}
	boundTrees := map[string]bool{}
	var repository func(map[string]any, string) error
	repository = func(member map[string]any, workspaceID string) error {
		treeID := member["working_tree_manifest_id"].(string)
		tree, ok := manifests[treeID]
		if !ok || !reached[treeID] || tree["kind"] != "workspace_tree" || tree["subject_id"] != workspaceID {
			return refuse(GateManifestClosure, "repository tree absent or bound to another workspace")
		}
		boundTrees[treeID] = true
		entries, err := manifestEntryClosure(member, manifests)
		if err != nil {
			return err
		}
		configs := []string{}
		for _, name := range member["agent_project_config_paths"].([]any) {
			configs = append(configs, name.(string))
		}
		if err := checkCapturedPaths(entries, member["repo_relative_cwd"].(string), configs); err != nil {
			return err
		}
		for _, name := range []string{"object_pack", "index", "features"} {
			object := member[name].(map[string]any)
			for _, pair := range [][2]string{{"blob_descriptor_id", "blob_id"}, {"inventory_blob_descriptor_id", "inventory_blob_id"}, {"sparse_patterns_blob_descriptor_id", "sparse_patterns_blob_id"}} {
				if descriptor, ok := object[pair[0]]; ok && descriptor != nil {
					if err := checkDescriptor(value, descriptor, object[pair[1]], nil); err != nil {
						return err
					}
				}
			}
		}
		for _, item := range member["submodules"].([]any) {
			sub := item.(map[string]any)
			if sub["initialized"] == true {
				if err := repository(sub, workspaceID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	groupPaths := []string{}
	for _, item := range root["workspace_snapshot"].(map[string]any)["members"].([]any) {
		member := item.(map[string]any)
		for _, previous := range groupPaths {
			if workspacePathsOverlap(previous, member["group_relative_path"].(string)) {
				return refuse(GateManifestClosure, "workspace paths overlap")
			}
		}
		groupPaths = append(groupPaths, member["group_relative_path"].(string))
		if member["kind"] != "git" {
			return refuse(GateManifestClosure, "provisional Git assembly cannot validate managed trees")
		}
		if err := repository(member, member["workspace_id"].(string)); err != nil {
			return err
		}
	}
	for id, manifest := range manifests {
		if manifest["kind"] == "workspace_tree" && !boundTrees[id] {
			return refuse(GateManifestClosure, "unbound workspace tree in closure")
		}
	}
	return nil
}

func checkDescriptor(value *ProvisionalAssembly, descriptorID, blobID, size any) error {
	descriptor, ok := value.Descriptors[descriptorID.(string)]
	if !ok || descriptor.DescriptorID != descriptorID || descriptor.BlobID != blobID || (size != nil && !reflect.DeepEqual(float64(descriptor.Size), size)) {
		return refuse(GateManifestClosure, "blob descriptor missing or disagrees with reference")
	}
	raw, err := json.Marshal(descriptor)
	if err != nil {
		return refuse(GateManifestClosure, "invalid descriptor")
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(raw); err != nil {
		return refuse(GateManifestClosure, "descriptor identity mismatch")
	}
	return nil
}

func sortedKeysManifest(values map[string]json.RawMessage) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func (s *assemblyState) recordDescriptor(descriptor BlobDescriptor) {
	s.result.Descriptors[descriptor.DescriptorID] = descriptor
	raw, _ := json.Marshal(descriptor)
	sum := sha256.Sum256(raw)
	s.result.ObjectAddresses[descriptor.DescriptorID] = digestHex(sum[:])
}

func workspacePathsOverlap(a, b string) bool {
	left, right := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(left) && i < len(right); i++ {
		if !strings.EqualFold(left[i], right[i]) {
			return false
		}
	}
	return true
}

func checkGroupRecord(raw json.RawMessage, root map[string]any) error {
	if _, _, err := canonicaljson.VerifyObjectIdentity(raw); err != nil {
		return refuse(GateManifestClosure, "invalid workspace group record")
	}
	var group map[string]any
	if err := json.Unmarshal(raw, &group); err != nil {
		return refuse(GateManifestClosure, "invalid workspace group JSON")
	}
	if group["schema"] != "urn:ax:schema:workspace-group" || group["workspace_group_id"] != root["subject_id"] {
		return refuse(GateManifestClosure, "workspace group identity differs from root")
	}
	expected := group["members"].([]any)
	actual := root["workspace_snapshot"].(map[string]any)["members"].([]any)
	if len(expected) != len(actual) {
		return refuse(GateManifestClosure, "workspace root omits group members")
	}
	for i, item := range expected {
		member, wire := item.(map[string]any), actual[i].(map[string]any)
		if member["kind"] != "git" || wire["kind"] != "git" {
			return refuse(GateManifestClosure, "provisional assembly requires Git members")
		}
		for _, key := range []string{"workspace_id", "kind", "group_relative_path", "repository_identity", "repo_relative_cwd", "agent_project_config_paths", "materialization_policy"} {
			if !reflect.DeepEqual(member[key], wire[key]) {
				return refuse(GateManifestClosure, "workspace root disagrees with group record "+key)
			}
		}
		urls := map[string]bool{}
		for _, item := range wire["remotes"].([]any) {
			remote := item.(map[string]any)
			urls[remote["fetch_url"].(string)] = true
			if remote["push_url"] != nil {
				urls[remote["push_url"].(string)] = true
			}
		}
		expectedURLs := []string{}
		for _, url := range member["sanitized_remote_urls"].([]any) {
			expectedURLs = append(expectedURLs, url.(string))
		}
		if !reflect.DeepEqual(sortedKeys(urls), expectedURLs) {
			return refuse(GateManifestClosure, "workspace remotes disagree with group record")
		}
	}
	return nil
}

// Root children are bounded at 1024 by Section 10.4. Large groups use one
// empty composite per workspace, each covering its root plus at most 256
// recursive submodule trees. No tree entries are copied into the composites.
func (s *assemblyState) rootChildren(ctx context.Context) ([]string, error) {
	ids := sortedKeysManifest(s.result.Manifests)
	if len(ids) <= 1024 {
		return ids, nil
	}
	groups := map[string][]string{}
	for _, id := range ids {
		var tree map[string]any
		if err := json.Unmarshal(s.result.Manifests[id], &tree); err != nil {
			return nil, refuse(GateManifestClosure, "invalid tree while partitioning closure")
		}
		subject := tree["subject_id"].(string)
		groups[subject] = append(groups[subject], id)
	}
	children := []string{}
	for _, ids := range groups {
		value := s.manifest("composite", s.options.WorkspaceGroupID, []map[string]any{}, ids, []string{})
		id, err := s.seal(ctx, value)
		if err != nil {
			return nil, err
		}
		children = append(children, id)
	}
	sort.Strings(children)
	return children, nil
}

func prefixEntries(entries []map[string]any, prefix string) []map[string]any {
	result := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		copy := make(map[string]any, len(entry))
		for key, value := range entry {
			copy[key] = value
		}
		copy["path"] = path.Join(prefix, entry["path"].(string))
		if target, ok := entry["target_path"]; ok {
			copy["target_path"] = path.Join(prefix, target.(string))
		}
		result = append(result, copy)
	}
	return result
}
func snapshotEntryClosure(snapshot *Snapshot) []map[string]any {
	entries := append([]map[string]any{}, snapshot.Content.Entries...)
	for _, sub := range snapshot.Content.Submodules {
		if sub.Initialized {
			entries = append(entries, prefixEntries(snapshotEntryClosure(sub.State), sub.Path)...)
		}
	}
	return entries
}
func manifestEntryClosure(member map[string]any, manifests map[string]map[string]any) ([]map[string]any, error) {
	tree, ok := manifests[member["working_tree_manifest_id"].(string)]
	if !ok {
		return nil, refuse(GateManifestClosure, "repository tree missing while resolving paths")
	}
	own := []map[string]any{}
	for _, entry := range tree["entries"].([]any) {
		own = append(own, entry.(map[string]any))
	}
	entries := append([]map[string]any{}, own...)
	siblingPaths := []string{}
	for _, item := range member["submodules"].([]any) {
		sub := item.(map[string]any)
		if sub["initialized"] != true {
			continue
		}
		subPath := sub["path"].(string)
		for _, other := range siblingPaths {
			if workspacePathsOverlap(other, subPath) {
				return nil, refuse(GateContentPath, "submodule partitions overlap")
			}
		}
		siblingPaths = append(siblingPaths, subPath)
		for _, entry := range own {
			name := entry["path"].(string)
			// A parent may name the containing directories and the gitlink directory,
			// but may not own entries inside a child's repository partition. Each
			// tree's own entry grammar/count is already checked by canonicaljson;
			// transitive closure must not acquire a per-manifest 65536-entry cap.
			if workspacePathsOverlap(name, subPath) && (entry["type"] != "directory" || len(strings.Split(name, "/")) > len(strings.Split(subPath, "/"))) {
				return nil, refuse(GateContentPath, "parent tree overlaps submodule partition")
			}
		}
		child, err := manifestEntryClosure(sub, manifests)
		if err != nil {
			return nil, err
		}
		entries = append(entries, prefixEntries(child, subPath)...)
	}
	return entries, nil
}
