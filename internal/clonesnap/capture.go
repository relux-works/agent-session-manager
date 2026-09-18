package clonesnap

import (
	"bytes"
	"os"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// This file owns the native capture operation: Capture walks a real
// provider store directory through the contained walk, installs
// payload blobs through the landed no-replace discipline, seals the
// Raw Object Manifest and Capture Manifest through the landed
// builders, publishes both through the object store, and checkpoints
// the workspace binding. The source-race gate runs after the
// post-capture measurement and before any seal: a detected mutation
// refuses, or seals the core-created unstable_archive form for
// archive-only output when the operator explicitly acknowledged
// archival. Nothing sealed here is trusted undecoded: every sealed
// manifest re-decodes through the landed decoder before it is
// published or returned.

// RacePolicy decides a detected source mutation: RaceRefuse aborts
// capture with no sealed manifest; RaceArchive seals the
// unstable_archive form for archive-only output, which can never
// enter a target branch.
type RacePolicy string

const (
	// RaceRefuse aborts capture on a detected source mutation.
	RaceRefuse RacePolicy = "refuse"
	// RaceArchive seals unstable_archive output on a detected
	// source mutation. It requires OperatorExplicit.
	RaceArchive RacePolicy = "archive"
)

// CaptureHooks carries the crash and race seams: AfterWalk runs
// between the capture walk and the post-capture measurement (race
// injection observes here), AfterRawPublish runs between the raw
// manifest publish and the capture manifest seal (crash kill lands
// here). A hook error aborts capture; a hook that never returns
// (process kill) leaves at most the published prefix behind.
type CaptureHooks struct {
	AfterWalk       func(storeRoot string) error
	AfterRawPublish func() error
}

// CaptureRequest is one native capture candidate over a provider
// store directory.
type CaptureRequest struct {
	// StoreRoot is the absolute provider store directory. Every
	// payload open descends from its verified handle.
	StoreRoot string
	// Platform governs member-name grammar.
	Platform scalar.Platform
	// Plan carries the sorted unique capture-plan candidates.
	Plan []PlanItem
	// OperationID and BundleID identify the clone operation and
	// the logical bundle chain.
	OperationID string
	BundleID    string
	// SourceBasis binds ax_session or external_native source IDs.
	SourceBasis clonebundle.SourceBasisInput
	// SourceEnvironment carries the exact probed Environment Tuple
	// bytes.
	SourceEnvironment []byte
	// SourceIdentity is the sanitized NativeIdentity. Capture
	// re-validates it through the landed IdentityDigest before
	// anything seals: an identity decoding would refuse fails the
	// entry, so no sealed artifact carries it.
	SourceIdentity clonebundle.NativeIdentity
	// CapturePlanDigest is the exact core-validated capture-plan
	// digest.
	CapturePlanDigest string
	// NativeSessionID is the sanitized source native session ID.
	NativeSessionID string
	// CreatedByHostID and CreatedAt stamp the capture manifest.
	CreatedByHostID string
	CreatedAt       string
	// Generation is the immutable source generation token.
	Generation string
	// ProofKind is the closed proof vocabulary:
	// closed_store|immutable_snapshot|verified_log_prefix|provider_quiescence.
	ProofKind string
	// SnapshotIdentity pins an immutable snapshot; nil unless the
	// proof kind requires it.
	SnapshotIdentity *string
	// InputBlocked, ForegroundIdle, and BackgroundIdle are the
	// measured idle facts the stable proof seals.
	InputBlocked   bool
	ForegroundIdle bool
	BackgroundIdle bool
	// OperatorExplicit acknowledges archive-only output when the
	// source is not quiescent.
	OperatorExplicit bool
	// OnRace decides a detected mutation: refuse or archive.
	OnRace RacePolicy
	// Workspace carries the repository state the workspace
	// checkpoint seals.
	Workspace WorkspaceRequest
	// Store receives payload blobs and published manifests.
	Store *localstore.ObjectStore
	// MaxSingleBytes bounds one captured member; zero selects
	// DefaultMaxSingleBytes. A larger member fails capture with
	// capability_unavailable before any manifest is published.
	MaxSingleBytes uint64
	// Hooks carries the crash and race seams; nil hooks are skipped.
	Hooks CaptureHooks
}

// CaptureResult is one completed native capture: the sealed bytes,
// their decoded forms, the measured digests, and the publish facts.
type CaptureResult struct {
	// RawManifest and CaptureManifest carry the sealed manifest
	// bytes produced through the landed builders.
	RawManifest     []byte
	CaptureManifest []byte
	// WorkspaceBinding carries the sealed workspace checkpoint.
	WorkspaceBinding []byte
	// Decoded forms of the sealed manifests.
	Raw     clonebundle.RawObjectManifest
	Capture clonebundle.CaptureManifest
	// BoundaryKind is stable or unstable_archive, read back from
	// the sealed capture manifest.
	BoundaryKind string
	// PreDigest and PostDigest are the measured source digests.
	PreDigest  scalar.Digest
	PostDigest scalar.Digest
	// RawInstalled and CaptureInstalled report whether this run
	// installed the manifest blobs (a replay verifies and reuses).
	RawInstalled     bool
	CaptureInstalled bool
	// Logs carries the digest/size/media-only capture log lines.
	Logs []string
}

// Capture runs one native capture over the provider store: validate
// the plan, checkpoint the workspace, walk and install payload
// bytes, measure the source race, seal and publish both manifests.
// The race gate precedes the seal (see ORDER-PIN below): a mutation
// refuses, or seals unstable_archive output when archival was
// explicitly acknowledged and requested.
func Capture(request CaptureRequest) (*CaptureResult, error) {
	state, err := newPipeline(request)
	if err != nil {
		return nil, err
	}
	defer state.close()
	if err := state.walkAndMeasure(); err != nil {
		return nil, err
	}
	// ORDER-PIN: the source-race gate precedes any seal or publish.
	// N-race-order-capture moves this gate after sealAndPublish and
	// the killer reddens on the published manifests.
	if err := state.gateSourceRace(); err != nil {
		return state.raceOutcome(err)
	}
	return state.sealAndPublish(false)
}

// pipeline is one running capture: the validated plan, the open
// store handles, the walked bytes, and the measured digests.
type pipeline struct {
	request        CaptureRequest
	log            *CaptureLog
	guard          secprim.Guard
	root           *os.File
	plan           map[string]PlanItem
	planKeys       []string
	limit          uint64
	captured       []capturedMember
	pre            scalar.Digest
	post           scalar.Digest
	preLines       []string
	postLines      []string
	workspaceBytes []byte
}

func newPipeline(request CaptureRequest) (*pipeline, error) {
	if request.Store == nil {
		return nil, invalid("capture requires an object store")
	}
	if request.OnRace != RaceRefuse && request.OnRace != RaceArchive {
		return nil, invalid("capture race policy %q is outside refuse|archive", string(request.OnRace))
	}
	plan, err := checkPlan(request.Platform, request.Plan)
	if err != nil {
		return nil, err
	}
	limit := request.MaxSingleBytes
	if limit == 0 {
		limit = DefaultMaxSingleBytes
	}
	workspaceBytes, err := CheckpointWorkspace(request.Workspace)
	if err != nil {
		return nil, err
	}
	root, err := secprim.OpenNoFollowDir(request.StoreRoot)
	if err != nil {
		return nil, invalid("capture store root %q: %v", request.StoreRoot, err)
	}
	guard, err := secprim.NewGuard(request.StoreRoot, request.Platform, nil)
	if err != nil {
		_ = root.Close()
		return nil, invalid("capture store root %q: %v", request.StoreRoot, err)
	}
	keys := make([]string, 0, len(request.Plan))
	for _, item := range request.Plan {
		keys = append(keys, item.NativeKey)
	}
	return &pipeline{
		request:        request,
		log:            &CaptureLog{},
		guard:          guard,
		root:           root,
		plan:           plan,
		planKeys:       keys,
		limit:          limit,
		workspaceBytes: workspaceBytes,
	}, nil
}

func (state *pipeline) close() {
	if state.root != nil {
		_ = state.root.Close()
		state.root = nil
	}
}

// walkAndMeasure enumerates the store, reconciles every member
// against the plan, opens and installs every included-class member
// through the Guard, and records the pre source digest. Excluded
// members are never opened. It then runs the AfterWalk hook and
// re-measures the post source digest. Any measurement failure
// refuses; mutation itself is decided by gateSourceRace.
func (state *pipeline) walkAndMeasure() error {
	observed, err := enumerateStore(state.request.StoreRoot)
	if err != nil {
		return err
	}
	byKey := make(map[string]observedMember, len(observed))
	for _, member := range observed {
		byKey[member.key] = member
		if member.kind == memberDir {
			continue
		}
		if member.kind == memberOther && intermediatePrefix(state.plan, member.key) {
			continue
		}
		if _, err := planMemberClass(state.plan, member.key); err != nil {
			return err
		}
	}
	// Early containment refusal: every included-class special
	// member (symlink, FIFO, device, socket) reaches the Guard,
	// which refuses it before any payload byte is read. The plan
	// loop would refuse it again; this pass fails fast with nothing
	// installed. Intermediates belong to their planned member's
	// open, which names the full member.
	for _, member := range observed {
		if member.kind != memberOther {
			continue
		}
		if intermediatePrefix(state.plan, member.key) {
			continue
		}
		if excludedClass(state.plan[member.key].Class) {
			continue
		}
		if _, err := openMember(state.guard, state.root, member.key, state.limit); err != nil {
			return err
		}
	}
	for _, key := range state.planKeys {
		item := state.plan[key]
		if excludedClass(item.Class) {
			state.log.Linef("exclude member=%q class=%s reason=%s", key, item.Class, exclusionReason(item.Class))
			continue
		}
		_, present := byKey[key]
		if !present && !item.Required && !blockedAncestor(byKey, key) {
			state.log.Linef("exclude member=%q class=%s reason=plan_optional_absent", key, item.Class)
			continue
		}
		if !present && item.Required && !blockedAncestor(byKey, key) {
			return invalid("capture plan candidate %q is required but absent from the store", key)
		}
		// Every planned included member — file, directory, or
		// special shape — opens through the Guard, which admits
		// regular files and refuses the rest with the member named.
		payload, err := openMember(state.guard, state.root, key, state.limit)
		if err != nil {
			return err
		}
		descriptor, err := BuildBlobDescriptor(payload)
		if err != nil {
			return invalid("capture member %q: %v", key, err)
		}
		if _, err := clonebundle.InstallRawBlob(state.request.Store, descriptor.Descriptor, bytes.NewReader(payload)); err != nil {
			return invalid("capture member %q: %v", key, err)
		}
		state.log.Linef("capture member=%q class=%s digest=%s size=%d media=%s", key, item.Class, descriptor.BlobID.String(), descriptor.Size, blobMediaType)
		state.captured = append(state.captured, capturedMember{key: key, class: item.Class, payload: payload, descriptor: descriptor})
	}
	state.pre, state.preLines = state.recordSource(observed, true)
	if state.request.Hooks.AfterWalk != nil {
		if err := state.request.Hooks.AfterWalk(state.request.StoreRoot); err != nil {
			return invalid("capture hook refused after the walk: %v", err)
		}
	}
	return state.measurePost()
}

// recordSource builds the canonical source record over one
// enumeration: directories contribute keys, specials contribute
// keys, excluded files contribute key and size, and included files
// contribute key, size, and content hash. Production always captured
// every included file it enumerates; the unknown-content fallback
// runs only when a mutant admits an unplanned member past the plan
// gate, and then the post measurement still disagrees.
func (state *pipeline) recordSource(observed []observedMember, useCaptured bool) (scalar.Digest, []string) {
	hashByKey := make(map[string]capturedMember, len(state.captured))
	if useCaptured {
		for _, member := range state.captured {
			hashByKey[member.key] = member
		}
	}
	record := &sourceRecord{}
	for _, member := range observed {
		switch member.kind {
		case memberDir:
			record.addDir(member.key)
		case memberOther:
			record.addOther(member.key)
		case memberFile:
			class := state.plan[member.key].Class
			if !useCaptured {
				if resolved, err := planMemberClass(state.plan, member.key); err == nil {
					class = resolved
				} else {
					class = ""
				}
			}
			if class == "" || excludedClass(class) {
				record.addFile(member.key, member.size, "-")
				continue
			}
			if captured, ok := hashByKey[member.key]; useCaptured && ok {
				record.addFile(member.key, captured.descriptor.Size, contentHash(captured.payload))
				continue
			}
			if !useCaptured {
				if payload, err := openMember(state.guard, state.root, member.key, state.limit); err == nil {
					record.addFile(member.key, uint64(len(payload)), contentHash(payload))
					continue
				}
			}
			record.addFile(member.key, member.size, "-")
		}
	}
	return record.digest(), append([]string(nil), record.lines...)
}

// measurePost re-enumerates the store and re-reads every included
// member through the Guard, recording the post source digest. A
// member that cannot be re-read — appended, removed, retyped, or
// unreadable — joins the record with unknown content, so the digest
// comparison, not the read error, decides the race, and the refusal
// names the member.
func (state *pipeline) measurePost() error {
	observed, err := enumerateStore(state.request.StoreRoot)
	if err != nil {
		return err
	}
	state.post, state.postLines = state.recordSource(observed, false)
	return nil
}

// gateSourceRace decides the measured source race: equal pre and
// post digests proceed; a difference refuses, or marks archive-only
// output when archival was explicitly acknowledged and requested.
// Size equality never decides: the comparison is digest equality
// over content hashes.
func (state *pipeline) gateSourceRace() error {
	if err := checkSourceRace(state.pre, state.post, state.preLines, state.postLines); err != nil {
		return err
	}
	return nil
}

// raceOutcome maps a detected race onto the requested policy: refuse
// aborts with no sealed manifest; archive seals the core-created
// unstable_archive form for archive-only output — never for a
// target branch — and only with explicit operator acknowledgement.
func (state *pipeline) raceOutcome(race error) (*CaptureResult, error) {
	if state.request.OnRace != RaceArchive {
		return nil, race
	}
	if !state.request.OperatorExplicit {
		return nil, invalid("capture archive requires explicit operator acknowledgement for source_not_quiescent")
	}
	return state.sealAndPublish(true)
}

// sealAndPublish seals both manifests through the landed builders,
// re-decodes each before publishing, publishes in raw-then-capture
// order through the object store, and returns the capture result.
// The stable proof seals the two measured source digests: the race
// gate proved them equal before this seal ran, so on the stable
// path the sealed bytes equal either measurement, and the landed
// builder refuses unequal digests. The N-race-order mutants move
// the gate after this seal and the killers redden on the published
// manifests, which proves the gate precedes the seal.
func (state *pipeline) sealAndPublish(unstable bool) (*CaptureResult, error) {
	request := state.request
	entries := make([]clonebundle.EntryInput, 0, len(state.captured))
	included := make([]string, 0, len(state.captured))
	for _, member := range state.captured {
		entries = append(entries, clonebundle.EntryInput{
			NativeItemKey: member.key,
			Class:         member.class,
			ByteCount:     member.descriptor.Size,
			BlobID:        member.descriptor.BlobID.String(),
			DescriptorID:  member.descriptor.DescriptorID.String(),
			Descriptor:    member.descriptor.Descriptor,
		})
		included = append(included, member.key)
	}
	identityDigest, err := clonebundle.IdentityDigest(request.SourceIdentity, nil)
	if err != nil {
		return nil, err
	}
	rawBytes, err := clonebundle.BuildRawObjectManifest(
		request.OperationID,
		request.SourceEnvironment,
		request.NativeSessionID,
		identityDigest.String(),
		request.CapturePlanDigest,
		entries,
		nil,
	)
	if err != nil {
		return nil, err
	}
	raw, err := clonebundle.DecodeRawObjectManifest(rawBytes)
	if err != nil {
		return nil, invalid("sealed raw object manifest refused: %v", err)
	}
	// Manifests publish under their content digest: the manifest ID
	// is the omit-self identity, not the digest of the sealed bytes.
	rawPut, err := request.Store.PutBlob(scalar.SHA256Digest(rawBytes), uint64(len(rawBytes)), bytes.NewReader(rawBytes))
	if err != nil {
		return nil, invalid("capture cannot publish the raw object manifest: %v", err)
	}
	if request.Hooks.AfterRawPublish != nil {
		if err := request.Hooks.AfterRawPublish(); err != nil {
			return nil, invalid("capture hook refused after the raw publish: %v", err)
		}
	}
	boundary := clonebundle.BoundaryInput{
		Kind:              "stable",
		ProofKind:         request.ProofKind,
		Generation:        request.Generation,
		PreCaptureDigest:  state.pre.String(),
		PostCaptureDigest: state.post.String(),
		InputBlocked:      request.InputBlocked,
		ForegroundIdle:    request.ForegroundIdle,
		BackgroundIdle:    request.BackgroundIdle,
	}
	if request.SnapshotIdentity != nil {
		identity := *request.SnapshotIdentity
		boundary.SnapshotIdentity = &identity
	}
	if unstable {
		boundary = clonebundle.BoundaryInput{
			Kind:              "unstable_archive",
			Generation:        request.Generation,
			PreCaptureDigest:  state.pre.String(),
			PostCaptureDigest: state.post.String(),
			Core:              true,
		}
	}
	items := make([]clonebundle.CaptureItemInput, 0, len(state.planKeys))
	capturedByKey := make(map[string]capturedMember, len(state.captured))
	for _, member := range state.captured {
		capturedByKey[member.key] = member
	}
	for _, key := range state.planKeys {
		item := state.plan[key]
		if member, ok := capturedByKey[key]; ok {
			descriptor := member.descriptor.DescriptorID.String()
			count := member.descriptor.Size
			items = append(items, clonebundle.CaptureItemInput{
				NativeItemKey:    key,
				Class:            item.Class,
				Disposition:      "included",
				BlobDescriptorID: &descriptor,
				ByteCount:        &count,
			})
			continue
		}
		reason := exclusionReason(item.Class)
		if !excludedClass(item.Class) {
			reason = "plan_optional_absent"
		}
		items = append(items, clonebundle.CaptureItemInput{
			NativeItemKey:   key,
			Class:           item.Class,
			Disposition:     "excluded",
			ExclusionReason: &reason,
		})
	}
	captureBytes, err := clonebundle.BuildCaptureManifest(clonebundle.CaptureManifestInput{
		OperationID:         request.OperationID,
		BundleID:            request.BundleID,
		SourceBasis:         request.SourceBasis,
		SourceEnvironment:   request.SourceEnvironment,
		SourceIdentity:      request.SourceIdentity,
		CapturePlanDigest:   request.CapturePlanDigest,
		Boundary:            boundary,
		SourceRawManifestID: raw.ManifestID.String(),
		Items:               items,
		PlanKeys:            append([]string(nil), state.planKeys...),
		RawKeys:             included,
		CreatedByHostID:     request.CreatedByHostID,
		CreatedAt:           request.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	manifest, err := clonebundle.DecodeCaptureManifest(captureBytes)
	if err != nil {
		return nil, invalid("sealed capture manifest refused: %v", err)
	}
	capturePut, err := request.Store.PutBlob(scalar.SHA256Digest(captureBytes), uint64(len(captureBytes)), bytes.NewReader(captureBytes))
	if err != nil {
		return nil, invalid("capture cannot publish the capture manifest: %v", err)
	}
	boundaryKind := "stable"
	if !manifest.CaptureBoundary.Stable() {
		boundaryKind = "unstable_archive"
	}
	state.log.Linef("boundary kind=%s pre=%s post=%s", boundaryKind, state.pre.String(), state.post.String())
	return &CaptureResult{
		RawManifest:      rawBytes,
		CaptureManifest:  captureBytes,
		WorkspaceBinding: append([]byte(nil), state.workspaceBytes...),
		Raw:              raw,
		Capture:          manifest,
		BoundaryKind:     boundaryKind,
		PreDigest:        state.pre,
		PostDigest:       state.post,
		RawInstalled:     rawPut.Installed,
		CaptureInstalled: capturePut.Installed,
		Logs:             state.log.Lines(),
	}, nil
}
