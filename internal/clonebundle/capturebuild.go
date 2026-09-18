package clonebundle

import (
	"encoding/json"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Clone Capture Manifest 1.0.0:
// the closed envelope over source basis, capture boundary, sorted
// Capture Items, exact excluded classes, and the core-derived
// raw_complete flag.

const (
	captureManifestSchema  = "urn:ax:schema:clone-capture-manifest"
	captureManifestVersion = "1.0.0"
	captureManifestSelf    = "capture_manifest_id"
)

var captureManifestMembers = map[string]bool{
	"schema": true, "schema_version": true, "capture_manifest_id": true,
	"operation_id": true, "bundle_id": true, "source_basis": true,
	"source_environment": true, "source_identity": true,
	"capture_plan_digest": true, "capture_boundary": true,
	"source_raw_object_manifest_id": true, "items": true,
	"excluded_classes": true, "raw_complete": true,
	"created_by_host_id": true, "created_at": true, "extensions": true,
}

var captureManifestRequired = []string{
	"schema", "schema_version", "capture_manifest_id",
	"operation_id", "bundle_id", "source_basis",
	"source_environment", "source_identity",
	"capture_plan_digest", "capture_boundary",
	"source_raw_object_manifest_id", "items",
	"excluded_classes", "raw_complete",
	"created_by_host_id", "created_at", "extensions",
}

// CaptureManifest is one validated Clone Capture Manifest.
type CaptureManifest struct {
	ManifestID              scalar.Digest
	OperationID             scalar.UUIDv7
	BundleID                scalar.UUIDv7
	SourceBasis             SourceBasis
	SourceEnvironment       sessadapter.Tuple
	SourceIdentity          NativeIdentity
	CapturePlanDigest       scalar.Digest
	CaptureBoundary         CaptureBoundary
	SourceRawObjectManifest scalar.Digest
	Items                   []CaptureItem
	ExcludedClasses         []string
	RawComplete             bool
	CreatedByHostID         scalar.UUIDv7
	CreatedAt               scalar.Timestamp
}

// CaptureManifestInput is the caller-supplied capture manifest
// candidate for Build. PlanKeys are the exact capture-plan
// candidate keys and RawKeys are the exact raw-manifest entry keys;
// raw_complete is core-derived from them and is not an input — a
// caller cannot assert completeness, only the reconciliation can.
type CaptureManifestInput struct {
	OperationID         string
	BundleID            string
	SourceBasis         SourceBasisInput
	SourceEnvironment   []byte
	SourceIdentity      NativeIdentity
	IdentityExtensions  map[string]any
	CapturePlanDigest   string
	Boundary            BoundaryInput
	SourceRawManifestID string
	Items               []CaptureItemInput
	PlanKeys            []string
	RawKeys             []string
	CreatedByHostID     string
	CreatedAt           string
	Extensions          map[string]any
}

// BuildCaptureManifest constructs one closed Clone Capture Manifest
// as canonical bytes. Identical inputs produce byte-identical
// outputs. raw_complete is derived, never accepted: it is true only
// after complete plan/object reconciliation with no unknown class.
func BuildCaptureManifest(input CaptureManifestInput) ([]byte, error) {
	_, object, err := buildCaptureManifest(input)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	manifestID := scalar.SHA256Digest(omitted)
	object[captureManifestSelf] = manifestID.String()
	return canonicalizeObject(object)
}

func buildCaptureManifest(input CaptureManifestInput) (CaptureManifest, map[string]any, error) {
	operation, err := scalar.ParseUUIDv7(input.OperationID)
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest operation_id is not a UUIDv7: %v", err)
	}
	bundle, err := scalar.ParseUUIDv7(input.BundleID)
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest bundle_id is not a UUIDv7: %v", err)
	}
	basis, basisObject, err := buildSourceBasis(input.SourceBasis)
	if err != nil {
		return CaptureManifest{}, nil, err
	}
	tuple, err := sessadapter.DecodeTuple(json.RawMessage(input.SourceEnvironment))
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest source_environment is not an Environment Tuple: %v", err)
	}
	identityBytes, err := EncodeNativeIdentity(input.SourceIdentity, input.IdentityExtensions)
	if err != nil {
		return CaptureManifest{}, nil, err
	}
	if _, err := DecodeNativeIdentity(json.RawMessage(identityBytes)); err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest source_identity is not a NativeIdentity: %v", err)
	}
	planDigest, err := scalar.ParseDigest(input.CapturePlanDigest)
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest capture_plan_digest is not a digest: %v", err)
	}
	boundary, boundaryObject, err := buildBoundary(input.Boundary)
	if err != nil {
		return CaptureManifest{}, nil, err
	}
	rawManifestID, err := scalar.ParseDigest(input.SourceRawManifestID)
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest source_raw_object_manifest_id is not a digest: %v", err)
	}
	if len(input.Items) > 65536 {
		return CaptureManifest{}, nil, invalid("capture manifest carries %d items, maximum is 65536", len(input.Items))
	}
	items := make([]CaptureItem, 0, len(input.Items))
	previous := ""
	for index, candidate := range input.Items {
		item, err := buildCaptureItem(candidate, index)
		if err != nil {
			return CaptureManifest{}, nil, err
		}
		if index > 0 && item.NativeItemKey <= previous {
			return CaptureManifest{}, nil, invalid("capture manifest items are not sorted bytewise by native item key")
		}
		previous = item.NativeItemKey
		items = append(items, item)
	}
	if err := checkPlanItemMatch(items, input.PlanKeys); err != nil {
		return CaptureManifest{}, nil, err
	}
	excluded := excludedRowClasses(items)
	rawComplete := deriveRawComplete(items, input.PlanKeys, input.RawKeys)
	hostID, err := scalar.ParseUUIDv7(input.CreatedByHostID)
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest created_by_host_id is not a UUIDv7: %v", err)
	}
	createdAt, err := scalar.ParseTimestamp(input.CreatedAt)
	if err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest created_at is not a timestamp: %v", err)
	}
	if _, err := encodeExtensions(input.Extensions); err != nil {
		return CaptureManifest{}, nil, err
	}
	var identityValue any
	if err := json.Unmarshal(identityBytes, &identityValue); err != nil {
		return CaptureManifest{}, nil, invalid("capture manifest source_identity is not JSON: %v", err)
	}
	objects := make([]any, 0, len(items))
	for _, item := range items {
		objects = append(objects, captureItemObject(item))
	}
	excludedValue := make([]any, 0, len(excluded))
	for _, class := range excluded {
		excludedValue = append(excludedValue, class)
	}
	manifest := CaptureManifest{
		OperationID:             operation,
		BundleID:                bundle,
		SourceBasis:             basis,
		SourceEnvironment:       tuple,
		SourceIdentity:          input.SourceIdentity,
		CapturePlanDigest:       planDigest,
		CaptureBoundary:         boundary,
		SourceRawObjectManifest: rawManifestID,
		Items:                   items,
		ExcludedClasses:         excluded,
		RawComplete:             rawComplete,
		CreatedByHostID:         hostID,
		CreatedAt:               createdAt,
	}
	object := map[string]any{
		"schema":                        captureManifestSchema,
		"schema_version":                captureManifestVersion,
		"operation_id":                  operation.String(),
		"bundle_id":                     bundle.String(),
		"source_basis":                  basisObject,
		"source_environment":            tupleObject(tuple),
		"source_identity":               identityValue,
		"capture_plan_digest":           planDigest.String(),
		"capture_boundary":              boundaryObject,
		"source_raw_object_manifest_id": rawManifestID.String(),
		"items":                         objects,
		"excluded_classes":              excludedValue,
		"raw_complete":                  rawComplete,
		"created_by_host_id":            hostID.String(),
		"created_at":                    createdAt.String(),
		"extensions":                    extensionValue(input.Extensions),
	}
	return manifest, object, nil
}

// DecodeCaptureManifest validates one closed Clone Capture Manifest:
// exact members, schema/version, self-digest agreement, sorted
// items, exact excluded classes, and the unknown-makes-incomplete
// rule. Full plan/object reconciliation needs the plan and raw key
// sets; VerifyCaptureReconciliation performs it.
func DecodeCaptureManifest(data []byte) (CaptureManifest, error) {
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return CaptureManifest{}, invalid("capture manifest %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, captureManifestMembers); unknown {
		return CaptureManifest{}, invalid("capture manifest carries unknown member %q", name)
	}
	if name, missing := missingMember(members, captureManifestRequired); missing {
		return CaptureManifest{}, invalid("capture manifest misses a required member %q", name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != captureManifestSchema {
		return CaptureManifest{}, invalid("capture manifest schema is not the clone capture manifest")
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != captureManifestVersion {
		return CaptureManifest{}, invalid("capture manifest version is not 1.0.0")
	}
	manifestID, ok := checkDigest(members[captureManifestSelf])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest capture_manifest_id is not a digest")
	}
	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest operation_id is not a UUIDv7")
	}
	bundle, ok := checkUUIDv7(members["bundle_id"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest bundle_id is not a UUIDv7")
	}
	basis, err := DecodeSourceBasis(members["source_basis"])
	if err != nil {
		return CaptureManifest{}, err
	}
	tuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil {
		return CaptureManifest{}, invalid("capture manifest source_environment is not an Environment Tuple: %v", err)
	}
	identity, err := DecodeNativeIdentity(members["source_identity"])
	if err != nil {
		return CaptureManifest{}, err
	}
	planDigest, ok := checkDigest(members["capture_plan_digest"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest capture_plan_digest is not a digest")
	}
	boundary, err := DecodeCaptureBoundary(members["capture_boundary"])
	if err != nil {
		return CaptureManifest{}, err
	}
	rawManifestID, ok := checkDigest(members["source_raw_object_manifest_id"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest source_raw_object_manifest_id is not a digest")
	}
	items, err := decodeCaptureItems(members["items"])
	if err != nil {
		return CaptureManifest{}, err
	}
	excluded, ok := checkSortedUniqueStrings(members["excluded_classes"], 1, 128, 0, 9)
	if !ok {
		return CaptureManifest{}, invalid("capture manifest excluded_classes are not sorted unique string[1..128][0..9]")
	}
	for _, class := range excluded {
		if !ValidCaptureClass(class) {
			return CaptureManifest{}, invalid("capture manifest excluded_classes carry unknown class %q", class)
		}
	}
	if !excludedClassesExact(items, excluded) {
		return CaptureManifest{}, invalid("capture manifest excluded_classes are not exactly the excluded-row classes")
	}
	rawComplete, ok := rawBool(members["raw_complete"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest raw_complete is not a boolean")
	}
	if rawComplete && hasUnknownClass(items) {
		return CaptureManifest{}, invalid("capture manifest raw_complete is true with an unknown-class item")
	}
	hostID, ok := checkUUIDv7(members["created_by_host_id"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest created_by_host_id is not a UUIDv7")
	}
	createdAt, ok := checkTimestamp(members["created_at"])
	if !ok {
		return CaptureManifest{}, invalid("capture manifest created_at is not a timestamp")
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return CaptureManifest{}, invalid("capture manifest extensions %s", extensionsFault)
	}
	if err := verifySelfDigest(members, captureManifestSelf, manifestID); err != nil {
		return CaptureManifest{}, err
	}
	return CaptureManifest{
		ManifestID:              manifestID,
		OperationID:             operation,
		BundleID:                bundle,
		SourceBasis:             basis,
		SourceEnvironment:       tuple,
		SourceIdentity:          identity,
		CapturePlanDigest:       planDigest,
		CaptureBoundary:         boundary,
		SourceRawObjectManifest: rawManifestID,
		Items:                   items,
		ExcludedClasses:         excluded,
		RawComplete:             rawComplete,
		CreatedByHostID:         hostID,
		CreatedAt:               createdAt,
	}, nil
}

func decodeCaptureItems(raw json.RawMessage) ([]CaptureItem, error) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, invalid("capture manifest items are not an array")
	}
	if len(elements) > 65536 {
		return nil, invalid("capture manifest carries %d items, maximum is 65536", len(elements))
	}
	items := make([]CaptureItem, 0, len(elements))
	previous := ""
	for index, element := range elements {
		item, err := decodeCaptureItem(element, index)
		if err != nil {
			return nil, err
		}
		if index > 0 && item.NativeItemKey <= previous {
			return nil, invalid("capture manifest items are not sorted bytewise by native item key")
		}
		previous = item.NativeItemKey
		items = append(items, item)
	}
	return items, nil
}

// checkPlanItemMatch enforces the one-per-plan-candidate rule at
// construction: the sealed items must be exactly one per capture-plan
// candidate, no missing and no extra. An incomplete raw closure still
// seals (with raw_complete false), but an items/plan mismatch is a
// malformed manifest, never an archive object. Duplicate plan keys
// are refused outright: a repeated key would defeat the count and
// hide an item that is not a plan candidate.
func checkPlanItemMatch(items []CaptureItem, planKeys []string) error {
	if len(items) != len(planKeys) {
		return invalid("capture manifest carries %d items for %d plan candidates, want one per plan candidate", len(items), len(planKeys))
	}
	seenPlan := map[string]bool{}
	for _, key := range planKeys {
		if seenPlan[key] {
			return invalid("capture manifest plan keys carry duplicate %q", key)
		}
		seenPlan[key] = true
	}
	itemSet := map[string]bool{}
	for _, item := range items {
		itemSet[item.NativeItemKey] = true
	}
	for _, key := range planKeys {
		if !itemSet[key] {
			return invalid("capture manifest plan candidate %q has no item", key)
		}
	}
	return nil
}

// excludedRowClasses derives the exact excluded-row class set:
// sorted unique classes of excluded rows.
func excludedRowClasses(items []CaptureItem) []string {
	seen := map[string]bool{}
	for _, item := range items {
		if !item.Included() {
			seen[item.Class] = true
		}
	}
	classes := make([]string, 0, len(seen))
	for class := range seen {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	return classes
}

func excludedClassesExact(items []CaptureItem, excluded []string) bool {
	derived := excludedRowClasses(items)
	if len(derived) != len(excluded) {
		return false
	}
	for index := range derived {
		if derived[index] != excluded[index] {
			return false
		}
	}
	return true
}

func hasUnknownClass(items []CaptureItem) bool {
	for _, item := range items {
		if item.Class == "unknown" {
			return true
		}
	}
	return false
}

// deriveRawComplete is the core raw_complete derivation: true only
// when no item has unknown class, every plan candidate reconciles
// to exactly one item, and the included items are exactly the raw
// manifest keys (all and only included objects).
func deriveRawComplete(items []CaptureItem, planKeys, rawKeys []string) bool {
	if hasUnknownClass(items) {
		return false
	}
	if len(items) != len(planKeys) {
		return false
	}
	planSet := map[string]bool{}
	for _, key := range planKeys {
		planSet[key] = true
	}
	if len(planSet) != len(planKeys) {
		return false
	}
	included := map[string]bool{}
	for _, item := range items {
		if !planSet[item.NativeItemKey] {
			return false
		}
		if item.Included() {
			included[item.NativeItemKey] = true
		}
	}
	if len(included) != len(rawKeys) {
		return false
	}
	seenRaw := map[string]bool{}
	for _, key := range rawKeys {
		if !included[key] {
			return false
		}
		if seenRaw[key] {
			return false
		}
		seenRaw[key] = true
	}
	return true
}

// VerifyCaptureReconciliation re-checks complete plan/object
// reconciliation for a decoded manifest: one item per plan
// candidate, the raw closure over exactly the included items, and a
// raw_complete flag that matches the derivation. Unknown classes
// always reconcile to false. Duplicate raw keys are refused
// outright: a duplicate would defeat the "all" half of the all-and-only
// closure, hiding a missing included object behind a repeated key.
func VerifyCaptureReconciliation(manifest CaptureManifest, planKeys, rawKeys []string) error {
	seenRaw := map[string]bool{}
	for _, key := range rawKeys {
		if seenRaw[key] {
			return invalid("capture manifest raw keys carry duplicate %q", key)
		}
		seenRaw[key] = true
	}
	derived := deriveRawComplete(manifest.Items, planKeys, rawKeys)
	if manifest.RawComplete != derived {
		return invalid("capture manifest raw_complete is %v, reconciliation derives %v", manifest.RawComplete, derived)
	}
	if !derived {
		return invalid("capture manifest plan/object reconciliation is incomplete")
	}
	return nil
}

// RefuseMaximalSafeUnlessComplete blocks the maximal_safe fidelity
// profile: unknown classes block it, and it requires a reconciled
// complete manifest. Projection planning calls this gate.
func RefuseMaximalSafeUnlessComplete(manifest CaptureManifest) error {
	if hasUnknownClass(manifest.Items) {
		return invalid("maximal_safe is blocked by an unknown-class item")
	}
	if !manifest.RawComplete {
		return invalid("maximal_safe requires raw_complete")
	}
	return nil
}
