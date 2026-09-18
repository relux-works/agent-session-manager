package clonebundle

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Canonical Session 1.0.0 and its
// Actor records: the logical session envelope over ordered unique
// Canonical Event IDs.

const (
	canonicalSessionSchema  = "urn:ax:schema:canonical-session"
	canonicalSessionVersion = "1.0.0"
	canonicalSessionSelf    = "canonical_session_id"
)

var actorMembers = map[string]bool{
	"actor_id": true, "kind": true, "parent_actor_id": true, "name": true,
	"source_native_id": true, "model": true, "extensions": true,
}

var actorRequired = []string{
	"actor_id", "kind", "parent_actor_id", "name",
	"source_native_id", "model", "extensions",
}

// Actor is one validated canonical actor.
type Actor struct {
	ActorID        scalar.UUIDv7
	Kind           string
	ParentActorID  *scalar.UUIDv7
	Name           *string
	SourceNativeID *string
	Model          *string
}

func validActorKind(kind string) bool {
	return kind == "main" || kind == "subagent" || kind == "external"
}

func decodeActor(raw json.RawMessage, index int) (Actor, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return Actor{}, invalid("actor[%d] %s (%s)", index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, actorMembers); unknown {
		return Actor{}, invalid("actor[%d] carries unknown member %q", index, name)
	}
	if name, missing := missingMember(members, actorRequired); missing {
		return Actor{}, invalid("actor[%d] misses a required member %q", index, name)
	}
	actorID, ok := checkUUIDv7(members["actor_id"])
	if !ok {
		return Actor{}, invalid("actor[%d] actor_id is not a UUIDv7", index)
	}
	kind, ok := rawString(members["kind"])
	if !ok || !validActorKind(kind) {
		return Actor{}, invalid("actor[%d] kind is outside main|subagent|external", index)
	}
	var parent *scalar.UUIDv7
	if !isNull(members["parent_actor_id"]) {
		value, ok := checkUUIDv7(members["parent_actor_id"])
		if !ok {
			return Actor{}, invalid("actor[%d] parent_actor_id is not a UUIDv7", index)
		}
		parent = &value
	}
	if kind == "main" && parent != nil {
		return Actor{}, invalid("actor[%d] main actor carries a parent", index)
	}
	if kind != "main" && parent == nil {
		return Actor{}, invalid("actor[%d] non-main actor requires a parent", index)
	}
	name, err := checkNullableBoundedString(members, "name", 1, 512, index, "actor")
	if err != nil {
		return Actor{}, err
	}
	nativeID, err := checkNullableBoundedString(members, "source_native_id", 1, 512, index, "actor")
	if err != nil {
		return Actor{}, err
	}
	if nativeID != nil {
		if err := SanitizeNativeKey(*nativeID); err != nil {
			return Actor{}, err
		}
	}
	model, err := checkNullableBoundedString(members, "model", 1, 512, index, "actor")
	if err != nil {
		return Actor{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return Actor{}, invalid("actor[%d] extensions %s", index, extensionsFault)
	}
	return Actor{
		ActorID:        actorID,
		Kind:           kind,
		ParentActorID:  parent,
		Name:           name,
		SourceNativeID: nativeID,
		Model:          model,
	}, nil
}

func checkNullableBoundedString(members map[string]json.RawMessage, name string, minimum, maximum, index int, owner string) (*string, error) {
	raw, present := members[name]
	if !present {
		return nil, invalid("%s[%d] misses a required member %q", owner, index, name)
	}
	if isNull(raw) {
		return nil, nil
	}
	value, ok := checkStringBounds(raw, minimum, maximum)
	if !ok {
		return nil, invalid("%s[%d] %s is not a string[%d..%d]", owner, index, name, minimum, maximum)
	}
	return &value, nil
}

// ActorInput is one caller-supplied actor candidate for Build.
type ActorInput struct {
	ActorID        string
	Kind           string
	ParentActorID  *string
	Name           *string
	SourceNativeID *string
	Model          *string
}

func buildActor(input ActorInput, index int) (Actor, error) {
	actorID, err := scalar.ParseUUIDv7(input.ActorID)
	if err != nil {
		return Actor{}, invalid("actor[%d] actor_id is not a UUIDv7: %v", index, err)
	}
	if !validActorKind(input.Kind) {
		return Actor{}, invalid("actor[%d] kind is outside main|subagent|external", index)
	}
	var parent *scalar.UUIDv7
	if input.ParentActorID != nil {
		value, err := scalar.ParseUUIDv7(*input.ParentActorID)
		if err != nil {
			return Actor{}, invalid("actor[%d] parent_actor_id is not a UUIDv7: %v", index, err)
		}
		parent = &value
	}
	if input.Kind == "main" && parent != nil {
		return Actor{}, invalid("actor[%d] main actor carries a parent", index)
	}
	if input.Kind != "main" && parent == nil {
		return Actor{}, invalid("actor[%d] non-main actor requires a parent", index)
	}
	if input.Name != nil {
		if !validText(*input.Name) {
			return Actor{}, invalid("actor[%d] name is not valid UTF-8", index)
		}
		if stringLength(*input.Name) < 1 || stringLength(*input.Name) > 512 {
			return Actor{}, invalid("actor[%d] name is not a string[1..512]", index)
		}
	}
	if input.SourceNativeID != nil {
		if stringLength(*input.SourceNativeID) < 1 || stringLength(*input.SourceNativeID) > 512 {
			return Actor{}, invalid("actor[%d] source_native_id is not a string[1..512]", index)
		}
		if err := SanitizeNativeKey(*input.SourceNativeID); err != nil {
			return Actor{}, err
		}
	}
	if input.Model != nil {
		if !validText(*input.Model) {
			return Actor{}, invalid("actor[%d] model is not valid UTF-8", index)
		}
		if stringLength(*input.Model) < 1 || stringLength(*input.Model) > 512 {
			return Actor{}, invalid("actor[%d] model is not a string[1..512]", index)
		}
	}
	return Actor{
		ActorID:        actorID,
		Kind:           input.Kind,
		ParentActorID:  parent,
		Name:           input.Name,
		SourceNativeID: input.SourceNativeID,
		Model:          input.Model,
	}, nil
}

func actorObject(actor Actor) map[string]any {
	var parent any
	if actor.ParentActorID != nil {
		parent = actor.ParentActorID.String()
	}
	var name any
	if actor.Name != nil {
		name = *actor.Name
	}
	var nativeID any
	if actor.SourceNativeID != nil {
		nativeID = *actor.SourceNativeID
	}
	var model any
	if actor.Model != nil {
		model = *actor.Model
	}
	return map[string]any{
		"actor_id":         actor.ActorID.String(),
		"kind":             actor.Kind,
		"parent_actor_id":  parent,
		"name":             name,
		"source_native_id": nativeID,
		"model":            model,
		"extensions":       map[string]any{},
	}
}

var canonicalSessionMembers = map[string]bool{
	"schema": true, "schema_version": true, "canonical_session_id": true,
	"logical_session_id": true, "source_environment": true,
	"source_native_session_id": true, "title": true, "workspace": true,
	"actors": true, "event_ids": true, "head_event_ids": true,
	"created_at": true, "updated_at": true, "extensions": true,
}

var canonicalSessionRequired = []string{
	"schema", "schema_version", "canonical_session_id",
	"logical_session_id", "source_environment",
	"source_native_session_id", "title", "workspace",
	"actors", "event_ids", "head_event_ids",
	"created_at", "updated_at", "extensions",
}

// CanonicalSession is one validated Canonical Session.
type CanonicalSession struct {
	SessionID             scalar.Digest
	LogicalSessionID      scalar.UUIDv7
	SourceEnvironment     sessadapter.Tuple
	SourceNativeSessionID string
	Title                 *string
	Workspace             WorkspaceBinding
	Actors                []Actor
	EventIDs              []scalar.Digest
	HeadEventIDs          []scalar.Digest
	CreatedAt             *scalar.Timestamp
	UpdatedAt             *scalar.Timestamp
}

// CanonicalSessionInput is the caller-supplied canonical session
// candidate for Build.
type CanonicalSessionInput struct {
	LogicalSessionID      string
	SourceEnvironment     []byte
	SourceNativeSessionID string
	Title                 *string
	Workspace             []byte
	Actors                []ActorInput
	EventIDs              []string
	HeadEventIDs          []string
	CreatedAt             *string
	UpdatedAt             *string
	Extensions            map[string]any
}

// BuildCanonicalSession constructs one closed Canonical Session
// 1.0.0 as canonical bytes. Identical inputs produce byte-identical
// outputs.
func BuildCanonicalSession(input CanonicalSessionInput) ([]byte, error) {
	object, err := buildCanonicalSession(input)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	sessionID := scalar.SHA256Digest(omitted)
	object[canonicalSessionSelf] = sessionID.String()
	return canonicalizeObject(object)
}

func buildCanonicalSession(input CanonicalSessionInput) (map[string]any, error) {
	logical, err := scalar.ParseUUIDv7(input.LogicalSessionID)
	if err != nil {
		return nil, invalid("canonical session logical_session_id is not a UUIDv7: %v", err)
	}
	tuple, err := sessadapter.DecodeTuple(json.RawMessage(input.SourceEnvironment))
	if err != nil {
		return nil, invalid("canonical session source_environment is not an Environment Tuple: %v", err)
	}
	if stringLength(input.SourceNativeSessionID) < 1 || stringLength(input.SourceNativeSessionID) > 512 {
		return nil, invalid("canonical session source_native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(input.SourceNativeSessionID); err != nil {
		return nil, err
	}
	var title any
	if input.Title != nil {
		if !validText(*input.Title) {
			return nil, invalid("canonical session title is not valid UTF-8")
		}
		if stringLength(*input.Title) < 1 || stringLength(*input.Title) > 4096 {
			return nil, invalid("canonical session title is not a string[1..4096]")
		}
		title = *input.Title
	}
	if _, err := DecodeWorkspaceBinding(json.RawMessage(input.Workspace)); err != nil {
		return nil, err
	}
	if len(input.Actors) < 1 || len(input.Actors) > 1024 {
		return nil, invalid("canonical session carries %d actors, want [1..1024]", len(input.Actors))
	}
	actors := make([]Actor, 0, len(input.Actors))
	seenActors := map[string]bool{}
	mainCount := 0
	for index, candidate := range input.Actors {
		actor, err := buildActor(candidate, index)
		if err != nil {
			return nil, err
		}
		if seenActors[actor.ActorID.String()] {
			return nil, invalid("canonical session actors are not unique by actor ID")
		}
		seenActors[actor.ActorID.String()] = true
		if actor.Kind == "main" {
			mainCount++
		}
		actors = append(actors, actor)
	}
	if mainCount != 1 {
		return nil, invalid("canonical session carries %d main actors, want exactly one", mainCount)
	}
	eventIDs, err := parseOrderedUniqueDigests(input.EventIDs, 1, 1000000, "event_ids")
	if err != nil {
		return nil, err
	}
	headIDs, err := parseSortedUniqueDigestStrings(input.HeadEventIDs, 1, 1024, "head_event_ids")
	if err != nil {
		return nil, err
	}
	eventSet := map[string]bool{}
	for _, id := range eventIDs {
		eventSet[id.String()] = true
	}
	for _, id := range headIDs {
		if !eventSet[id.String()] {
			return nil, invalid("canonical session head_event_ids carry an ID outside event_ids")
		}
	}
	createdAt, err := parseNullableTimestamp(input.CreatedAt, "created_at")
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseNullableTimestamp(input.UpdatedAt, "updated_at")
	if err != nil {
		return nil, err
	}
	if _, err := encodeExtensions(input.Extensions); err != nil {
		return nil, err
	}
	var workspaceValue any
	if err := json.Unmarshal(bytesTrimSpace(json.RawMessage(input.Workspace)), &workspaceValue); err != nil {
		return nil, invalid("canonical session workspace is not JSON: %v", err)
	}
	actorObjects := make([]any, 0, len(actors))
	for _, actor := range actors {
		actorObjects = append(actorObjects, actorObject(actor))
	}
	return map[string]any{
		"schema":                   canonicalSessionSchema,
		"schema_version":           canonicalSessionVersion,
		"logical_session_id":       logical.String(),
		"source_environment":       tupleObject(tuple),
		"source_native_session_id": input.SourceNativeSessionID,
		"title":                    title,
		"workspace":                workspaceValue,
		"actors":                   actorObjects,
		"event_ids":                digestStrings(eventIDs),
		"head_event_ids":           digestStrings(headIDs),
		"created_at":               createdAt,
		"updated_at":               updatedAt,
		"extensions":               extensionValue(input.Extensions),
	}, nil
}

// DecodeCanonicalSession validates one closed Canonical Session
// 1.0.0: exact members, schema/version, self-digest agreement,
// exactly one main actor, ordered unique event IDs, and head IDs
// drawn from the event set.
func DecodeCanonicalSession(data []byte) (CanonicalSession, error) {
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return CanonicalSession{}, invalid("canonical session %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, canonicalSessionMembers); unknown {
		return CanonicalSession{}, invalid("canonical session carries unknown member %q", name)
	}
	if name, missing := missingMember(members, canonicalSessionRequired); missing {
		return CanonicalSession{}, invalid("canonical session misses a required member %q", name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != canonicalSessionSchema {
		return CanonicalSession{}, invalid("canonical session schema is not the canonical session")
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != canonicalSessionVersion {
		return CanonicalSession{}, invalid("canonical session version is not 1.0.0")
	}
	sessionID, ok := checkDigest(members[canonicalSessionSelf])
	if !ok {
		return CanonicalSession{}, invalid("canonical session canonical_session_id is not a digest")
	}
	logical, ok := checkUUIDv7(members["logical_session_id"])
	if !ok {
		return CanonicalSession{}, invalid("canonical session logical_session_id is not a UUIDv7")
	}
	tuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil {
		return CanonicalSession{}, invalid("canonical session source_environment is not an Environment Tuple: %v", err)
	}
	nativeID, ok := checkStringBounds(members["source_native_session_id"], 1, 512)
	if !ok {
		return CanonicalSession{}, invalid("canonical session source_native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(nativeID); err != nil {
		return CanonicalSession{}, err
	}
	var title *string
	if !isNull(members["title"]) {
		value, ok := checkStringBounds(members["title"], 1, 4096)
		if !ok {
			return CanonicalSession{}, invalid("canonical session title is not a string[1..4096]")
		}
		title = &value
	}
	workspace, err := DecodeWorkspaceBinding(members["workspace"])
	if err != nil {
		return CanonicalSession{}, err
	}
	actors, err := decodeActors(members["actors"])
	if err != nil {
		return CanonicalSession{}, err
	}
	eventIDs, err := decodeOrderedUniqueDigests(members["event_ids"], 1, 1000000, "event_ids")
	if err != nil {
		return CanonicalSession{}, err
	}
	headIDs, ok := checkSortedUniqueDigests(members["head_event_ids"], 1, 1024)
	if !ok {
		return CanonicalSession{}, invalid("canonical session head_event_ids are not sorted unique digest[1..1024]")
	}
	eventSet := map[string]bool{}
	for _, id := range eventIDs {
		eventSet[id.String()] = true
	}
	for _, id := range headIDs {
		if !eventSet[id.String()] {
			return CanonicalSession{}, invalid("canonical session head_event_ids carry an ID outside event_ids")
		}
	}
	createdAt, err := decodeNullableTimestamp(members, "created_at", "canonical session")
	if err != nil {
		return CanonicalSession{}, err
	}
	updatedAt, err := decodeNullableTimestamp(members, "updated_at", "canonical session")
	if err != nil {
		return CanonicalSession{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return CanonicalSession{}, invalid("canonical session extensions %s", extensionsFault)
	}
	if err := verifySelfDigest(members, canonicalSessionSelf, sessionID); err != nil {
		return CanonicalSession{}, err
	}
	return CanonicalSession{
		SessionID:             sessionID,
		LogicalSessionID:      logical,
		SourceEnvironment:     tuple,
		SourceNativeSessionID: nativeID,
		Title:                 title,
		Workspace:             workspace,
		Actors:                actors,
		EventIDs:              eventIDs,
		HeadEventIDs:          headIDs,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	}, nil
}

func decodeActors(raw json.RawMessage) ([]Actor, error) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, invalid("canonical session actors are not an array")
	}
	if len(elements) < 1 || len(elements) > 1024 {
		return nil, invalid("canonical session carries %d actors, want [1..1024]", len(elements))
	}
	actors := make([]Actor, 0, len(elements))
	seen := map[string]bool{}
	mainCount := 0
	for index, element := range elements {
		actor, err := decodeActor(element, index)
		if err != nil {
			return nil, err
		}
		if seen[actor.ActorID.String()] {
			return nil, invalid("canonical session actors are not unique by actor ID")
		}
		seen[actor.ActorID.String()] = true
		if actor.Kind == "main" {
			mainCount++
		}
		actors = append(actors, actor)
	}
	if mainCount != 1 {
		return nil, invalid("canonical session carries %d main actors, want exactly one", mainCount)
	}
	return actors, nil
}

// decodeOrderedUniqueDigests validates ordered unique digests: order
// is the caller's (event order), uniqueness is exact.
func decodeOrderedUniqueDigests(raw json.RawMessage, minimum, maximum int, name string) ([]scalar.Digest, error) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, invalid("canonical session %s are not an array", name)
	}
	if len(elements) < minimum || len(elements) > maximum {
		return nil, invalid("canonical session %s carry %d IDs, want [%d..%d]", name, len(elements), minimum, maximum)
	}
	ids := make([]scalar.Digest, 0, len(elements))
	seen := map[string]bool{}
	for index, element := range elements {
		id, ok := checkDigest(element)
		if !ok {
			return nil, invalid("canonical session %s[%d] is not a digest", name, index)
		}
		if seen[id.String()] {
			return nil, invalid("canonical session %s are not unique", name)
		}
		seen[id.String()] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func parseOrderedUniqueDigests(values []string, minimum, maximum int, name string) ([]scalar.Digest, error) {
	if len(values) < minimum || len(values) > maximum {
		return nil, invalid("canonical session %s carry %d IDs, want [%d..%d]", name, len(values), minimum, maximum)
	}
	ids := make([]scalar.Digest, 0, len(values))
	seen := map[string]bool{}
	for index, value := range values {
		id, err := scalar.ParseDigest(value)
		if err != nil {
			return nil, invalid("canonical session %s[%d] is not a digest: %v", name, index, err)
		}
		if seen[id.String()] {
			return nil, invalid("canonical session %s are not unique", name)
		}
		seen[id.String()] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func parseSortedUniqueDigestStrings(values []string, minimum, maximum int, name string) ([]scalar.Digest, error) {
	if len(values) < minimum || len(values) > maximum {
		return nil, invalid("canonical session %s carry %d IDs, want [%d..%d]", name, len(values), minimum, maximum)
	}
	ids := make([]scalar.Digest, 0, len(values))
	previous := ""
	for index, value := range values {
		id, err := scalar.ParseDigest(value)
		if err != nil {
			return nil, invalid("canonical session %s[%d] is not a digest: %v", name, index, err)
		}
		if index > 0 && id.String() <= previous {
			return nil, invalid("canonical session %s are not sorted unique", name)
		}
		previous = id.String()
		ids = append(ids, id)
	}
	return ids, nil
}

func parseNullableTimestamp(value *string, name string) (any, error) {
	if value == nil {
		return nil, nil
	}
	stamp, err := scalar.ParseTimestamp(*value)
	if err != nil {
		return nil, invalid("canonical session %s is not a timestamp: %v", name, err)
	}
	return stamp.String(), nil
}

func decodeNullableTimestamp(members map[string]json.RawMessage, name, owner string) (*scalar.Timestamp, error) {
	raw, present := members[name]
	if !present {
		return nil, invalid("%s misses a required member %q", owner, name)
	}
	if isNull(raw) {
		return nil, nil
	}
	stamp, ok := checkTimestamp(raw)
	if !ok {
		return nil, invalid("%s %s is not a timestamp", owner, name)
	}
	return &stamp, nil
}

func digestStrings(ids []scalar.Digest) []any {
	values := make([]any, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}
