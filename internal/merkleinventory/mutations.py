#!/usr/bin/env python3
"""Run isolated narrowing overlays for the namespace inventory gates."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
PROBES = []
TOKEN_PRESERVING_IDENTITY_MUTANT = "identity-token-preserving-refusal-bypass"
INVENTORY_ALLOWLIST_MUTANT = "identity-inventory-owner-prefix"


T = "internal/merkleinventory/trie.go"
I = "internal/merkleinventory/index.go"
S = "internal/merkleinventory/serve.go"
C = "internal/canonicaljson/tombstones.go"
SHAPES = "internal/canonicaljson/closed_shapes.go"
SYNC_COMMON_AUDIT = '''if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}'''

def probe(name, path, old, new, test, regex, narrowing, package="./internal/merkleinventory"):
    PROBES.append({"name": name, "path": path, "old": old, "new": new,
                   "test": test, "regex": regex, "narrowing": narrowing,
                   "package": package})


probe("control-neutral-comment", T,
      "// ComputeNode is the shared Section 11.4 node construction entry.",
      "// ComputeNode is the shared deterministic Section 11.4 node construction entry.",
      "TestDispatchServesNormativeChildrenBody", "^TestDispatchServesNormativeChildrenBody$",
      "Equivalent comment-only edit; this applied control must survive.")
probe("rule1-duplicate-set-member", T,
      "ids = slices.Compact(ids)",
      "if len(ids) > 1 && ids[0] == ids[1] { ids = append(ids, ids[1]) } else { ids = slices.Compact(ids) }",
      "TestNodeRuleShapesAndDepth64IntegrityFailure", "^TestNodeRuleShapesAndDepth64IntegrityFailure$",
      "Retains a second copy of one duplicate ID in the node set.")
probe("rule2-empty-root-count", T,
      "scalar.NewUint53(uint64(len(ids)))", "scalar.NewUint53(uint64(len(ids) + 1))",
      "TestDispatchServesNormativeChildrenBody", "^TestDispatchServesNormativeChildrenBody$/^empty$",
      "Admits one phantom member into the empty-root count.")
probe("rule3-root-singleton-leaf", T,
      "if len(ids) == 1 {", "if len(ids) == 1 && prefix != \"\" {",
      "TestDispatchServesNormativeChildrenBody", "^TestDispatchServesNormativeChildrenBody$/^singleton$",
      "Stops encoding a singleton as a leaf at the root prefix.")
probe("rule4-retain-single-child", T,
      "} else if len(ids) > 1 {", "} else if len(ids) > 1 && prefix == \"\" {",
      "TestRuleFourRetainsSingleChildAtEverySharedPrefix", "^TestRuleFourRetainsSingleChildAtEverySharedPrefix$",
      "Drops a required one-child node below the root.")
probe("rule5-depth64-duplicate", T,
      "len(prefix) == 64 && len(matching) > 1", "len(prefix) == 64 && len(matching) > 2",
      "TestNodeRuleShapesAndDepth64IntegrityFailure", "^TestNodeRuleShapesAndDepth64IntegrityFailure$",
      "Admits two entries at a full 64-nibble prefix.")
probe("children-label-order", T,
      "sort.Slice(labels, func(a, b int) bool { return labels[a] < labels[b] })",
      "sort.Slice(labels, func(a, b int) bool { return labels[a] > labels[b] })",
      "TestDispatchServesNormativeChildrenBody", "^TestDispatchServesNormativeChildrenBody$/^branch$",
      "Reverses the specified ascending nibble-label order in the served branch body.")
probe("prefix-length-65", T, "if len(prefix) > 64 {", "if len(prefix) > 65 {",
      "TestDispatchPrefixAxesAndNotFound", "^TestDispatchPrefixAxesAndNotFound$/^length-65$",
      "Admits a 65-nibble prefix.")
probe("uppercase-prefix", T,
      "if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {",
      "if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {",
      "TestDispatchPrefixAxesAndNotFound", "^TestDispatchPrefixAxesAndNotFound$/^uppercase$",
      "Admits uppercase A–F in a prefix.")
probe("missing-prefix-not-found", S,
      'if prefix != "" && node.Count.Uint64() == 0 {',
      'if prefix != "" && node.Count.Uint64() == 0 && prefix != "f" {',
      "TestDispatchPrefixAxesAndNotFound", "^TestDispatchPrefixAxesAndNotFound$/^valid-missing-prefix$",
      "Returns an invented empty body for the valid missing prefix f.")
probe("children-prefix-type-null", "internal/merkleinventory/strict_json.go",
      '''if trimmed[0] != '"' {''',
      '''if trimmed[0] != '"' && !bytes.Equal(trimmed, []byte("null")) {''',
      "TestDispatchRequestShapeStrictTypes", "^TestDispatchRequestShapeStrictTypes$/^inventory_children_prefix_null$",
      "Admits JSON null as the empty root prefix through encoding/json's string zero value.")
probe("server-casefold-prefix", S,
      '''object, err := strictObject(requestBody)
	if err != nil || !exactMembers(object, "namespace", "prefix") {''',
      '''object, err := strictObject(requestBody)
	if err == nil {
		if raw, ok := object["PREFIX"]; ok {
			object["prefix"] = raw
			delete(object, "PREFIX")
		}
	}
	if err != nil || !exactMembers(object, "namespace", "prefix") {''',
      "TestDispatchRejectsEveryNonExactOperationBodyShape",
      "^TestDispatchRejectsEveryNonExactOperationBodyShape$/^inventory_children_prefix_casefold$",
      "Admits the miscased Prefix member at the server inventory.children entry.")
probe("client-casefold-wireobject", S,
      '''inner, err := strictObject(rawObjects[0])
	if err != nil || !exactMembers(inner, "object_id", "media_type", "encoding", "data") {''',
      '''inner, err := strictObject(rawObjects[0])
	if err == nil {
		if raw, ok := inner["OBJECT_ID"]; ok {
			inner["object_id"] = raw
			delete(inner, "OBJECT_ID")
		}
	}
	if err != nil || !exactMembers(inner, "object_id", "media_type", "encoding", "data") {''',
      "TestFetchObjectsRejectsEveryNonExactSuccessWireShape",
      "^TestFetchObjectsRejectsEveryNonExactSuccessWireShape$/^wireobject_object_id_casefold$",
      "Admits one case-folded WireObject member at the hostile client response entry.")
probe("client-two-wireobjects", S,
      "if err != nil || len(rawObjects) != 1 {",
      "if err != nil || len(rawObjects) != 1 && len(rawObjects) != 2 {",
      "TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality",
      "^TestFetchObjectsRejectsNonObjectItemsAndWrongArrayCardinality$/^two_items$",
      "Accepts two WireObjects for the one-ID singleton FetchObjects request and silently returns only the first.")
probe("client-fetch-cbor-label", S,
      'encoding != "json" {',
      'encoding != "json" && encoding != "cbor" {',
      "TestFetchObjectsRefusesCBORLabelAfterJSONRequest",
      "^TestFetchObjectsRefusesCBORLabelAfterJSONRequest$",
      "Admits a cbor-labelled response object after the client requested json.")
probe("guard-aliased-json-unmarshal", "internal/merkleinventory/strict_json.go",
      'var errStrictJSON = errors.New("invalid strict JSON shape")',
      'var errStrictJSON = errors.New("invalid strict JSON shape")\n\nvar decodeJSONAlias = json.Unmarshal',
      "TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings",
      "^TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings$",
      "Control plant: adds an aliased json.Unmarshal function value outside strictJSON; the AST guard must fail.")
probe("emptychild2", S,
      'if prefix != "" && node.Count.Uint64() == 0 {',
      'if prefix != "" && node.Count.Uint64() == 0 && len(prefix) != 2 {',
      "TestDispatchPrefixAxisCoversEveryLengthAndAlphabet", "^TestDispatchPrefixAxisCoversEveryLengthAndAlphabet$/^lower_hex_len_02_absent$",
      "Returns an invented empty child only for a valid absent two-nibble prefix.")
probe("crossns", S,
      "if namespace, found := index.locations[id]; found && namespace != context.Namespace {",
      'if namespace, found := index.locations[id]; found && namespace != context.Namespace && context.Namespace != "manifest" {',
      "TestObjectsGetRefusesEveryForeignNamespacePair", "^TestObjectsGetRefusesEveryForeignNamespacePair$/^requested_manifest_object_record$",
      "Lets a stored record requested through manifest fall through to not_found instead of the cross-namespace refusal.")
probe("fetchns", S,
      "membership.Namespace != expectedNamespace || membership.ID.String() != expectedID",
      'membership.Namespace != expectedNamespace && expectedNamespace != "manifest" || membership.ID.String() != expectedID',
      "TestFetchObjectsRefusesEveryHostileForeignNamespacePair", "^TestFetchObjectsRefusesEveryHostileForeignNamespacePair$/^requested_manifest_object_record$",
      "Admits a hostile peer's valid record as a manifest object on the client fetch path.")
probe("descriptor-namespace", I,
      '{blobDescriptorV1, "1.0.0"}:                     "manifest",',
      '{blobDescriptorV1, "1.0.0"}:                     "record",',
      "TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob", "^TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob$",
      "Relabels Blob Descriptor schema membership from manifest to record.")
probe("excluded-local-marker", I,
      'if membership.Excluded {\n\t\treturn ErrExcluded\n\t}\n\tindex.mu.Lock()',
      'if membership.Excluded {\n\t\tif membership.Schema == "urn:ax:schema:terminal-instance-binding" {\n\t\t\tindex.mu.Lock()\n\t\t\tindex.putLocked("record", "sha256:0000000000000000000000000000000000000000000000000000000000000000", data, true)\n\t\t\tindex.mu.Unlock()\n\t\t\treturn nil\n\t\t}\n\t\treturn ErrExcluded\n\t}\n\tindex.mu.Lock()',
      "TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker", "^TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker$",
      "Admits one machine-local terminal marker into the record root.")
probe("mixed-ns1-root-member-narrowing", S,
      "for id := range index.objects[namespace] {\n\t\tids = append(ids, id)\n\t}",
      'for id := range index.objects[namespace] {\n\t\tif namespace == "tombstone_ack" && id == "sha256:4444444444444444444444444444444444444444444444444444444444444444" {\n\t\t\tcontinue\n\t\t}\n\t\tids = append(ids, id)\n\t}',
      "TestMixedNS1RootsFromNormativeSyntheticIDs", "^TestMixedNS1RootsFromNormativeSyntheticIDs$",
      "Omits the exact synthetic Tombstone Acknowledgement identity from its production namespace root.")
probe("objects-get-schema-membership", S,
      'if verifyErr != nil || membership.Excluded || membership.Namespace != context.Namespace || membership.ID.String() != id {\n\t\t\tindex.mu.RUnlock()',
      'if verifyErr != nil || membership.Excluded || (membership.Namespace != context.Namespace && context.Namespace != "record") || membership.ID.String() != id {\n\t\t\tindex.mu.RUnlock()',
      "TestObjectsGetRevalidatesStoredSchemaMembership", "^TestObjectsGetRevalidatesStoredSchemaMembership$",
      "Admits one manifest object from the stored-object check when queried as record.")
probe("exchange-objects-get-schema-membership", S,
      'if verifyErr != nil || membership.Excluded || membership.Namespace != context.Namespace || membership.ID.String() != id {\n\t\t\tindex.mu.RUnlock()',
      'if verifyErr != nil || membership.Excluded || (membership.Namespace != context.Namespace && context.Namespace != "record") || membership.ID.String() != id {\n\t\t\tindex.mu.RUnlock()',
      "TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects",
      "^TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects$",
      "Admits a real manifest Descriptor from a misrouted record store during the real-byte exchange.")
probe("objects-get-line-limit", S,
      "if uint64(len(lineOut)) > context.MaxLineBytes {", "if uint64(len(lineOut)) > context.MaxLineBytes+1 {",
      "TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit", "^TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit$",
      "Emits exactly one byte over the negotiated response line limit.")
probe("objects-get-request-count", S,
      "len(objectIDs) > 4096", "len(objectIDs) > 4097",
      "TestObjectsGetRefusesRequestIDCountOutsideBound", "^TestObjectsGetRefusesRequestIDCountOutsideBound$",
      "Admits a request with 4097 object IDs.")
probe("same-id-quarantine", I,
      "if bytes.Equal(previous.data, data) {\n\t\t\treturn nil\n\t\t}",
      'if bytes.Equal(previous.data, data) || bytes.Equal(previous.data, []byte("tampered prior payload")) {\n\t\t\treturn nil\n\t\t}',
      "TestAddJSONQuarantinesSameIDDifferentStoredBytes", "^TestAddJSONQuarantinesSameIDDifferentStoredBytes$",
      "Treats one different-byte stored variant as an identical object instead of quarantining it.")
probe("tombstone-timestamp-winner", I,
      '''func (index *Index) insertLocked(namespace, id string, data []byte, isJSON bool) error {
	if err := index.preflightLocked(namespace, id, data); err != nil {''',
      '''func (index *Index) insertLocked(namespace, id string, data []byte, isJSON bool) error {
	if namespace == "tombstone" {
		incoming, decodeErr := strictObject(data)
		incomingSubjectID, incomingSubjectOK := requiredJSONString(incoming, "subject_id")
		incomingCreatedAt, incomingTimeOK := requiredJSONString(incoming, "created_at")
		if decodeErr == nil && incomingSubjectOK && incomingTimeOK {
			for priorID, prior := range index.objects[namespace] {
				existing, existingErr := strictObject(prior.data)
				existingSubjectID, existingSubjectOK := requiredJSONString(existing, "subject_id")
				existingCreatedAt, existingTimeOK := requiredJSONString(existing, "created_at")
				if existingErr == nil && existingSubjectOK && existingTimeOK && existingSubjectID == incomingSubjectID {
					if incomingCreatedAt > existingCreatedAt {
						delete(index.objects[namespace], priorID)
						delete(index.locations, priorID)
					} else {
						return nil
					}
				}
			}
		}
	}
	if err := index.preflightLocked(namespace, id, data); err != nil {''',
      "TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements",
      "^TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements$",
      "Selects the later-created Tombstone as a winner and discards the same-subject earlier immutable Tombstone.")
probe("rpc2-version-admission", I,
      "if version != RPC2Version || !slices.Equal(namespaces, rpc2Namespaces) {",
      "if (version != RPC2Version && version != \"3.0.0\") || (version == RPC2Version && !slices.Equal(namespaces, rpc2Namespaces)) {",
      "TestNewRefusesHigherRPCNamespaceSets", "^TestNewRefusesHigherRPCNamespaceSets$",
      "Admits RPC 3.0.0 through this RPC 2-only inventory constructor.")
probe("blob-chunk-content-mismatch", I,
      "if err != nil || scalar.SHA256Digest(raw[offset.Uint64():end]).String() != parsedChunkID.String() {",
      "if err != nil || scalar.SHA256Digest(raw[offset.Uint64():end]).String() != parsedChunkID.String() && parsedChunkID.String() != scalar.SHA256Digest([]byte(\"different chunk\")).String() {",
      "TestBlobDescriptorByteReferencesAreVerifiedBeforeAdmission", "^TestBlobDescriptorByteReferencesAreVerifiedBeforeAdmission$",
      "Admits the one wrong chunk digest used by the negative fixture while retaining all other digest checks.")
probe("unknown-schema-fail-closed", I,
      'if !supported {\n\t\treturn Membership{}, refusal("integrity_failure", fmt.Errorf("%w: unsupported schema class %s@%s", ErrInvalidObject, schema, version))\n\t}',
      'if !supported {\n\t\treturn Membership{Schema: schema, Excluded: true}, nil\n\t}',
      "TestRPC2SchemaMembershipTableIsTotalAndDisjoint", "^TestRPC2SchemaMembershipTableIsTotalAndDisjoint$",
      "Classifies one unknown schema as excluded instead of refusing it.")
for name, schema, version, test in [
    ("session-record-31", "urn:ax:schema:session-record", "3.1.0", "TestUnsupportedSessionRecord31RemainsFailClosed"),
    ("materialization-plan-10", "urn:ax:schema:materialization-plan", "1.0.0", "TestUnsupportedMaterializationPlan10RemainsFailClosed"),
    ("materialization-plan-20", "urn:ax:schema:materialization-plan", "2.0.0", "TestUnsupportedMaterializationPlan20RemainsFailClosed"),
    ("task-board-bundle-10", "urn:ax:schema:task-board-bundle", "1.0.0", "TestUnsupportedTaskBoardBundle10RemainsFailClosed"),
]:
    if name == "session-record-31":
        probe("unsupported-schema-" + name, I,
              'digest, selfField, err := canonicaljson.VerifyObjectIdentity(data)\n\tif err != nil {\n\t\treturn Membership{}, refusal("integrity_failure", fmt.Errorf("%w: %v", ErrInvalidObject, err))\n\t}',
              'digest, selfField, err := canonicaljson.VerifyObjectIdentity(data)\n\tif err != nil && schema == "' + schema + '" && version == "' + version + '" {\n\t\treturn Membership{Namespace: namespace, ID: scalar.SHA256Digest(data), Schema: schema, Version: version}, nil\n\t}\n\tif err != nil {\n\t\treturn Membership{}, refusal("integrity_failure", fmt.Errorf("%w: %v", ErrInvalidObject, err))\n\t}',
              test, "^" + test + "$",
              "Admits only " + schema + "@" + version + " after canonicaljson refuses its identity contract; the injected byte digest makes the bad admission measurable at inventory roots/counts.")
    else:
        probe("unsupported-schema-" + name, SHAPES,
              'return invalidIdentity("complete immutable-object shape validation is unavailable for %s@%s", schema, version)',
              'if schema == "' + schema + '" && version == "' + version + '" {\n\t\treturn nil\n\t}\n\treturn invalidIdentity("complete immutable-object shape validation is unavailable for %s@%s", schema, version)',
              test, "^" + test + "$",
              "Admits only " + schema + "@" + version + " by narrowing the fail-closed shape validator for that row; the fixture has a valid omit-self digest so validation refusal is isolated to the missing schema shape.")
probe("identity-walk-blob-narrowing", S,
      "if local == nil || peer == nil || local.version != peer.version || !validNamespace(namespace) {",
      'if local == nil || peer == nil || local.version != peer.version || !validNamespace(namespace) || namespace == "blob" {',
      "TestMixedNSExchangeIdentityLevelSyntheticIDs", "^TestMixedNSExchangeIdentityLevelSyntheticIDs$",
      "Narrows identity-only recursive inventory walking to omit the fixture's valid raw-blob identity.")
probe("objects-get-blob-transfer-boundary", S,
      'if namespace == "blob" {\n\t\treturn nil, ErrInvalidRequest\n\t}',
      'if namespace == "blob" && local.version != RPC2Version {\n\t\treturn nil, ErrInvalidRequest\n\t}',
      "TestMixedNSExchangeIdentityLevelSyntheticIDs", "^TestMixedNSExchangeIdentityLevelSyntheticIDs$",
      "Admits a raw-blob identity through the object-only MissingObjectIDs walk in the supported RPC 2 version.")
probe("objects-get-duplicate-id", S,
      "if _, err := scalar.ParseDigest(raw); err != nil || (i > 0 && objectIDs[i-1] >= raw) {",
      "if _, err := scalar.ParseDigest(raw); err != nil || (i > 0 && objectIDs[i-1] > raw) {",
      "TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests",
      "^TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests$",
      "Admits a repeated object ID by weakening strict ascending order to non-decreasing order.")
probe("objects-get-cbor-only", S,
      "if !slices.Contains(encodings, \"json\") {",
      "if !slices.Contains(encodings, \"json\") && !slices.Contains(encodings, \"cbor\") {",
      "TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests",
      "^TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests$",
      "Admits a CBOR-only request even though this RPC 2 implementation can return JSON only.")
probe("objects-get-unbounded-batch", S,
      "ObjectIDs: []string{id}, Encodings: []string{\"json\"}",
      "ObjectIDs: ids, Encodings: []string{\"json\"}",
      "TestFetchObjectsUsesBoundedSingletonBatches", "^TestFetchObjectsUsesBoundedSingletonBatches$",
      "Sends the complete missing-ID list in each request instead of one bounded object per request.")
probe("tombstone-target-scope-kind", C,
      "if kind != scope {", 'if kind != scope && scope != "session" {',
      "TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes",
      "^TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes$/^target_kind_mismatch$",
      "Admits a target-kind mismatch for the session scope.", package="./internal/canonicaljson")
probe("tombstone-path-question-wildcard", C,
      'strings.ContainsAny(value, "*?[]")', 'strings.ContainsAny(value, "*[]")',
      "TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes",
      "^TestTombstoneIdentityEntryRefusesNarrowedTargetAndAuthorityShapes$/^workspace_path_question_wildcard$",
      "Admits a workspace path containing the question-mark wildcard.", package="./internal/canonicaljson")
probe("tombstone-logical-root-maximum", C,
      'requireBoundedString(target, "logical_root", 1, 64)',
      'requireBoundedString(target, "logical_root", 1, 65)',
      "TestEveryCoreRecordDeclaredBoundAcceptsAtItsLimitAndRefusesPastIt",
      "^TestEveryCoreRecordDeclaredBoundAcceptsAtItsLimitAndRefusesPastIt$",
      "Admits a 65-character logical_root beyond the pinned 64-character maximum.",
      package="./internal/canonicaljson")
probe("tombstone-ack-conflict-checkpoint", C,
      'if conflictPresent != (disposition == "retained_conflict") {',
      'if conflictPresent && disposition != "retained_conflict" {',
      "TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings",
      "^TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings$/^conflict_requires_checkpoint$",
      "Admits a retained_conflict acknowledgement with a null checkpoint.",
      package="./internal/canonicaljson")
probe("tombstone-ack-created-by", C,
      'if createdByHostID != acknowledgingHostID {',
      'if createdByHostID != acknowledgingHostID && disposition != "applied" {',
      "TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings",
      "^TestTombstoneAckIdentityEntryEnforcesDispositionAndIssuerCouplings$/^created_by_must_be_acknowledging_host$",
      "Admits an applied acknowledgement created by a different host.",
      package="./internal/canonicaljson")
probe("tombstone-ack-requires-reference", I,
      '''if !exists {
		return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement references a Tombstone not admitted to this inventory", ErrInvalidObject))
	}''',
	'''if !exists && subjectID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" {
		return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement references a Tombstone not admitted to this inventory", ErrInvalidObject))
	}
	if !exists {
		return nil
	}''',
      "TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone",
      "^TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone$",
      "Admits an unreferenced acknowledgement for one pinned fixture subject.")
probe("tombstone-ack-subject-link", I,
      "if referencedSubjectID != subjectID {",
      'if referencedSubjectID != subjectID && subjectID != "0198f4c8-5b20-7c33-8c4d-1234567890ab" {',
      "TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone",
      "^TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone$",
      "Admits a mismatched acknowledgement subject for one pinned negative fixture.")

# Rev3 carry-over probes: these arms were reachable or directly seeded by the
# preceding review. Each named behavior test runs alone against its overlay.
probe("fetch-dup-ids", S,
      "ids[index-1] >= id", "ids[index-1] > id",
      "TestFetchObjectsRefusesDuplicateIDs", "^TestFetchObjectsRefusesDuplicateIDs$",
      "Admits one duplicate digest in the client request ordering gate.")
probe("b64-roundtrip", S,
      "if err != nil || base64URL(dataBytes) != encodedData {",
      "if err != nil {",
      "TestFetchObjectsRefusesNewlineBearingBase64urlData",
      "^TestFetchObjectsRefusesNewlineBearingBase64urlData$",
      "Admits CRLF-bearing base64url data because Go's Strict decoder ignores embedded CR/LF.")
probe("walk-quarantine", S,
      "if quarantined {", 'if quarantined && namespace != "record" {',
      "TestMissingObjectIDsRefusesLocalQuarantinedIdentity",
      "^TestMissingObjectIDsRefusesLocalQuarantinedIdentity$",
      "Treats a quarantined local record identity as an ordinary missing object.")
probe("walk-crossns", S,
      "if exists && namespaceOfID != namespace {",
      'if exists && namespaceOfID != namespace && namespace != "record" {',
      "TestMissingObjectIDsRefusesCrossNamespaceIdentity",
      "^TestMissingObjectIDsRefusesCrossNamespaceIdentity$",
      "Admits a locally stored manifest identity through the record recursive-walk arm.")

# Durable-union gates are attacked through the public sync/rebuild behavior.
probe("durable-unknown-schema-admit", "internal/merkleinventory/durable.go",
      '''membership, err := ClassifyJSON(data)
	if err != nil {
		return err
	}''',
      '''membership, err := ClassifyJSON(data)
	if err != nil {
		if bytes.Contains(data, []byte("urn:ax:schema:unknown")) {
			membership = Membership{Namespace: "record", ID: scalar.SHA256Digest(data)}
		} else {
			return err
		}
	}''',
      "TestDurableAddRejectsBeforePersistingInvalidObjects",
      "^TestDurableAddRejectsBeforePersistingInvalidObjects$",
      "Admits the exact unsupported schema fixture as a durable record after ClassifyJSON refuses its class.")
probe("durable-event-replay", "internal/merkleinventory/durable.go",
      "if priorNamespace == membership.Namespace && hasPrior && bytes.Equal(prior.data, data) {",
      'if priorNamespace == membership.Namespace && hasPrior && bytes.Equal(prior.data, data) && membership.Namespace != "event" {',
      "TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace",
      "^TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace$/^event$",
      "Treats an identical replay of one event as a byte conflict after a crash/restart.")
probe("durable-repeat-record-after-crash", "internal/merkleinventory/durable.go",
      "if priorNamespace == membership.Namespace && hasPrior && bytes.Equal(prior.data, data) {\n\t\t\treturn nil\n\t\t}",
      'if priorNamespace == membership.Namespace && hasPrior && bytes.Equal(prior.data, data) && membership.Namespace != "record" {\n\t\t\treturn nil\n\t\t}',
      "TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace",
      "^TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace$/^record$",
      "Lets one exact Session Record retry after a simulated install crash fall through to the quarantine/removal path.")
probe("durable-same-id-different-bytes", "internal/merkleinventory/durable.go",
      "if priorNamespace == membership.Namespace && hasPrior && bytes.Equal(prior.data, data) {",
      "if priorNamespace == membership.Namespace && hasPrior {",
      "TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes",
      "^TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes$",
      "Admits a same-digest record with different bytes as an identical object instead of quarantining and aborting.")
probe("sync-common-record-audit", "internal/merkleinventory/durable.go",
      "if _, exists := localSet[id]; exists {",
      'if _, exists := localSet[id]; exists && namespace != "record" {',
      "TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes",
      "^TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes$",
      "Skips byte auditing for common record IDs, allowing equal Merkle roots to hide a same-digest byte conflict.")
probe("sync-common-audit-partial-overlap-peer", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''if len(common) == len(peerIDs) {
			if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
				return err
			}
		}''',
      "TestDurableSyncConflictWithPartialOverlapPeer",
      "^TestDurableSyncConflictWithPartialOverlapPeer$",
      "Reviewer reproduction narrowing: audit common bytes only when every peer identity is already common, so a same-ID byte conflict plus a peer-only object silently diverges.")
probe("sync-common-audit-only-without-missing", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''if missing, missingErr := MissingObjectIDs(store.index, peer.index, namespace); missingErr == nil && len(missing) == 0 {
			if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
				return err
			}
		}''',
      "TestDurableSyncConflictWithPartialOverlapPeer",
      "^TestDurableSyncConflictWithPartialOverlapPeer$",
      "Audits common bytes only when the peer has no missing identities in that namespace, admitting the partial-overlap conflict class.")
probe("sync-common-audit-first-namespace-only", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''if namespace == durableJSONNamespaces[0] {
			if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
				return err
			}
		}''',
      "TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord",
      "^TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord$",
      "Audits common bytes only in the first durable namespace, skipping a Session Record conflict in a later namespace.")
probe("sync-common-audit-first-id-only", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''common = common[:min(1, len(common))]
		if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}''',
      "TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord",
      "^TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord$",
      "Audits only the first common ID in a namespace; the generated middle and last targets must still refuse the conflicting peer.")
probe("sync-common-audit-last-id-only", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''common = common[max(0, len(common)-1):]
		if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}''',
      "TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord",
      "^TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord$",
      "Audits only the last common ID; the generated first and middle targets must still refuse the conflicting peer.")
probe("sync-common-audit-even-ids-only", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''selected := make([]string, 0, (len(common)+1)/2)
		for index, id := range common {
			if index%2 == 0 {
				selected = append(selected, id)
			}
		}
		common = selected
		if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}''',
      "TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord",
      "^TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord$",
      "Audits only even-indexed common IDs; the generated middle target must still refuse the conflicting peer.")
probe("skip-census-unjustified-planted-skip", "internal/merkleinventory/sync_property_test.go",
      "func TestDurableSyncGeneratedPerturbationProduct(t *testing.T) {",
      "func TestDurableSyncGeneratedPerturbationProduct(t *testing.T) {\n\tt.Skip(\"planted without a platform or environment capability reason\")",
      "TestNoUnjustifiedTestSkips", "^TestNoUnjustifiedTestSkips$",
      "Adds an unjustified Skip call to the package source; the AST census must refuse it.")
probe("skip-census-token-preserving-behavior", "internal/merkleinventory/sync_rework_test.go",
      '''if !ok || !slices.Contains([]string{"Skip", "Skipf", "SkipNow"}, selector.Sel.Name) {''',
      '''if !ok || !slices.Contains([]string{"Skip", "Skipf", "SkipNow"}, "Not"+selector.Sel.Name) {''',
      "TestSkipCensusRejectsAnUnjustifiedPlantedSkip",
      "^TestSkipCensusRejectsAnUnjustifiedPlantedSkip$",
      "Keeps each searched-for Skip method token but changes the AST check so a real planted skip is ignored.")
probe("sync-common-audit-first-sync-only", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''if len(store.index.locations) == 1 {
			if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
				return err
			}
		}''',
      "TestDurableSyncConflictWithPartialOverlapPeer",
      "^TestDurableSyncConflictWithPartialOverlapPeer$",
      "Audits only while the local union still has its initial one identity; after a successful first sync, a new common-ID conflict silently passes.")
probe("sync-common-audit-ast-control", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''if len(common) == len(peerIDs) {
			if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
				return err
			}
		}''',
      "TestDurableSyncCommonIDAuditIsUnconditionalInAST",
      "^TestDurableSync(CommonIDAuditIsUnconditionalInAST|ConflictWithPartialOverlapPeer)$",
      "Applied source-structure control: the AST audit guard and partial-overlap behavioral suite must both reject a gated common-ID audit.")
probe("sync-common-audit-early-namespace-return", "internal/merkleinventory/durable.go",
      SYNC_COMMON_AUDIT,
      '''if namespace != durableJSONNamespaces[0] {
			return nil
		}
		if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}''',
      "TestDurableSyncCommonIDAuditIsUnconditionalInAST",
      "^TestDurableSync(CommonIDAuditIsUnconditionalInAST|ConflictWithPartialOverlapPeer)$",
      "Returns early for every namespace after the first before the common-ID audit; both AST structure and the partial-overlap behavior must fail.")
probe("sync-missing-record-fetch", "internal/merkleinventory/durable.go",
      '''if err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil {
			return err
		}''',
      '''if namespace != "record" {
			if err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil {
				return err
			}
		}''',
      "TestDurableSyncGapFillsOnLaterPass",
      "^TestDurableSyncGapFillsOnLaterPass$",
      "Leaves missing record objects undiscovered and unfetched during a sync.")
probe("sync-arrival-order-truncates-fetch", "internal/merkleinventory/durable.go",
      '''for _, object := range objects {
			data, err := base64.RawURLEncoding.Strict().DecodeString(object.Data)''',
      '''for index, object := range objects {
			if index > 0 { break }
			data, err := base64.RawURLEncoding.Strict().DecodeString(object.Data)''',
      "TestDurableSyncArrivalOrderConverges",
      "^TestDurableSyncArrivalOrderConverges$",
      "Stops a fetched batch after its first arrival and omits the remaining immutable identities.")
probe("sync-peer-namespace-mismatch", "internal/merkleinventory/durable.go",
      '''if err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil {
			return err
		}''',
      '''if err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil && namespace != "event" {
			return err
		}''',
      "TestDurableSyncRefusesPeerNamespaceMismatch",
      "^TestDurableSyncRefusesPeerNamespaceMismatch$",
      "Suppresses one event-namespace schema mismatch returned while fetching a missing identity, allowing sync to continue.")
probe("sync-partial-peer-regresses-local-root", "internal/merkleinventory/durable.go",
      '''\t\tif err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil {
\t\t\treturn err
\t\t}
\t}
\treturn store.validateTombstoneAckClosureLocked()
}''',
      '''\t\tif err := store.fetchAndAdd(peer, namespace, missing, nextRequestID); err != nil {
\t\t\treturn err
\t\t}
\t}
\tpeerRecords := peer.index.objectIDs("record")
\tlocalRecords := store.index.objectIDs("record")
\tif len(localRecords) > len(peerRecords) && len(localRecords) > 0 {
\t\tid := localRecords[0]
\t\tstore.index.mu.Lock()
\t\tdelete(store.index.objects["record"], id)
\t\tdelete(store.index.locations, id)
\t\tstore.index.mu.Unlock()
\t\tif err := store.removeActive("record", id); err != nil {
\t\t\treturn err
\t\t}
\t}
\treturn store.validateTombstoneAckClosureLocked()
}''',
      "TestDurableSyncPartialPeerNeverRegressesLocalRoots",
      "^TestDurableSyncPartialPeerNeverRegressesLocalRoots$",
      "Drops the smallest local record identity whenever the peer has fewer records, admitting a root regression on a proper subset exchange.")
probe("sync-repeat-record-quarantine-write", "internal/merkleinventory/durable.go",
      '''if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}''',
      '''if err := store.fetchAndAdd(peer, namespace, common, nextRequestID); err != nil {
			return err
		}
		if namespace == "record" && len(common) > 0 {
			data, err := peer.Object(namespace, common[0])
			if err != nil {
				return err
			}
			if err := store.persistQuarantineCandidate(namespace, common[0], data); err != nil {
				return err
			}
		}''',
      "TestDurableSyncGeneratedPerturbationProduct/N1/skew0/order000/duplicatefalse/gap0/peer00",
      "^TestDurableSyncGeneratedPerturbationProduct$/N1/skew0/order000/duplicatefalse/gap0/peer00$",
      "Writes the exact common Session Record into durable quarantine on every sync, admitting a non-no-op mutation for a converged record set.")
probe("sync-skip-tombstone-union", "internal/merkleinventory/durable.go",
      "for _, namespace := range durableJSONNamespaces {",
      'for _, namespace := range durableJSONNamespaces {\n\t\tif namespace == "tombstone" { continue }',
      "TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords",
      "^TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords$",
      "Drops Tombstone records from the durable union instead of retaining them as immutable data.")
probe("union-ack-arrival-order", I,
      "return index.addJSON(data, false)", "return index.addJSON(data, true)",
      "TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords",
      "^TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords$",
      "Requires a Tombstone before accepting an acknowledgement, refusing the valid acknowledgement-first union order.")
probe("quarantine-crash-recovery", "internal/merkleinventory/durable.go",
      "if candidates := quarantined[membership.ID.String()]; len(candidates) > 0 {",
      "if candidates := quarantined[membership.ID.String()]; len(candidates) > 1 {",
      "TestDurableConflictCrashRestoresQuarantineAndAbortsSync",
      "^TestDurableConflictCrashRestoresQuarantineAndAbortsSync$",
      "Fails to quarantine an active object after a crash leaves exactly one persisted conflicting candidate.")
probe("union-lease-tuple-omission", "internal/sessquery/lease.go",
      "heads = append(heads, sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID})",
      'if candidate.LeaseID != "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff" { heads = append(heads, sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID}) }',
      "TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation",
      "^TestLeaseHeadsForSessionReturnsEveryTupleAcrossTimestampPerturbation$",
      "Drops one competing validated lease tuple instead of deriving the complete post-union head set.",
      package="./internal/sessquery")
probe("union-lease-created-at-winner", "internal/sessquery/lease.go",
      "heads = append(heads, sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID})",
      '''head := sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID}
		candidateMembers, _ := environ.DecodeStrictObject(candidate.Raw)
		candidateTime, _ := leaseStringMember(candidateMembers, "created_at")
		for _, other := range candidates {
			otherMembers, _ := environ.DecodeStrictObject(other.Raw)
			otherTime, _ := leaseStringMember(otherMembers, "created_at")
			if candidateTime > otherTime { head.Epoch++ }
		}
		heads = append(heads, head)''',
      "TestDurableSyncClockSkewDoesNotSelectLeaseWinner",
      "^TestDurableSyncClockSkewDoesNotSelectLeaseWinner$",
      "Uses the later diagnostic created_at value to promote that lease above its competing tuple.")
probe("union-lease-conflicting-same-id", "internal/sessquery/lease.go",
      "if prior.Digest != candidate.Digest || string(prior.Raw) != string(candidate.Raw) {",
      'if (prior.Digest != candidate.Digest || string(prior.Raw) != string(candidate.Raw)) && candidate.LeaseID != "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee" {',
      "TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID",
      "^TestLeaseHeadsForSessionRefusesConflictingBytesForOneLeaseID$",
      "Silently deduplicates conflicting bytes for one pinned lease UUID instead of refusing integrity_failure.",
      package="./internal/sessquery")
probe("union-lease-generated-cardinality-omission", "internal/sessquery/lease.go",
      "heads = append(heads, sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID})",
      'if candidate.LeaseID != "00000000-0000-4000-8000-000000000011" { heads = append(heads, sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID}) }',
      "TestLeaseHeadsForSessionCoversGeneratedCardinalityRange",
      "^TestLeaseHeadsForSessionCoversGeneratedCardinalityRange$/^size_17$",
      "Drops one generated lease tuple inside the tested cardinality range instead of returning every validated tuple.",
      package="./internal/sessquery")
probe("sync-unclosed-tombstone-ack", "internal/merkleinventory/durable.go",
      "if !exists {\n\t\t\treturn refusal(\"integrity_failure\", fmt.Errorf(\"%w: Tombstone Acknowledgement references no unioned Tombstone\", ErrInvalidObject))\n\t\t}",
      "if !exists && subjectID != \"0198f4c8-3e70-7a11-8a2b-1234567890ab\" {\n\t\t\treturn refusal(\"integrity_failure\", fmt.Errorf(\"%w: Tombstone Acknowledgement references no unioned Tombstone\", ErrInvalidObject))\n\t\t}\n\t\tif !exists {\n\t\t\tcontinue\n\t\t}",
      "TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers",
      "^TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers$",
      "Admits one missing Tombstone link for the pinned union subject at the post-union closure gate.")
probe("projection-missing-union-record", "internal/merkleinventory/durable.go",
      "if !recordPresent || !bytes.Equal(storedRecord.data, recordBytes) {",
      'if (!recordPresent || !bytes.Equal(storedRecord.data, recordBytes)) && sessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" {',
      "TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion",
      "^TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion$/^record$",
      "Projects one session whose authoritative record bytes are absent from the immutable union.")
probe("projection-missing-union-event", "internal/merkleinventory/durable.go",
      "if !present || !bytes.Equal(stored.data, eventBytes) {",
      "if (!present || !bytes.Equal(stored.data, eventBytes)) && len(events) != 1 {",
      "TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion",
      "^TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion$/^event$",
      "Projects a one-event authoritative chain even though that event's exact bytes are absent from the union.")
probe("durable-ack-subject-link", "internal/merkleinventory/durable.go",
      "if !ok || referencedSubject != subjectID {",
      "if !ok || (referencedSubject != subjectID && subjectID != \"0198f4c8-3e70-7a11-8a2b-1234567890ac\") {",
      "TestDurableSyncRefusesAcknowledgementWithMismatchedTombstoneSubject",
      "^TestDurableSyncRefusesAcknowledgementWithMismatchedTombstoneSubject$",
      "Admits one validated Acknowledgement whose subject_id differs from its unioned Tombstone.")
probe("projection-skip-unclosed-ack", "internal/merkleinventory/durable.go",
      "if err := index.validateTombstoneAckClosure(); err != nil {\n\t\treturn sessstate.Projection{}, err\n\t}",
      "if err := index.validateTombstoneAckClosure(); err != nil && !strings.Contains(err.Error(), \"references no unioned Tombstone\") {\n\t\treturn sessstate.Projection{}, err\n\t}",
      "TestProjectionRebuildRefusesUnclosedAcknowledgement",
      "^TestProjectionRebuildRefusesUnclosedAcknowledgement$",
      "Admits one projection rebuild when an Acknowledgement has no unioned Tombstone.")
probe("durable-load-skips-forged-object", "internal/merkleinventory/durable.go",
      "if err != nil || membership.Excluded || membership.Namespace != object.namespace || membership.ID.Hex()+\".json\" != object.name {\n\t\t\treturn fmt.Errorf(\"%w: active object %s/%s failed schema, digest, or path validation\", ErrDurableCorrupt, object.namespace, object.name)\n\t\t}",
      "if err != nil || membership.Excluded || membership.Namespace != object.namespace || membership.ID.Hex()+\".json\" != object.name {\n\t\t\tif bytes.Equal(object.data, []byte(\"{\\\"forged\\\":true}\")) { continue }\n\t\t\treturn fmt.Errorf(\"%w: active object %s/%s failed schema, digest, or path validation\", ErrDurableCorrupt, object.namespace, object.name)\n\t\t}",
      "TestDurableStoreRejectsCorruptActiveBytesOnOpen",
      "^TestDurableStoreRejectsCorruptActiveBytesOnOpen$",
      "Silently skips one corrupt active payload instead of refusing to reopen the durable union.")
probe("projection-drops-union-head", "internal/merkleinventory/durable.go",
      "return (&sessstate.Projector{Repo: repo, Union: leaseHeads}).Project(sessionID)",
      "if len(leaseHeads) > 1 { leaseHeads = leaseHeads[:len(leaseHeads)-1] }; return (&sessstate.Projector{Repo: repo, Union: leaseHeads}).Project(sessionID)",
      "TestDurableSyncClockSkewDoesNotSelectLeaseWinner",
      "^TestDurableSyncClockSkewDoesNotSelectLeaseWinner$",
      "Omits the greatest derived lease head at the production projection rebuild boundary.")


def run_probe(output, item):
    source = ROOT / item["path"]
    original = source.read_bytes()
    content = original.decode()
    count = content.count(item["old"])
    if count != 1:
        raise RuntimeError(f'{item["name"]}: exact source plant matched {count} times, expected once')
    folder = output / item["name"]
    folder.mkdir(parents=True, exist_ok=True)
    replacement = folder / source.name
    replacement.write_text(content.replace(item["old"], item["new"], 1))
    overlay = folder / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}) + "\n")
    command = ["go", "test", "-overlay", str(overlay), item.get("package", "./internal/merkleinventory"),
               "-count=1", "-json", "-timeout=120s", "-run", item["regex"]]
    try:
        environment = None
        if item["name"] == "guard-aliased-json-unmarshal":
            environment = os.environ.copy()
            environment["MERKLE_INVENTORY_GUARD_SCAN_DIR"] = str(folder)
        if item["name"] == "sync-common-audit-ast-control":
            if environment is None:
                environment = os.environ.copy()
            environment["MERKLE_SYNC_AUDIT_SOURCE"] = str(replacement)
        if item["name"] == "sync-common-audit-early-namespace-return":
            if environment is None:
                environment = os.environ.copy()
            environment["MERKLE_SYNC_AUDIT_SOURCE"] = str(replacement)
        if item["name"] == "skip-census-unjustified-planted-skip":
            if environment is None:
                environment = os.environ.copy()
            environment["MERKLE_TEST_SKIP_SCAN_DIR"] = str(folder)
        process = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                 env=environment, timeout=180, check=False)
        raw = process.stdout
        exit_code = process.returncode
    except subprocess.TimeoutExpired as exc:
        raw = exc.stdout or b""
        exit_code = 124
    (folder / "test.log").write_bytes(raw)
    failed = []
    passed = []
    for line in raw.decode(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("Action") == "fail" and event.get("Test"):
            failed.append(event["Test"])
        if event.get("Action") == "pass" and event.get("Test"):
            passed.append(event["Test"])
    exact_failure = any(value == item["test"] or value.startswith(item["test"] + "/") for value in failed)
    is_control = item["name"] == "control-neutral-comment"
    if is_control:
        state = "survived-control" if exit_code == 0 and not failed and passed else "invalid-control"
        valid = state == "survived-control"
    else:
        state = "killed" if exit_code != 0 and exact_failure else "survived" if exit_code == 0 and passed else "invalid-instrument-or-compile-failure"
        valid = state == "killed"
    if source.read_bytes() != original:
        raise RuntimeError(f'{item["name"]}: tracked source changed despite overlay')
    return {
        "mutant": item["name"], "source": item["path"], "narrowing": item["narrowing"],
        "named_test": item["test"], "command": command, "exit_code": exit_code,
        "result": state, "failing_tests": failed, "valid": valid,
        "raw_log": str((folder / "test.log").relative_to(output)),
    }


def run_token_preserving_identity_probe(output):
    """Keep the source census token and call site while breaking its behavior."""
    source = ROOT / I
    original = source.read_bytes()
    content = original.decode()
    old = '''digest, selfField, err := canonicaljson.VerifyObjectIdentity(data)
	if err != nil {
		return Membership{}, refusal("integrity_failure", fmt.Errorf("%w: %v", ErrInvalidObject, err))
	}'''
    new = '''digest, selfField, err := canonicaljson.VerifyObjectIdentity(data)
	if err != nil {
		return Membership{Namespace: namespace, ID: digest, Schema: schema, Version: version}, nil
	}'''
    if content.count(old) != 1:
        raise RuntimeError(f"{TOKEN_PRESERVING_IDENTITY_MUTANT}: exact source plant did not match once")
    if "canonicaljson.VerifyObjectIdentity" not in new:
        raise RuntimeError("token-preserving mutant unexpectedly removed the searched-for token")
    folder = output / TOKEN_PRESERVING_IDENTITY_MUTANT
    folder.mkdir(parents=True, exist_ok=True)
    (folder / source.name).write_text(content.replace(old, new, 1))
    behavior_log = folder / "behavioral-suite.log"
    source_gate_log = folder / "source-gate.log"
    behavior_command = ["go", "test", "./internal/merkleinventory", "-count=1", "-json",
                        "-timeout=30s", "-run", "^TestClassifyRefusesIdentityDigestMismatch$"]
    source_gate_command = ["go", "test", "./internal/provhost", "-count=1", "-json",
                           "-timeout=30s", "-run", "^TestNoProductionPathAttestsProviderIdentityBinding$"]

    def invoke(command, path):
        try:
            process = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE,
                                     stderr=subprocess.STDOUT, timeout=120, check=False)
            raw, code = process.stdout, process.returncode
        except subprocess.TimeoutExpired as exc:
            raw, code = exc.stdout or b"", 124
        path.write_bytes(raw)
        failed, passed = [], []
        for line in raw.decode(errors="replace").splitlines():
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                continue
            if event.get("Action") == "fail" and event.get("Test"):
                failed.append(event["Test"])
            if event.get("Action") == "pass" and event.get("Test"):
                passed.append(event["Test"])
        return code, failed, passed

    try:
        source.write_text(content.replace(old, new, 1))
        behavior_code, behavior_failed, _ = invoke(behavior_command, behavior_log)
        gate_code, gate_failed, gate_passed = invoke(source_gate_command, source_gate_log)
    finally:
        source.write_bytes(original)

    behavior_killed = behavior_code != 0 and "TestClassifyRefusesIdentityDigestMismatch" in behavior_failed
    gate_still_sees_owned_token = (
        gate_code == 0
        and "TestNoProductionPathAttestsProviderIdentityBinding" in gate_passed
        and not gate_failed
    )
    valid = behavior_killed and gate_still_sees_owned_token and source.read_bytes() == original
    return {
        "mutant": TOKEN_PRESERVING_IDENTITY_MUTANT,
        "source": I,
        "narrowing": "Leaves canonicaljson.VerifyObjectIdentity in the owned inventory call site but converts its validation error into an accepted membership.",
        "named_test": "TestClassifyRefusesIdentityDigestMismatch",
        "command": behavior_command,
        "exit_code": behavior_code,
        "result": "killed-by-behavior; source census passed" if valid else "invalid-probe",
        "failing_tests": behavior_failed,
        "source_gate_command": source_gate_command,
        "source_gate_exit_code": gate_code,
        "source_gate_result": "passed with searched token preserved" if gate_still_sees_owned_token else "failed",
        "source_gate_log": str(source_gate_log.relative_to(output)),
        "raw_log": str(behavior_log.relative_to(output)),
        "valid": valid,
    }


def run_inventory_allowlist_probe(output):
    """Narrow the identity census by admitting a non-index inventory file."""
    source = ROOT / "internal/provhost/identity_test.go"
    original = source.read_bytes()
    content = original.decode()
    old = 'return filepath.Clean(path) == filepath.Join(root, "internal", "merkleinventory", "index.go")'
    new = 'return strings.HasPrefix(filepath.Clean(path), filepath.Join(root, "internal", "merkleinventory")+string(filepath.Separator))'
    if content.count(old) != 1:
        raise RuntimeError(f"{INVENTORY_ALLOWLIST_MUTANT}: exact source plant did not match once")
    folder = output / INVENTORY_ALLOWLIST_MUTANT
    folder.mkdir(parents=True, exist_ok=True)
    replacement = folder / source.name
    replacement.write_text(content.replace(old, new, 1))
    overlay = folder / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}) + "\n")
    command = ["go", "test", "-overlay", str(overlay), "./internal/provhost", "-count=1",
               "-json", "-timeout=20s", "-run", "^TestAttestationAllowlistAnchorsOwningPath$"]
    try:
        process = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE,
                                 stderr=subprocess.STDOUT, timeout=120, check=False)
        raw, exit_code = process.stdout, process.returncode
    except subprocess.TimeoutExpired as exc:
        raw, exit_code = exc.stdout or b"", 124
    (folder / "test.log").write_bytes(raw)
    failed, passed = [], []
    for line in raw.decode(errors="replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("Action") == "fail" and event.get("Test"):
            failed.append(event["Test"])
        if event.get("Action") == "pass" and event.get("Test"):
            passed.append(event["Test"])
    killed = exit_code != 0 and "TestAttestationAllowlistAnchorsOwningPath" in failed
    return {
        "mutant": INVENTORY_ALLOWLIST_MUTANT,
        "source": "internal/provhost/identity_test.go",
        "narrowing": "Broadens the exact inventory index allowlist to every Go file under internal/merkleinventory, admitting the synthetic other.go caller.",
        "named_test": "TestAttestationAllowlistAnchorsOwningPath",
        "command": command,
        "exit_code": exit_code,
        "result": "killed" if killed else "survived" if exit_code == 0 else "invalid-instrument-or-compile-failure",
        "failing_tests": failed,
        "valid": killed,
        "raw_log": str((folder / "test.log").relative_to(output)),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True)
    parser.add_argument("--only", help="Comma-separated probe names for a bounded rerun")
    args = parser.parse_args()
    selected_names = args.only.split(",") if args.only else [item["name"] for item in PROBES] + [TOKEN_PRESERVING_IDENTITY_MUTANT, INVENTORY_ALLOWLIST_MUTANT]
    known = {item["name"] for item in PROBES} | {TOKEN_PRESERVING_IDENTITY_MUTANT, INVENTORY_ALLOWLIST_MUTANT}
    if not set(selected_names) <= known:
        parser.error("--only contains an unknown probe")
    output = Path(args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    results = []
    if TOKEN_PRESERVING_IDENTITY_MUTANT in selected_names:
        result = run_token_preserving_identity_probe(output)
        results.append(result)
        print(f'{result["mutant"]}: {result["result"]}, behavior exit={result["exit_code"]}, source gate exit={result["source_gate_exit_code"]}', flush=True)
        (output / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    if INVENTORY_ALLOWLIST_MUTANT in selected_names:
        result = run_inventory_allowlist_probe(output)
        results.append(result)
        print(f'{result["mutant"]}: {result["result"]}, exit={result["exit_code"]}, failing={",".join(result["failing_tests"])}', flush=True)
        (output / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    for item in PROBES:
        if item["name"] not in selected_names:
            continue
        result = run_probe(output, item)
        results.append(result)
        print(f'{result["mutant"]}: {result["result"]}, exit={result["exit_code"]}, failing={",".join(result["failing_tests"])}', flush=True)
        (output / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    table = ["| Mutant | Narrowing | Named test | Exit | Result | Raw log |",
             "| --- | --- | --- | ---: | --- | --- |"]
    for result in results:
        detail = result["result"]
        if result.get("source_gate_log"):
            detail += f'; source gate log `{result["source_gate_log"]}`'
        table.append(f'| {result["mutant"]} | {result["narrowing"]} | {result["named_test"]} | {result["exit_code"]} | {detail} | `{result["raw_log"]}` |')
    (output / "table.md").write_text("\n".join(table) + "\n")
    return 0 if all(item["valid"] for item in results) and results else 1


if __name__ == "__main__":
    sys.exit(main())
