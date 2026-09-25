package clonefidelity

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates and constructs FidelityDispositionRecord: the
// closed per-item row Section 13.14.2 states. Exact requires an
// empty reason set and every other disposition at least one reason;
// synthesized requires no source canonical object; every
// non-synthesized row carries captured source evidence IDs (their
// resolution against the Capture Manifest needs the manifest as an
// input and is a stated bound).

var dispositionRecordMembers = map[string]bool{
	"source_item_key":            true,
	"source_class":               true,
	"source_evidence_ids":        true,
	"canonical_object_id":        true,
	"target_locator":             true,
	"disposition":                true,
	"reason_codes":               true,
	"explanation":                true,
	"staged_evidence_object_ids": true,
	"live_evidence_object_ids":   true,
	"extensions":                 true,
}

var dispositionRecordRequired = []string{
	"source_item_key",
	"source_class",
	"source_evidence_ids",
	"canonical_object_id",
	"target_locator",
	"disposition",
	"reason_codes",
	"explanation",
	"staged_evidence_object_ids",
	"live_evidence_object_ids",
	"extensions",
}

// DispositionRecord is one validated fidelity disposition row.
type DispositionRecord struct {
	SourceItemKey           string
	SourceClass             string
	SourceEvidenceIDs       []scalar.Digest
	CanonicalObjectID       *scalar.Digest
	TargetLocator           *string
	Disposition             string
	ReasonCodes             []string
	Explanation             string
	StagedEvidenceObjectIDs []scalar.Digest
	LiveEvidenceObjectIDs   []scalar.Digest
}

// DispositionRecordInput is the caller-supplied disposition row
// candidate for Build. Nil CanonicalObjectID and TargetLocator seal
// as null.
type DispositionRecordInput struct {
	SourceItemKey           string
	SourceClass             string
	SourceEvidenceIDs       []string
	CanonicalObjectID       *string
	TargetLocator           *string
	Disposition             string
	ReasonCodes             []string
	Explanation             string
	StagedEvidenceObjectIDs []string
	LiveEvidenceObjectIDs   []string
	Extensions              map[string]any
}

// ValidateDispositionRow validates one caller-supplied fidelity
// disposition row against every Section 13.14.2 record rule the
// Build and Decode entries share: the disposition and reason
// vocabularies, the string and count bounds, sorted-unique arrays,
// the exact/non-exact reason-set coupling ("Exact requires an
// empty reason set; every other disposition requires at least one
// reason"), and the synthesized canonical-object rule. It is the
// single-owner seam sibling packages call instead of
// re-implementing any subset: it delegates to the same builder
// the report entry uses, discarding the sealed row and object.
func ValidateDispositionRow(input DispositionRecordInput) error {
	_, _, err := buildDispositionRecord(input, 0)
	return err
}

// buildDispositionRecord validates one caller-supplied row and
// renders its closed object. Every rule is a refusal: unknown
// dispositions, out-of-vocabulary reasons, broken bounds, unsorted
// or duplicate arrays, exact with reasons, non-exact without
// reasons, and synthesized with a source canonical object are all
// refused, never repaired.
func buildDispositionRecord(input DispositionRecordInput, index int) (DispositionRecord, map[string]any, error) {
	owner := fmt.Sprintf("fidelity disposition record[%d]", index)
	if !validText(input.SourceItemKey) {
		return DispositionRecord{}, nil, invalid("%s source_item_key is not valid UTF-8", owner)
	}
	if length := stringLength(input.SourceItemKey); length < 1 || length > 512 {
		return DispositionRecord{}, nil, invalid("%s source_item_key is not a string[1..512]", owner)
	}
	if !validText(input.SourceClass) {
		return DispositionRecord{}, nil, invalid("%s source_class is not valid UTF-8", owner)
	}
	if length := stringLength(input.SourceClass); length < 1 || length > 128 {
		return DispositionRecord{}, nil, invalid("%s source_class is not a string[1..128]", owner)
	}
	evidence, err := parseSortedUniqueDigestStrings(input.SourceEvidenceIDs, 1, 65536, owner, "source_evidence_ids")
	if err != nil {
		return DispositionRecord{}, nil, err
	}
	var canonical *scalar.Digest
	var canonicalValue any
	if input.CanonicalObjectID != nil {
		id, err := scalar.ParseDigest(*input.CanonicalObjectID)
		if err != nil {
			return DispositionRecord{}, nil, invalid("%s canonical_object_id is not a digest: %v", owner, err)
		}
		canonical = &id
		canonicalValue = id.String()
	}
	var locator *string
	var locatorValue any
	if input.TargetLocator != nil {
		if !validText(*input.TargetLocator) {
			return DispositionRecord{}, nil, invalid("%s target_locator is not valid UTF-8", owner)
		}
		if length := stringLength(*input.TargetLocator); length < 1 || length > 1024 {
			return DispositionRecord{}, nil, invalid("%s target_locator is not a string[1..1024]", owner)
		}
		locator = input.TargetLocator
		locatorValue = *input.TargetLocator
	}
	if !ValidDisposition(input.Disposition) {
		return DispositionRecord{}, nil, invalid("%s disposition %q is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, input.Disposition)
	}
	reasons, err := checkSortedUniqueReasonStrings(input.ReasonCodes, 0, 128, owner)
	if err != nil {
		return DispositionRecord{}, nil, err
	}
	if err := checkRecordReasonRule(owner, input.Disposition, len(reasons)); err != nil {
		return DispositionRecord{}, nil, err
	}
	if err := checkRecordCanonicalRule(owner, input.Disposition, canonical != nil); err != nil {
		return DispositionRecord{}, nil, err
	}
	if !validText(input.Explanation) {
		return DispositionRecord{}, nil, invalid("%s explanation is not valid UTF-8", owner)
	}
	if length := stringLength(input.Explanation); length < 1 || length > 4096 {
		return DispositionRecord{}, nil, invalid("%s explanation is not a string[1..4096]", owner)
	}
	staged, err := parseSortedUniqueDigestStrings(input.StagedEvidenceObjectIDs, 0, 65536, owner, "staged_evidence_object_ids")
	if err != nil {
		return DispositionRecord{}, nil, err
	}
	live, err := parseSortedUniqueDigestStrings(input.LiveEvidenceObjectIDs, 0, 65536, owner, "live_evidence_object_ids")
	if err != nil {
		return DispositionRecord{}, nil, err
	}
	if _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {
		return DispositionRecord{}, nil, invalid("%s extensions invalid: %v", owner, err)
	}
	extensions := input.Extensions
	if extensions == nil {
		extensions = map[string]any{}
	}
	record := DispositionRecord{
		SourceItemKey:           input.SourceItemKey,
		SourceClass:             input.SourceClass,
		SourceEvidenceIDs:       evidence,
		CanonicalObjectID:       canonical,
		TargetLocator:           locator,
		Disposition:             input.Disposition,
		ReasonCodes:             reasons,
		Explanation:             input.Explanation,
		StagedEvidenceObjectIDs: staged,
		LiveEvidenceObjectIDs:   live,
	}
	object := map[string]any{
		"source_item_key":            input.SourceItemKey,
		"source_class":               input.SourceClass,
		"source_evidence_ids":        digestStrings(evidence),
		"canonical_object_id":        canonicalValue,
		"target_locator":             locatorValue,
		"disposition":                input.Disposition,
		"reason_codes":               stringValues(reasons),
		"explanation":                input.Explanation,
		"staged_evidence_object_ids": digestStrings(staged),
		"live_evidence_object_ids":   digestStrings(live),
		"extensions":                 extensions,
	}
	return record, object, nil
}

// checkRecordReasonRule enforces the reason-set rule both entries
// share: exact requires an empty reason set and every other
// disposition at least one reason.
func checkRecordReasonRule(owner, disposition string, reasonCount int) error {
	if disposition == "exact" {
		if reasonCount != 0 {
			return invalid("%s exact carries %d reason_codes, want an empty reason set", owner, reasonCount)
		}
		return nil
	}
	if reasonCount < 1 {
		return invalid("%s %s carries no reason_codes, want at least one reason", owner, disposition)
	}
	return nil
}

// checkRecordCanonicalRule enforces the canonical-object rule both
// entries share: synthesized requires no source canonical object.
// Non-synthesized rows admit a digest or null: capture items that
// never normalized (omitted, unrecoverable) carry null, canonical
// events carry their object ID.
func checkRecordCanonicalRule(owner, disposition string, hasCanonical bool) error {
	if disposition == "synthesized" && hasCanonical {
		return invalid("%s synthesized carries a source canonical object", owner)
	}
	return nil
}

// decodeDispositionRecord validates one closed disposition row:
// exact members, bounded scalars, the disposition and reason
// vocabularies, sorted-unique arrays, the exact/synthesized rules,
// and closed extensions.
func decodeDispositionRecord(raw json.RawMessage, index int) (DispositionRecord, error) {
	owner := "fidelity disposition record"
	members, fault := decodeStrictObject(raw)
	if fault != nil {
		return DispositionRecord{}, invalid("%s[%d] %s (%s)", owner, index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, dispositionRecordMembers); unknown {
		return DispositionRecord{}, invalid("%s[%d] carries unknown member %q", owner, index, name)
	}
	if name, missing := missingMember(members, dispositionRecordRequired); missing {
		return DispositionRecord{}, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	key, ok := checkStringBounds(members["source_item_key"], 1, 512)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] source_item_key is not a string[1..512]", owner, index)
	}
	class, ok := checkStringBounds(members["source_class"], 1, 128)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] source_class is not a string[1..128]", owner, index)
	}
	evidence, ok := checkSortedUniqueDigests(members["source_evidence_ids"], 1, 65536)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] source_evidence_ids are not sorted unique digest[1..65536]", owner, index)
	}
	var canonical *scalar.Digest
	if !isNull(members["canonical_object_id"]) {
		id, ok := checkDigest(members["canonical_object_id"])
		if !ok {
			return DispositionRecord{}, invalid("%s[%d] canonical_object_id is not a digest or null", owner, index)
		}
		canonical = &id
	}
	var locator *string
	if !isNull(members["target_locator"]) {
		value, ok := checkStringBounds(members["target_locator"], 1, 1024)
		if !ok {
			return DispositionRecord{}, invalid("%s[%d] target_locator is not a string[1..1024] or null", owner, index)
		}
		locator = &value
	}
	disposition, ok := rawString(members["disposition"])
	if !ok || !ValidDisposition(disposition) {
		return DispositionRecord{}, invalid("%s[%d] disposition is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable", owner, index)
	}
	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] reason_codes are not sorted unique string[1..128][0..128]", owner, index)
	}
	for _, reason := range reasons {
		if !ValidReasonCode(reason) {
			return DispositionRecord{}, invalid("%s[%d] reason_codes carry %q outside the core and reverse-DNS reason vocabulary", owner, index, reason)
		}
	}
	rowOwner := fmt.Sprintf("%s[%d]", owner, index)
	if err := checkRecordReasonRule(rowOwner, disposition, len(reasons)); err != nil {
		return DispositionRecord{}, err
	}
	if err := checkRecordCanonicalRule(rowOwner, disposition, canonical != nil); err != nil {
		return DispositionRecord{}, err
	}
	explanation, ok := checkStringBounds(members["explanation"], 1, 4096)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] explanation is not a string[1..4096]", owner, index)
	}
	staged, ok := checkSortedUniqueDigests(members["staged_evidence_object_ids"], 0, 65536)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] staged_evidence_object_ids are not sorted unique digest[0..65536]", owner, index)
	}
	live, ok := checkSortedUniqueDigests(members["live_evidence_object_ids"], 0, 65536)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] live_evidence_object_ids are not sorted unique digest[0..65536]", owner, index)
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return DispositionRecord{}, invalid("%s[%d] extensions %s", owner, index, extensionsFault)
	}
	return DispositionRecord{
		SourceItemKey:           key,
		SourceClass:             class,
		SourceEvidenceIDs:       evidence,
		CanonicalObjectID:       canonical,
		TargetLocator:           locator,
		Disposition:             disposition,
		ReasonCodes:             reasons,
		Explanation:             explanation,
		StagedEvidenceObjectIDs: staged,
		LiveEvidenceObjectIDs:   live,
	}, nil
}
