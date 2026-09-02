package canonicaljson

import (
	"encoding/json"
	"fmt"
	"net/url"
	pathpkg "path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	blobSchema             = "urn:ax:schema:blob"
	transferManifestSchema = "urn:ax:schema:transfer-manifest"
	closedSchemaVersion    = "1.0.0"
	maxBlobChunks          = 32_768
	maxChunkSize           = 4_194_304
)

var (
	mediaTypePattern       = regexp.MustCompile("^[a-z0-9!#$&^_.+%'*`|~-]+/[a-z0-9!#$&^_.+%'*`|~-]+$")
	semverPattern          = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	reverseDNSPattern      = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$`)
	environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
	environmentIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}$`)
	boardLogicalIDPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	sessionNamePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

type immutableObjectShapeValidator func(map[string]any) error

// immutableObjectShapeValidators is deliberately explicit. The self-identity
// catalog selects the self field, but catalog membership alone does not prove
// that this package implements a schema's complete closed shape. Every
// registered schema/version therefore resolves either to a concrete validator
// or to an explicit refusal. validateImmutableObjectShapeValidators binds this
// table back to the generated registry so a newly registered row cannot fall
// through to extension-only attestation.
var immutableObjectShapeValidators = mustBuildImmutableObjectShapeValidators()

func mustBuildImmutableObjectShapeValidators() map[schemaIdentityKey]immutableObjectShapeValidator {
	validators := make(map[schemaIdentityKey]immutableObjectShapeValidator)
	register := func(schema string, validator immutableObjectShapeValidator, versions ...string) {
		for _, version := range versions {
			key := schemaIdentityKey{schema: schema, version: version}
			if _, duplicate := validators[key]; duplicate {
				panic(fmt.Sprintf("duplicate immutable-object shape validator for %s@%s", schema, version))
			}
			validators[key] = validator
		}
	}

	// Complete validators owned by the scoped Sections 10.1-10.4 gate.
	register("urn:ax:schema:session-record", validateSessionRecordV1, "1.0.0")
	register("urn:ax:schema:session-record", validateSessionRecordV2, "2.0.0")
	register("urn:ax:schema:session-record", validateSessionRecordV3, "3.0.0")
	register(blobSchema, validateBlobDescriptor, closedSchemaVersion)
	register(transferManifestSchema, validateTransferManifest, closedSchemaVersion)

	// Complete Section 5 and Section 10.1 core-record validators.
	register("urn:ax:schema:lease", validateLeaseRecord, "1.0.0")
	register("urn:ax:schema:workspace-group", validateWorkspaceGroupRecord, "1.0.0")
	register("urn:ax:schema:provider-identity", validateProviderIdentityRecord, "1.0.0")
	register("urn:ax:schema:session-event", validateSessionEventV1, "1.0.0")
	register("urn:ax:schema:session-event", validateSessionEventV2, "2.0.0")
	register("urn:ax:schema:session-event", validateSessionEventV3, "3.0.0")
	register("urn:ax:schema:session-event", validateSessionEventV4, "4.0.0")
	register("urn:ax:schema:checkpoint", validateCheckpointRecord, "1.0.0")

	// Remaining Section 10.1 records retain their common-envelope gate before
	// the public identity surface explicitly refuses an unsupported shape.
	register("urn:ax:schema:tombstone", validateUnsupportedRecordEnvelopeShape, "1.0.0")
	register("urn:ax:schema:tombstone-ack", validateUnsupportedRecordEnvelopeShape, "1.0.0")

	// The remaining registered identities are recognized for self-field
	// selection but their schema-specific closed shapes are outside this task.
	// Explicit refusal is safer and more honest than attesting an opaque object.
	register("urn:ax:schema:canonical-event", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:terminal-backend-manifest", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-clone-bundle", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:materialization-plan", rejectUnsupportedImmutableObjectShape, "1.0.0", "2.0.0")
	register("urn:ax:schema:session-continuation-plan", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:task-board-bundle", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:materialization-journal", rejectUnsupportedImmutableObjectShape, "2.0.0")
	register("urn:ax:schema:environment-observation", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:native-session-observation", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-inventory-batch", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:conversation-lineage-link", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-annotation", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-enrichment-profile", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-enrichment-job-request", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-enrichment-job-receipt", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:session-directory-operation-receipt", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:terminal-backend-probe", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:terminal-instance-binding", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:terminal-capability-evidence", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:clone-raw-object-manifest", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:clone-capture-manifest", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:canonical-session", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:fidelity-report", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:projection-plan", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:clone-projected-object-manifest", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:clone-read-back-evidence-manifest", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:clone-validation-report", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:migration-checkpoint", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:clone-lineage-receipt", rejectUnsupportedImmutableObjectShape, "1.0.0")
	register("urn:ax:schema:supported-environment-tuples", rejectUnsupportedImmutableObjectShape, "1.0.0")

	if err := validateImmutableObjectShapeValidators(validators, catalog.Current().SelfIdentities); err != nil {
		panic(fmt.Sprintf("immutable-object shape validator registry is invalid: %v", err))
	}
	return validators
}

func validateImmutableObjectShapeValidators(
	validators map[schemaIdentityKey]immutableObjectShapeValidator,
	definitions []catalog.SelfIdentityContract,
) error {
	expected := make(map[schemaIdentityKey]struct{})
	for _, definition := range definitions {
		for _, version := range definition.ContractVersions {
			key := schemaIdentityKey{schema: string(definition.ContractID), version: version}
			expected[key] = struct{}{}
			validator, ok := validators[key]
			if !ok || validator == nil {
				return fmt.Errorf("missing immutable-object shape validator for %s@%s", key.schema, key.version)
			}
		}
	}
	if len(validators) != len(expected) {
		return fmt.Errorf("immutable-object shape validator table has %d rows, generated registry has %d", len(validators), len(expected))
	}
	for key := range validators {
		if _, ok := expected[key]; !ok {
			return fmt.Errorf("immutable-object shape validator for unregistered schema %s@%s", key.schema, key.version)
		}
	}
	return nil
}

// validateImmutableObjectShape is the total closed-shape gate used by both
// object-identity production entries. Registry membership never creates an
// extension-only fall-through: each schema/version has an explicit validator
// or an explicit unsupported-shape refusal.
func validateImmutableObjectShape(object map[string]any) error {
	schema, err := requiredStringMember(object, "schema")
	if err != nil {
		return err
	}
	version, err := requiredStringMember(object, "schema_version")
	if err != nil {
		return err
	}

	validator, ok := immutableObjectShapeValidators[schemaIdentityKey{schema: schema, version: version}]
	if !ok || validator == nil {
		return invalidIdentity("no immutable-object shape validator for %s@%s", schema, version)
	}
	return validator(object)
}

func validateSessionRecordV1(object map[string]any) error {
	if err := requireExactMembers("Session Record 1.0.0", object,
		"schema", "schema_version", "record_id", "subject_id", "session_id", "name", "kind",
		"created_at", "created_by_host_id", "provider_id", "workspace_group_id", "execution_profile",
		"launch_plan", "task_board", "fork_provenance", "extensions"); err != nil {
		return err
	}
	sessionID, _, _, err := validateSessionRecordCommon(object, "1.0.0")
	if err != nil {
		return err
	}
	return validateSessionForkProvenance(object, sessionID)
}

func validateSessionRecordV2(object map[string]any) error {
	return validateSessionRecordWithDerivation(object, "2.0.0")
}

func validateSessionRecordV3(object map[string]any) error {
	return validateSessionRecordWithDerivation(object, "3.0.0")
}

func validateSessionRecordWithDerivation(object map[string]any, version string) error {
	if err := requireExactMembers("Session Record 2.0.0 and 3.0.0", object,
		"schema", "schema_version", "record_id", "subject_id", "session_id", "name", "kind",
		"created_at", "created_by_host_id", "provider_id", "workspace_group_id", "execution_profile",
		"launch_plan", "task_board", "derivation_provenance", "extensions"); err != nil {
		return err
	}
	sessionID, providerID, _, err := validateSessionRecordCommon(object, version)
	if err != nil {
		return err
	}
	return validateSessionDerivationProvenance(object, version, sessionID, providerID)
}

func validateSessionRecordCommon(object map[string]any, version string) (string, string, string, error) {
	if err := requireExactString(object, "schema", "urn:ax:schema:session-record"); err != nil {
		return "", "", "", err
	}
	if err := requireExactString(object, "schema_version", version); err != nil {
		return "", "", "", err
	}
	if err := requireDigest(object, "record_id"); err != nil {
		return "", "", "", err
	}
	if err := validateCommonRecordEnvelope(object); err != nil {
		return "", "", "", err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return "", "", "", err
	}
	sessionID, err := requireUUIDv7(object, "session_id")
	if err != nil {
		return "", "", "", err
	}
	if subjectID != sessionID {
		return "", "", "", invalidIdentity("Session Record subject_id must equal session_id")
	}
	name, err := requireString(object, "name")
	if err != nil {
		return "", "", "", err
	}
	if !sessionNamePattern.MatchString(name) {
		return "", "", "", invalidIdentity("Session Record name must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}")
	}
	kind, err := requireEnum(object, "kind", "direct", "task_board")
	if err != nil {
		return "", "", "", err
	}
	providerID, err := requireString(object, "provider_id")
	if err != nil {
		return "", "", "", err
	}
	if _, err := scalar.ParseProviderID(providerID); err != nil {
		return "", "", "", invalidIdentity("Session Record provider_id: %v", err)
	}
	if _, err := requireUUIDv7(object, "workspace_group_id"); err != nil {
		return "", "", "", err
	}
	if _, err := requireEnum(object, "execution_profile", "standard", "yolo"); err != nil {
		return "", "", "", err
	}
	launchPlan, err := requireObject(object, "launch_plan")
	if err != nil {
		return "", "", "", err
	}
	if err := validateSessionLaunchPlan(launchPlan); err != nil {
		return "", "", "", err
	}
	if err := validateSessionTaskBoardReference(object, kind); err != nil {
		return "", "", "", err
	}
	return sessionID, providerID, kind, nil
}

func validateSessionLaunchPlan(object map[string]any) error {
	if err := requireExactMembers("Session Record Launch Plan", object,
		"argv", "cwd_workspace_id", "cwd_relative", "env_names", "env_literals", "contains_secrets", "extensions"); err != nil {
		return err
	}
	argv, err := requireArray(object, "argv", 128)
	if err != nil {
		return err
	}
	if len(argv) == 0 {
		return invalidIdentity("Session Record Launch Plan argv must contain at least one element")
	}
	for index, value := range argv {
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) || len(text) == 0 || len(text) > 4096 {
			return invalidIdentity("Session Record Launch Plan argv[%d] must contain 1..4096 UTF-8 bytes", index)
		}
	}
	encodedArgv, err := json.Marshal(argv)
	if err != nil {
		return invalidIdentity("serialize Session Record Launch Plan argv: %v", err)
	}
	if len(encodedArgv) > 65_536 {
		return invalidIdentity("Session Record Launch Plan argv encodes to %d bytes, maximum is 65536", len(encodedArgv))
	}
	if _, err := requireUUIDv7(object, "cwd_workspace_id"); err != nil {
		return err
	}
	if err := requireDotOrRelativePath(object, "cwd_relative"); err != nil {
		return err
	}
	envNames, err := requireArray(object, "env_names", 64)
	if err != nil {
		return err
	}
	seenNames := make(map[string]struct{}, len(envNames))
	previous := ""
	for index, value := range envNames {
		name, ok := value.(string)
		if !ok || !environmentNamePattern.MatchString(name) {
			return invalidIdentity("Session Record Launch Plan env_names[%d] has invalid environment-name grammar", index)
		}
		if index > 0 && name <= previous {
			return invalidIdentity("Session Record Launch Plan env_names must be strictly sorted and unique")
		}
		seenNames[name] = struct{}{}
		previous = name
	}
	envLiterals, err := requireObject(object, "env_literals")
	if err != nil {
		return err
	}
	if len(envLiterals) > 64 {
		return invalidIdentity("Session Record Launch Plan env_literals contains %d members, maximum is 64", len(envLiterals))
	}
	for name, value := range envLiterals {
		if !environmentNamePattern.MatchString(name) {
			return invalidIdentity("Session Record Launch Plan env_literals key %q has invalid environment-name grammar", name)
		}
		if _, duplicate := seenNames[name]; duplicate {
			return invalidIdentity("Session Record Launch Plan environment name %q occurs in both env_names and env_literals", name)
		}
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) || len(text) > 4096 {
			return invalidIdentity("Session Record Launch Plan env_literals[%q] must contain at most 4096 UTF-8 bytes", name)
		}
	}
	containsSecrets, err := requireBool(object, "contains_secrets")
	if err != nil {
		return err
	}
	if containsSecrets {
		return invalidIdentity("Session Record Launch Plan contains_secrets must be false")
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionTaskBoardReference(parent map[string]any, kind string) error {
	value, ok := parent["task_board"]
	if !ok {
		return invalidIdentity("identity input requires member task_board")
	}
	if kind == "direct" {
		if value != nil {
			return invalidIdentity("direct Session Record task_board must be null")
		}
		return nil
	}
	object, err := requireObjectValue(value, "Session Record task_board")
	if err != nil {
		return err
	}
	if err := requireExactMembers("Session Record Task-board Reference", object,
		"bridge_protocol_version", "board", "task_element_id", "launch_mode", "manager_session_ref",
		"board_goal", "native_goal_binding", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "bridge_protocol_version", "1.0.0"); err != nil {
		return err
	}
	board, err := requireObject(object, "board")
	if err != nil {
		return err
	}
	if err := validateSessionBoardIdentity(board); err != nil {
		return err
	}
	if _, err := requirePrintableByteBoundedString(object, "task_element_id", 1, 128); err != nil {
		return err
	}
	launchMode, err := requireEnum(object, "launch_mode", "primary_owner", "tracked_prompt")
	if err != nil {
		return err
	}
	if value, ok := object["manager_session_ref"]; !ok || value != nil {
		return invalidIdentity("Session Record Task-board Reference manager_session_ref must be null at creation")
	}
	binding, err := requireEnum(object, "native_goal_binding", "bound", "prompt", "none")
	if err != nil {
		return err
	}
	goalValue, ok := object["board_goal"]
	if !ok {
		return invalidIdentity("identity input requires member board_goal")
	}
	if launchMode == "primary_owner" {
		if goalValue == nil || binding != "bound" {
			return invalidIdentity("primary_owner Task-board Reference requires a board goal and bound native goal")
		}
	} else if binding == "bound" {
		return invalidIdentity("tracked_prompt Task-board Reference native_goal_binding must be prompt or none")
	}
	if goalValue != nil {
		goal, err := requireObjectValue(goalValue, "Session Record board_goal")
		if err != nil {
			return err
		}
		if err := validateSessionBoardGoal(goal); err != nil {
			return err
		}
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionBoardIdentity(object map[string]any) error {
	if err := requireExactMembers("Session Record Board Identity", object, "kind", "logical_id", "remote_url", "extensions"); err != nil {
		return err
	}
	kind, err := requireEnum(object, "kind", "local", "remote")
	if err != nil {
		return err
	}
	logicalID, err := requireBoundedString(object, "logical_id", 1, 128)
	if err != nil {
		return err
	}
	if !boardLogicalIDPattern.MatchString(logicalID) {
		return invalidIdentity("Session Record Board Identity logical_id has invalid grammar")
	}
	remoteValue, ok := object["remote_url"]
	if !ok {
		return invalidIdentity("identity input requires member remote_url")
	}
	if kind == "local" {
		if remoteValue != nil {
			return invalidIdentity("local Session Record Board Identity remote_url must be null")
		}
	} else {
		remoteURL, ok := remoteValue.(string)
		if !ok || !utf8.ValidString(remoteURL) {
			return invalidIdentity("remote Session Record Board Identity remote_url must be a UTF-8 string")
		}
		parsed, err := url.Parse(remoteURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return invalidIdentity("remote Session Record Board Identity remote_url must be absolute HTTPS without userinfo, query, or fragment")
		}
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionBoardGoal(object map[string]any) error {
	if err := requireExactMembers("Session Record Board Goal", object, "schema", "goal_id", "revision", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "schema", "board-goal-v2"); err != nil {
		return err
	}
	if _, err := requireBoundedString(object, "goal_id", 1, 128); err != nil {
		return err
	}
	revision, err := requireUint(object, "revision", scalar.MaxUint53)
	if err != nil {
		return err
	}
	if revision == 0 {
		return invalidIdentity("Session Record Board Goal revision must be greater than zero")
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionForkProvenance(parent map[string]any, sessionID string) error {
	value, ok := parent["fork_provenance"]
	if !ok {
		return invalidIdentity("identity input requires member fork_provenance")
	}
	if value == nil {
		return nil
	}
	object, err := requireObjectValue(value, "Session Record fork_provenance")
	if err != nil {
		return err
	}
	if err := requireExactMembers("Session Record Fork Provenance", object,
		"source_session_id", "source_checkpoint_id", "source_workspace_group_id", "operation_id", "provider_fork_mode", "extensions"); err != nil {
		return err
	}
	sourceSessionID, err := requireUUIDv7(object, "source_session_id")
	if err != nil {
		return err
	}
	if sourceSessionID == sessionID {
		return invalidIdentity("Session Record Fork Provenance source_session_id must differ from target session_id")
	}
	if err := requireDigest(object, "source_checkpoint_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "source_workspace_group_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "operation_id"); err != nil {
		return err
	}
	if _, err := requireEnum(object, "provider_fork_mode", "native", "supported_import", "task_board_clone"); err != nil {
		return err
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionDerivationProvenance(parent map[string]any, version, sessionID, providerID string) error {
	provenance, err := requireObject(parent, "derivation_provenance")
	if err != nil {
		return err
	}
	kind, err := requireString(provenance, "kind")
	if err != nil {
		return invalidIdentity("Session Record derivation_provenance.kind: %v", err)
	}
	switch kind {
	case "origin":
		return validateSessionOriginProvenance(provenance)
	case "same_provider_fork":
		return validateSessionSameProviderForkProvenance(provenance, sessionID)
	case "cross_environment_clone":
		return validateSessionCrossEnvironmentCloneProvenance(provenance, sessionID)
	case "native_adoption":
		if version != "3.0.0" {
			return invalidIdentity("Session Record derivation_provenance.kind %q is unavailable in %s", kind, version)
		}
		return validateSessionNativeAdoptionProvenance(provenance, providerID)
	default:
		return invalidIdentity("Session Record derivation_provenance.kind %q is not a closed %s union member", kind, version)
	}
}

func validateSessionOriginProvenance(object map[string]any) error {
	if err := requireExactMembers("Session Record origin provenance", object,
		"kind", "creation_operation_id", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "kind", "origin"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "creation_operation_id"); err != nil {
		return err
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionSameProviderForkProvenance(object map[string]any, sessionID string) error {
	if err := requireExactMembers("Session Record same-provider-fork provenance", object,
		"kind", "source_session_id", "source_checkpoint_id", "source_workspace_group_id", "operation_id",
		"provider_fork_mode", "source_profile_event_id", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "kind", "same_provider_fork"); err != nil {
		return err
	}
	sourceSessionID, err := requireUUIDv7(object, "source_session_id")
	if err != nil {
		return err
	}
	if sourceSessionID == sessionID {
		return invalidIdentity("Session Record same-provider-fork provenance source_session_id must differ from target session_id")
	}
	if err := requireDigest(object, "source_checkpoint_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "source_workspace_group_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "operation_id"); err != nil {
		return err
	}
	if _, err := requireEnum(object, "provider_fork_mode", "native", "supported_import", "task_board_clone"); err != nil {
		return err
	}
	if err := requireNullableDigest(object, "source_profile_event_id"); err != nil {
		return err
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionCrossEnvironmentCloneProvenance(object map[string]any, sessionID string) error {
	if err := requireExactMembers("Session Record cross-environment-clone provenance", object,
		"kind", "operation_id", "bundle_id", "source_kind", "source_session_id", "source_session_record_id",
		"source_checkpoint_id", "source_provider_identity_record_id", "source_native_session_id", "source_environment",
		"target_environment", "source_snapshot_digest", "capture_manifest_id", "canonical_session_id",
		"projection_plan_id", "migration_checkpoint_id", "previous_lineage_receipt_id", "source_profile_event_id",
		"extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "kind", "cross_environment_clone"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "operation_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "bundle_id"); err != nil {
		return err
	}
	sourceKind, err := requireEnum(object, "source_kind", "ax_session", "external_native")
	if err != nil {
		return err
	}
	sourceSessionID, sourceSessionPresent, err := requireNullableUUIDv7(object, "source_session_id")
	if err != nil {
		return err
	}
	if sourceSessionPresent && sourceSessionID == sessionID {
		return invalidIdentity("Session Record cross-environment-clone provenance source_session_id must differ from target session_id")
	}
	axSourcePresence := make([]bool, 0, 4)
	axSourcePresence = append(axSourcePresence, sourceSessionPresent)
	for _, name := range []string{"source_session_record_id", "source_checkpoint_id", "source_provider_identity_record_id"} {
		present, err := requireNullableDigestPresence(object, name)
		if err != nil {
			return err
		}
		axSourcePresence = append(axSourcePresence, present)
	}
	for _, present := range axSourcePresence {
		if sourceKind == "ax_session" && !present {
			return invalidIdentity("Session Record cross-environment-clone provenance all four AX-source IDs must be non-null for ax_session")
		}
		if sourceKind == "external_native" && present {
			return invalidIdentity("Session Record cross-environment-clone provenance all four AX-source IDs must be null for external_native")
		}
	}
	if _, err := requirePrintableBoundedString(object, "source_native_session_id", 1, 512); err != nil {
		return err
	}
	for _, name := range []string{"source_environment", "target_environment"} {
		environment, err := requireObject(object, name)
		if err != nil {
			return err
		}
		if err := validateEnvironmentTuple(environment); err != nil {
			return err
		}
	}
	for _, name := range []string{
		"source_snapshot_digest", "capture_manifest_id", "canonical_session_id", "projection_plan_id", "migration_checkpoint_id",
	} {
		if err := requireDigest(object, name); err != nil {
			return err
		}
	}
	for _, name := range []string{"previous_lineage_receipt_id", "source_profile_event_id"} {
		if err := requireNullableDigest(object, name); err != nil {
			return err
		}
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateSessionNativeAdoptionProvenance(object map[string]any, providerID string) error {
	if err := requireExactMembers("Session Record native-adoption provenance", object,
		"kind", "operation_id", "source_host_id", "source_instance_id", "source_observation_id", "source_head_digest",
		"source_environment", "target_provider_id", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "kind", "native_adoption"); err != nil {
		return err
	}
	for _, name := range []string{"operation_id", "source_host_id"} {
		if _, err := requireUUIDv7(object, name); err != nil {
			return err
		}
	}
	for _, name := range []string{"source_instance_id", "source_observation_id", "source_head_digest"} {
		if err := requireDigest(object, name); err != nil {
			return err
		}
	}
	environment, err := requireObject(object, "source_environment")
	if err != nil {
		return err
	}
	if err := validateEnvironmentTuple(environment); err != nil {
		return err
	}
	targetProviderID, err := requireString(object, "target_provider_id")
	if err != nil {
		return err
	}
	if targetProviderID != providerID {
		return invalidIdentity("Session Record native-adoption provenance target_provider_id must equal Session Record provider_id")
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateExtensionsObject(extensions)
}

func validateEnvironmentTuple(object map[string]any) error {
	if err := requireExactMembers("EnvironmentTuple", object,
		"environment_id", "environment_version", "platform", "architecture", "store_schema_fingerprint", "adapter_version"); err != nil {
		return err
	}
	environmentID, err := requireString(object, "environment_id")
	if err != nil {
		return err
	}
	if !environmentIDPattern.MatchString(environmentID) {
		return invalidIdentity("EnvironmentTuple environment_id must match [a-z][a-z0-9.-]{0,63}")
	}
	// The pinned EnvironmentTuple declaration requires these two members but
	// assigns neither a JSON type nor a local format. In particular, the
	// string[1..128] environment_version bound belongs to Environment
	// Observation, and store_schema_fingerprint is not declared as a digest.
	// Exact-member validation above proves presence without inferring either
	// constraint from a different schema or from the member name.
	platform, err := requireString(object, "platform")
	if err != nil {
		return err
	}
	if _, err := scalar.ParsePlatform(platform); err != nil {
		return invalidIdentity("EnvironmentTuple platform: %v", err)
	}
	if _, err := requireEnum(object, "architecture", "amd64", "arm64"); err != nil {
		return err
	}
	adapterVersion, err := requireString(object, "adapter_version")
	if err != nil {
		return err
	}
	if !semverPattern.MatchString(adapterVersion) {
		return invalidIdentity("EnvironmentTuple adapter_version must be canonical semver")
	}
	return nil
}

func validateCommonRecordEnvelope(object map[string]any) error {
	if _, err := requireUUIDv7(object, "subject_id"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "created_by_host_id"); err != nil {
		return err
	}
	createdAt, err := requireString(object, "created_at")
	if err != nil {
		return err
	}
	if _, err := scalar.ParseTimestamp(createdAt); err != nil {
		return invalidIdentity("record envelope created_at: %v", err)
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	return validateMigrationExtensionObject(extensions)
}

func validateUnsupportedRecordEnvelopeShape(object map[string]any) error {
	if err := validateCommonRecordEnvelope(object); err != nil {
		return err
	}
	return rejectUnsupportedImmutableObjectShape(object)
}

func rejectUnsupportedImmutableObjectShape(object map[string]any) error {
	if err := validateMigrationProvenance(object); err != nil {
		return err
	}
	schema, _ := object["schema"].(string)
	version, _ := object["schema_version"].(string)
	return invalidIdentity("complete immutable-object shape validation is unavailable for %s@%s", schema, version)
}

func requirePrintableByteBoundedString(object map[string]any, name string, minimum, maximum int) (string, error) {
	text, err := requireUTF8String(object, name)
	if err != nil {
		return "", err
	}
	if len(text) < minimum || len(text) > maximum {
		return "", invalidIdentity("member %s must contain %d..%d UTF-8 bytes", name, minimum, maximum)
	}
	for _, character := range text {
		if unicode.IsControl(character) {
			return "", invalidIdentity("member %s must contain printable non-control UTF-8", name)
		}
	}
	return text, nil
}

func requirePrintableBoundedString(object map[string]any, name string, minimum, maximum int) (string, error) {
	text, err := requireBoundedString(object, name, minimum, maximum)
	if err != nil {
		return "", err
	}
	for _, character := range text {
		if unicode.IsControl(character) {
			return "", invalidIdentity("member %s must contain printable non-control UTF-8", name)
		}
	}
	return text, nil
}

func validateBlobDescriptor(object map[string]any) error {
	if err := requireExactMembers("Blob Descriptor", object,
		"schema", "schema_version", "descriptor_id", "blob_id", "size", "media_type", "chunks"); err != nil {
		return err
	}
	if err := requireExactString(object, "schema", blobSchema); err != nil {
		return err
	}
	if err := requireExactString(object, "schema_version", closedSchemaVersion); err != nil {
		return err
	}
	if err := requireDigest(object, "descriptor_id"); err != nil {
		return err
	}
	if err := requireDigest(object, "blob_id"); err != nil {
		return err
	}
	totalSize, err := requireUint(object, "size", scalar.MaxUint53)
	if err != nil {
		return err
	}
	mediaType, err := requireString(object, "media_type")
	if err != nil {
		return err
	}
	if utf8.RuneCountInString(mediaType) > 255 || !mediaTypePattern.MatchString(mediaType) {
		return invalidIdentity("Blob Descriptor media_type must be lowercase ASCII type/subtype without parameters")
	}
	chunks, err := requireArray(object, "chunks", maxBlobChunks)
	if err != nil {
		return err
	}
	if totalSize == 0 && len(chunks) != 0 {
		return invalidIdentity("empty Blob Descriptor must contain no chunks")
	}
	if totalSize > 0 && len(chunks) == 0 {
		return invalidIdentity("non-empty Blob Descriptor must contain at least one chunk")
	}

	var covered uint64
	for index, value := range chunks {
		chunk, err := requireObjectValue(value, fmt.Sprintf("BlobChunk[%d]", index))
		if err != nil {
			return err
		}
		if err := requireExactMembers(fmt.Sprintf("BlobChunk[%d]", index), chunk,
			"index", "offset", "size", "chunk_id"); err != nil {
			return err
		}
		chunkIndex, err := requireUint(chunk, "index", 1<<32-1)
		if err != nil {
			return err
		}
		if chunkIndex != uint64(index) {
			return invalidIdentity("BlobChunk[%d] index is %d, want %d", index, chunkIndex, index)
		}
		offset, err := requireUint(chunk, "offset", scalar.MaxUint53)
		if err != nil {
			return err
		}
		if offset != covered {
			return invalidIdentity("BlobChunk[%d] offset is %d, want %d", index, offset, covered)
		}
		size, err := requireUint(chunk, "size", maxChunkSize)
		if err != nil {
			return err
		}
		if size == 0 {
			return invalidIdentity("BlobChunk[%d] size must lie in [1, %d]", index, maxChunkSize)
		}
		if index < len(chunks)-1 && size != maxChunkSize {
			return invalidIdentity("non-final BlobChunk[%d] size is %d, want %d", index, size, maxChunkSize)
		}
		if covered > scalar.MaxUint53-size {
			return invalidIdentity("BlobChunk coverage exceeds uint53")
		}
		covered += size
		if covered > totalSize {
			return invalidIdentity("BlobChunk[%d] exceeds Blob Descriptor size %d", index, totalSize)
		}
		if err := requireDigest(chunk, "chunk_id"); err != nil {
			return err
		}
	}
	if covered != totalSize {
		return invalidIdentity("BlobChunk coverage is %d bytes, want exactly %d", covered, totalSize)
	}
	return nil
}

func validateTransferManifest(object map[string]any) error {
	if err := requireExactMembers("Transfer Manifest", object,
		"schema", "schema_version", "manifest_id", "kind", "subject_id", "base_checkpoint_id",
		"entries", "child_manifest_ids", "workspace_snapshot", "provider_identity_record_id",
		"task_board_bundle_id", "excluded_classes", "created_by_host_id", "created_at", "extensions"); err != nil {
		return err
	}
	if err := requireExactString(object, "schema", transferManifestSchema); err != nil {
		return err
	}
	if err := requireExactString(object, "schema_version", closedSchemaVersion); err != nil {
		return err
	}
	if err := requireDigest(object, "manifest_id"); err != nil {
		return err
	}
	kind, err := requireEnum(object, "kind", "workspace_group", "workspace_tree", "provider", "task_board", "composite")
	if err != nil {
		return err
	}
	subjectID, err := requireUUIDv7(object, "subject_id")
	if err != nil {
		return err
	}
	if err := requireNullableDigest(object, "base_checkpoint_id"); err != nil {
		return err
	}
	entries, err := requireArray(object, "entries", 65_536)
	if err != nil {
		return err
	}
	if err := validateManifestEntries(entries); err != nil {
		return err
	}
	children, err := requireArray(object, "child_manifest_ids", 1_024)
	if err != nil {
		return err
	}
	if err := validateSortedUniqueDigests(children, "child_manifest_ids"); err != nil {
		return err
	}
	if err := requireNullableDigest(object, "provider_identity_record_id"); err != nil {
		return err
	}
	if err := requireNullableDigest(object, "task_board_bundle_id"); err != nil {
		return err
	}
	excluded, err := requireArray(object, "excluded_classes", 128)
	if err != nil {
		return err
	}
	if err := validateSortedUniqueStrings(excluded, "excluded_classes"); err != nil {
		return err
	}
	if _, err := requireUUIDv7(object, "created_by_host_id"); err != nil {
		return err
	}
	createdAt, err := requireString(object, "created_at")
	if err != nil {
		return err
	}
	if _, err := scalar.ParseTimestamp(createdAt); err != nil {
		return invalidIdentity("Transfer Manifest created_at: %v", err)
	}
	extensions, err := requireObject(object, "extensions")
	if err != nil {
		return err
	}
	if err := validateMigrationExtensionObject(extensions); err != nil {
		return err
	}

	snapshotValue := object["workspace_snapshot"]
	providerValue := object["provider_identity_record_id"]
	bundleValue := object["task_board_bundle_id"]
	switch kind {
	case "workspace_group":
		if len(entries) != 0 || len(children) == 0 || snapshotValue == nil || providerValue != nil || bundleValue != nil {
			return invalidIdentity("workspace_group Transfer Manifest tagged-field invariants are not satisfied")
		}
		snapshot, err := requireObjectValue(snapshotValue, "workspace_snapshot")
		if err != nil {
			return err
		}
		return validateWorkspaceSnapshot(snapshot, subjectID)
	case "workspace_tree":
		if snapshotValue != nil || providerValue != nil || bundleValue != nil {
			return invalidIdentity("workspace_tree Transfer Manifest tagged fields must be null")
		}
	case "provider":
		if snapshotValue != nil || providerValue == nil || bundleValue != nil {
			return invalidIdentity("provider Transfer Manifest tagged-field invariants are not satisfied")
		}
	case "task_board":
		if snapshotValue != nil || providerValue != nil || bundleValue == nil {
			return invalidIdentity("task_board Transfer Manifest tagged-field invariants are not satisfied")
		}
	case "composite":
		if len(entries) != 0 || len(children) == 0 || snapshotValue != nil || providerValue != nil || bundleValue != nil {
			return invalidIdentity("composite Transfer Manifest tagged-field invariants are not satisfied")
		}
	}
	return nil
}

// manifestPathOwner is one declared non-directory Transfer Manifest entry.
// Section 10.4 admits children only below a directory, so any later entry that
// resolves through one of these paths is an overlapping destination.
type manifestPathOwner struct {
	path      string
	entryType string
}

func validateManifestEntries(values []any) error {
	previous := ""
	foldedPaths := make(map[string]struct{}, len(values))
	fileModes := make(map[string]uint64)
	// openNonDirectories carries the non-directory entries whose descendant
	// range the scan has not yet passed, deepest last. Overlap detection reuses
	// the strict bytewise order already proven below instead of re-deriving the
	// ancestor set of every path: a descendant of p always starts with p + "/",
	// so it sorts after p and before any later path that is neither a
	// descendant of p nor ordered below p + "/".
	var openNonDirectories []manifestPathOwner
	for index, value := range values {
		entry, err := requireObjectValue(value, fmt.Sprintf("ManifestEntry[%d]", index))
		if err != nil {
			return err
		}
		entryType, err := requireEnum(entry, "type", "directory", "file", "symlink", "hardlink")
		if err != nil {
			return err
		}
		pathValue, err := requireString(entry, "path")
		if err != nil {
			return err
		}
		if _, err := scalar.ParseRelativePath(pathValue); err != nil {
			return invalidIdentity("ManifestEntry[%d] path: %v", index, err)
		}
		if index > 0 && pathValue <= previous {
			return invalidIdentity("Transfer Manifest entries must be strictly bytewise path-sorted")
		}
		foldedPath := simpleFoldKey(pathValue)
		if _, duplicate := foldedPaths[foldedPath]; duplicate {
			return invalidIdentity("Transfer Manifest entry paths must not duplicate or destination-case-collide")
		}
		foldedPaths[foldedPath] = struct{}{}
		previous = pathValue
		for len(openNonDirectories) > 0 {
			owner := openNonDirectories[len(openNonDirectories)-1]
			subtree := owner.path + "/"
			if strings.HasPrefix(pathValue, subtree) {
				return invalidIdentity(
					"ManifestEntry[%d] path %q overlaps earlier %s entry %q; Transfer Manifest entry paths must not overlap",
					index, pathValue, owner.entryType, owner.path)
			}
			if pathValue <= subtree {
				break
			}
			openNonDirectories = openNonDirectories[:len(openNonDirectories)-1]
		}
		if entryType != "directory" {
			openNonDirectories = append(openNonDirectories, manifestPathOwner{path: pathValue, entryType: entryType})
		}
		mode, err := requireUint(entry, "mode", 4095)
		if err != nil {
			return err
		}

		switch entryType {
		case "directory":
			err = requireExactMembers(fmt.Sprintf("directory ManifestEntry[%d]", index), entry, "path", "type", "mode")
		case "file":
			err = requireExactMembers(fmt.Sprintf("file ManifestEntry[%d]", index), entry,
				"path", "type", "mode", "size", "blob_id", "blob_descriptor_id")
			if err == nil {
				_, err = requireUint(entry, "size", scalar.MaxUint53)
			}
			if err == nil {
				err = requireDigest(entry, "blob_id")
			}
			if err == nil {
				err = requireDigest(entry, "blob_descriptor_id")
			}
		case "symlink":
			err = requireExactMembers(fmt.Sprintf("symlink ManifestEntry[%d]", index), entry, "path", "type", "mode", "target")
			if err == nil {
				target, targetErr := requireString(entry, "target")
				if targetErr != nil {
					err = targetErr
				} else if utf8.RuneCountInString(target) > 4096 {
					err = invalidIdentity("symlink ManifestEntry[%d] target exceeds 4096 characters", index)
				} else if symlinkTargetEscapes(pathValue, target) {
					err = invalidIdentity("symlink ManifestEntry[%d] target escapes the materialization root", index)
				}
			}
		case "hardlink":
			err = requireExactMembers(fmt.Sprintf("hardlink ManifestEntry[%d]", index), entry, "path", "type", "mode", "target_path")
			if err == nil {
				target, targetErr := requireString(entry, "target_path")
				if targetErr != nil {
					err = targetErr
				} else if _, targetErr = scalar.ParseRelativePath(target); targetErr != nil {
					err = invalidIdentity("hardlink ManifestEntry[%d] target_path: %v", index, targetErr)
				} else if targetMode, ok := fileModes[target]; !ok || targetMode != mode {
					err = invalidIdentity("hardlink ManifestEntry[%d] target_path must name an earlier file with the same mode", index)
				}
			}
		}
		if err != nil {
			return err
		}
		if entryType == "file" {
			fileModes[pathValue] = mode
		}
	}
	return nil
}

// simpleFoldKey canonicalizes each Unicode simple-fold orbit to its smallest
// rune. Equal keys therefore preserve strings.EqualFold collision semantics
// while allowing a linear set-membership check.
func simpleFoldKey(value string) string {
	var folded strings.Builder
	folded.Grow(len(value))
	for _, character := range value {
		minimum := character
		for candidate := unicode.SimpleFold(character); candidate != character; candidate = unicode.SimpleFold(candidate) {
			if candidate < minimum {
				minimum = candidate
			}
		}
		folded.WriteRune(minimum)
	}
	return folded.String()
}

func symlinkTargetEscapes(entryPath, target string) bool {
	if strings.ContainsRune(target, 0) || strings.HasPrefix(target, "/") || strings.Contains(target, `\`) ||
		len(target) >= 2 && ((target[0] >= 'a' && target[0] <= 'z') || (target[0] >= 'A' && target[0] <= 'Z')) && target[1] == ':' {
		return true
	}
	resolved := pathpkg.Clean(pathpkg.Join(pathpkg.Dir(entryPath), target))
	return resolved == ".." || strings.HasPrefix(resolved, "../")
}

func validateWorkspaceSnapshot(object map[string]any, subjectID string) error {
	if err := requireExactMembers("WorkspaceSnapshot", object, "workspace_group_id", "members"); err != nil {
		return err
	}
	groupID, err := requireUUIDv7(object, "workspace_group_id")
	if err != nil {
		return err
	}
	if groupID != subjectID {
		return invalidIdentity("WorkspaceSnapshot workspace_group_id must equal manifest subject_id")
	}
	members, err := requireArray(object, "members", 256)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return invalidIdentity("WorkspaceSnapshot requires at least one member")
	}
	previousID := ""
	groupPaths := make([]string, 0, len(members))
	for index, value := range members {
		member, err := requireObjectValue(value, fmt.Sprintf("WorkspaceSnapshotMember[%d]", index))
		if err != nil {
			return err
		}
		workspaceID, err := validateWorkspaceSnapshotMember(member, index)
		if err != nil {
			return err
		}
		if index > 0 && workspaceID <= previousID {
			return invalidIdentity("WorkspaceSnapshot members must be strictly workspace-ID sorted")
		}
		previousID = workspaceID
		groupPath, _ := objectString(member, "group_relative_path")
		for _, previousPath := range groupPaths {
			if strings.EqualFold(groupPath, previousPath) {
				return invalidIdentity("WorkspaceSnapshot group_relative_path values must not duplicate or case-collide")
			}
		}
		groupPaths = append(groupPaths, groupPath)
	}
	return nil
}

func validateWorkspaceSnapshotMember(object map[string]any, index int) (string, error) {
	kind, err := requireEnum(object, "kind", "git", "managed_tree")
	if err != nil {
		return "", err
	}
	workspaceID, err := requireUUIDv7(object, "workspace_id")
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("WorkspaceSnapshotMember[%d]", index)
	if kind == "managed_tree" {
		if err := requireExactMembers(name, object,
			"workspace_id", "kind", "group_relative_path", "tree_identity", "tree_manifest_id",
			"repo_relative_cwd", "agent_project_config_paths", "materialization_policy"); err != nil {
			return "", err
		}
		if err := requireRelativePath(object, "group_relative_path"); err != nil {
			return "", err
		}
		if _, err := requireBoundedString(object, "tree_identity", 1, 256); err != nil {
			return "", err
		}
		if err := requireDigest(object, "tree_manifest_id"); err != nil {
			return "", err
		}
		if err := requireDotOrRelativePath(object, "repo_relative_cwd"); err != nil {
			return "", err
		}
		if err := requireSortedUniquePaths(object, "agent_project_config_paths", 256); err != nil {
			return "", err
		}
		if _, err := requireEnum(object, "materialization_policy", "shared_tree", "separate_copy"); err != nil {
			return "", err
		}
		return workspaceID, nil
	}
	if err := requireExactMembers(name, object,
		"workspace_id", "kind", "group_relative_path", "repository_identity", "remotes", "head",
		"upstream_ref", "object_pack", "index", "working_tree_manifest_id", "submodules", "features",
		"repo_relative_cwd", "agent_project_config_paths", "materialization_policy"); err != nil {
		return "", err
	}
	if err := requireRelativePath(object, "group_relative_path"); err != nil {
		return "", err
	}
	repositoryIdentity, err := requireBoundedString(object, "repository_identity", 1, 256)
	if err != nil {
		return "", err
	}
	if err := validateGitRemotes(object); err != nil {
		return "", err
	}
	headObject, err := requireObject(object, "head")
	if err != nil {
		return "", err
	}
	head, err := validateGitHead(headObject)
	if err != nil {
		return "", err
	}
	if err := requireNullableGitRef(object, "upstream_ref"); err != nil {
		return "", err
	}
	packObject, err := requireObject(object, "object_pack")
	if err != nil {
		return "", err
	}
	pack, err := validateGitObjectPack(packObject)
	if err != nil {
		return "", err
	}
	indexObject, err := requireObject(object, "index")
	if err != nil {
		return "", err
	}
	gitIndex, err := validateGitIndex(indexObject)
	if err != nil {
		return "", err
	}
	if err := requireDigest(object, "working_tree_manifest_id"); err != nil {
		return "", err
	}
	featuresObject, err := requireObject(object, "features")
	if err != nil {
		return "", err
	}
	features, err := validateGitFeatures(featuresObject)
	if err != nil {
		return "", err
	}
	if pack.objectFormat != features.objectFormat {
		return "", invalidIdentity("GitObjectPack object_format must match GitFeatures object_format")
	}
	if err := validateGitHeadObjectFormat(head, features.objectFormat); err != nil {
		return "", err
	}
	if err := validateGitIndexObjectFormat(gitIndex, features.objectFormat); err != nil {
		return "", err
	}
	submodules, err := requireArray(object, "submodules", 256)
	if err != nil {
		return "", err
	}
	totalSubmodules := 0
	if err := validateGitSubmodules(submodules, features.objectFormat, gitIndex, 1, &totalSubmodules, map[string]struct{}{repositoryIdentity: {}}); err != nil {
		return "", err
	}
	if err := requireDotOrRelativePath(object, "repo_relative_cwd"); err != nil {
		return "", err
	}
	if err := requireSortedUniquePaths(object, "agent_project_config_paths", 256); err != nil {
		return "", err
	}
	if _, err := requireEnum(object, "materialization_policy", "shared_checkout", "separate_worktree"); err != nil {
		return "", err
	}
	return workspaceID, nil
}

func validateGitRemotes(parent map[string]any) error {
	values, err := requireArray(parent, "remotes", 16)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return invalidIdentity("Git remotes requires at least one member")
	}
	previous := ""
	for index, value := range values {
		object, err := requireObjectValue(value, fmt.Sprintf("GitRemote[%d]", index))
		if err != nil {
			return err
		}
		name, err := validateGitRemote(object)
		if err != nil {
			return err
		}
		if index > 0 && name <= previous {
			return invalidIdentity("GitRemote values must be strictly name-sorted with no duplicates")
		}
		previous = name
	}
	return nil
}

func validateGitRemote(object map[string]any) (string, error) {
	if err := requireExactMembers("GitRemote", object, "name", "fetch_url", "push_url"); err != nil {
		return "", err
	}
	name, err := requireBoundedString(object, "name", 1, 128)
	if err != nil {
		return "", err
	}
	if err := requireSanitizedGitURL(object, "fetch_url"); err != nil {
		return "", err
	}
	if value := object["push_url"]; value != nil {
		if err := requireSanitizedGitURL(object, "push_url"); err != nil {
			return "", err
		}
	}
	return name, nil
}

type gitHeadShape struct {
	mode string
	oid  string
}

func validateGitHead(object map[string]any) (gitHeadShape, error) {
	if err := requireExactMembers("GitHead", object, "mode", "oid", "ref"); err != nil {
		return gitHeadShape{}, err
	}
	mode, err := requireEnum(object, "mode", "branch", "detached", "unborn")
	if err != nil {
		return gitHeadShape{}, err
	}
	oid, oidPresent, err := requireNullableGitOID(object, "oid")
	if err != nil {
		return gitHeadShape{}, err
	}
	ref, refPresent, err := nullableString(object, "ref")
	if err != nil {
		return gitHeadShape{}, err
	}
	if refPresent {
		if _, err := scalar.ParseGitRef(ref); err != nil {
			return gitHeadShape{}, invalidIdentity("member ref: %v", err)
		}
	}
	switch mode {
	case "branch":
		if !oidPresent || !refPresent {
			return gitHeadShape{}, invalidIdentity("branch GitHead requires non-null oid and ref")
		}
	case "detached":
		if !oidPresent || refPresent {
			return gitHeadShape{}, invalidIdentity("detached GitHead requires oid and null ref")
		}
	case "unborn":
		if oidPresent || !refPresent || !strings.HasPrefix(ref, "refs/heads/") {
			return gitHeadShape{}, invalidIdentity("unborn GitHead requires null oid and refs/heads/ ref")
		}
	}
	return gitHeadShape{mode: mode, oid: oid}, nil
}

type gitObjectPackShape struct{ objectFormat string }

func validateGitObjectPack(object map[string]any) (gitObjectPackShape, error) {
	if err := requireExactMembers("GitObjectPack", object,
		"format", "object_format", "blob_id", "blob_descriptor_id", "object_count", "inventory_blob_id", "inventory_blob_descriptor_id"); err != nil {
		return gitObjectPackShape{}, err
	}
	if err := requireExactString(object, "format", "git_pack_v2"); err != nil {
		return gitObjectPackShape{}, err
	}
	objectFormat, err := requireEnum(object, "object_format", "sha1", "sha256")
	if err != nil {
		return gitObjectPackShape{}, err
	}
	for _, name := range []string{"blob_id", "blob_descriptor_id", "inventory_blob_id", "inventory_blob_descriptor_id"} {
		if err := requireDigest(object, name); err != nil {
			return gitObjectPackShape{}, err
		}
	}
	if _, err := requireUint(object, "object_count", scalar.MaxUint53); err != nil {
		return gitObjectPackShape{}, err
	}
	return gitObjectPackShape{objectFormat: objectFormat}, nil
}

type gitIndexEntryShape struct {
	path  string
	stage uint64
	mode  uint64
	oid   string
}

type gitIndexShape struct{ entries []gitIndexEntryShape }

func validateGitIndex(object map[string]any) (gitIndexShape, error) {
	if err := requireExactMembers("GitIndex", object,
		"format", "version", "blob_id", "blob_descriptor_id", "entries", "entry_count"); err != nil {
		return gitIndexShape{}, err
	}
	if err := requireExactString(object, "format", "git_index"); err != nil {
		return gitIndexShape{}, err
	}
	version, err := requireUint(object, "version", 4)
	if err != nil || version < 2 {
		if err != nil {
			return gitIndexShape{}, err
		}
		return gitIndexShape{}, invalidIdentity("GitIndex version must be 2, 3, or 4")
	}
	for _, name := range []string{"blob_id", "blob_descriptor_id"} {
		if err := requireDigest(object, name); err != nil {
			return gitIndexShape{}, err
		}
	}
	values, err := requireArray(object, "entries", 65_536)
	if err != nil {
		return gitIndexShape{}, err
	}
	entries := make([]gitIndexEntryShape, 0, len(values))
	for index, value := range values {
		entry, err := requireObjectValue(value, fmt.Sprintf("GitIndexEntry[%d]", index))
		if err != nil {
			return gitIndexShape{}, err
		}
		shape, err := validateGitIndexEntry(entry)
		if err != nil {
			return gitIndexShape{}, err
		}
		if index > 0 {
			previous := entries[index-1]
			if shape.path < previous.path || shape.path == previous.path && shape.stage <= previous.stage {
				return gitIndexShape{}, invalidIdentity("GitIndex entries must be strictly sorted by path then stage")
			}
		}
		entries = append(entries, shape)
	}
	count, err := requireUint(object, "entry_count", scalar.MaxUint53)
	if err != nil {
		return gitIndexShape{}, err
	}
	if count != uint64(len(entries)) {
		return gitIndexShape{}, invalidIdentity("GitIndex entry_count is %d, want %d", count, len(entries))
	}
	return gitIndexShape{entries: entries}, nil
}

func validateGitIndexEntry(object map[string]any) (gitIndexEntryShape, error) {
	if err := requireExactMembers("GitIndexEntry", object,
		"path", "stage", "mode", "oid", "intent_to_add", "skip_worktree", "assume_unchanged", "fsmonitor_valid"); err != nil {
		return gitIndexEntryShape{}, err
	}
	pathValue, err := requireRelativePathValue(object, "path")
	if err != nil {
		return gitIndexEntryShape{}, err
	}
	stage, err := requireUint(object, "stage", 3)
	if err != nil {
		return gitIndexEntryShape{}, err
	}
	mode, err := requireUint(object, "mode", 1<<32-1)
	if err != nil {
		return gitIndexEntryShape{}, err
	}
	oid, err := requireGitOID(object, "oid")
	if err != nil {
		return gitIndexEntryShape{}, err
	}
	for _, name := range []string{"intent_to_add", "skip_worktree", "assume_unchanged", "fsmonitor_valid"} {
		if _, err := requireBool(object, name); err != nil {
			return gitIndexEntryShape{}, err
		}
	}
	return gitIndexEntryShape{path: pathValue, stage: stage, mode: mode, oid: oid}, nil
}

func validateGitSubmodules(values []any, objectFormat string, parentIndex gitIndexShape, depth int, total *int, ancestors map[string]struct{}) error {
	paths := make([]string, 0, len(values))
	for index, value := range values {
		object, err := requireObjectValue(value, fmt.Sprintf("GitSubmodule[%d]", index))
		if err != nil {
			return err
		}
		pathValue, err := validateGitSubmodule(object, objectFormat, parentIndex, depth, total, ancestors)
		if err != nil {
			return err
		}
		for _, previous := range paths {
			if strings.EqualFold(pathValue, previous) {
				return invalidIdentity("GitSubmodule paths must not duplicate or case-collide")
			}
		}
		paths = append(paths, pathValue)
	}
	return nil
}

func validateGitSubmodule(object map[string]any, parentFormat string, parentIndex gitIndexShape, depth int, total *int, ancestors map[string]struct{}) (string, error) {
	if depth > 16 {
		return "", invalidIdentity("GitSubmodule tree exceeds maximum depth 16")
	}
	(*total)++
	if *total > 256 {
		return "", invalidIdentity("GitSubmodule tree exceeds maximum total count 256")
	}
	if err := requireExactMembers("GitSubmodule", object,
		"path", "repository_identity", "sanitized_url", "gitlink_oid", "initialized", "head", "upstream_ref",
		"object_pack", "index", "working_tree_manifest_id", "submodules", "features", "repo_relative_cwd", "agent_project_config_paths"); err != nil {
		return "", err
	}
	pathValue, err := requireRelativePathValue(object, "path")
	if err != nil {
		return "", err
	}
	repositoryIdentity, err := requireBoundedString(object, "repository_identity", 1, 256)
	if err != nil {
		return "", err
	}
	if _, duplicate := ancestors[repositoryIdentity]; duplicate {
		return "", invalidIdentity("GitSubmodule tree must be acyclic by repository_identity")
	}
	if err := requireSanitizedGitURL(object, "sanitized_url"); err != nil {
		return "", err
	}
	gitlinkOID, err := requireGitOIDForFormat(object, "gitlink_oid", parentFormat)
	if err != nil {
		return "", err
	}
	matchedParent := false
	for _, entry := range parentIndex.entries {
		if entry.path == pathValue && entry.stage == 0 && entry.mode == 57_344 && entry.oid == gitlinkOID {
			matchedParent = true
			break
		}
	}
	if !matchedParent {
		return "", invalidIdentity("GitSubmodule gitlink_oid must equal the containing stage-0 mode-160000 index entry")
	}
	initialized, err := requireBool(object, "initialized")
	if err != nil {
		return "", err
	}
	stateFields := []string{"head", "upstream_ref", "object_pack", "index", "working_tree_manifest_id", "submodules", "features", "repo_relative_cwd", "agent_project_config_paths"}
	if !initialized {
		for _, name := range stateFields {
			if object[name] != nil {
				return "", invalidIdentity("uninitialized GitSubmodule member %s must be null", name)
			}
		}
		return pathValue, nil
	}
	for _, name := range stateFields {
		if name != "upstream_ref" && object[name] == nil {
			return "", invalidIdentity("initialized GitSubmodule member %s must be non-null", name)
		}
	}
	if err := requireNullableGitRef(object, "upstream_ref"); err != nil {
		return "", err
	}
	headObject, _ := requireObject(object, "head")
	head, err := validateGitHead(headObject)
	if err != nil {
		return "", err
	}
	if head.mode == "unborn" {
		return "", invalidIdentity("initialized GitSubmodule head must be branch or detached")
	}
	packObject, _ := requireObject(object, "object_pack")
	pack, err := validateGitObjectPack(packObject)
	if err != nil {
		return "", err
	}
	indexObject, _ := requireObject(object, "index")
	gitIndex, err := validateGitIndex(indexObject)
	if err != nil {
		return "", err
	}
	if err := requireDigest(object, "working_tree_manifest_id"); err != nil {
		return "", err
	}
	featuresObject, _ := requireObject(object, "features")
	features, err := validateGitFeatures(featuresObject)
	if err != nil {
		return "", err
	}
	if pack.objectFormat != features.objectFormat {
		return "", invalidIdentity("GitSubmodule pack and features object formats must match")
	}
	if _, err := scalar.ParseGitOIDForObjectFormat(gitlinkOID, features.objectFormat); err != nil {
		return "", invalidIdentity("GitSubmodule gitlink_oid: %v", err)
	}
	if err := validateGitHeadObjectFormat(head, features.objectFormat); err != nil {
		return "", err
	}
	if err := validateGitIndexObjectFormat(gitIndex, features.objectFormat); err != nil {
		return "", err
	}
	if err := requireDotOrRelativePath(object, "repo_relative_cwd"); err != nil {
		return "", err
	}
	if err := requireSortedUniquePaths(object, "agent_project_config_paths", 256); err != nil {
		return "", err
	}
	nestedValues, err := requireArray(object, "submodules", 256)
	if err != nil {
		return "", err
	}
	nestedAncestors := make(map[string]struct{}, len(ancestors)+1)
	for identity := range ancestors {
		nestedAncestors[identity] = struct{}{}
	}
	nestedAncestors[repositoryIdentity] = struct{}{}
	if err := validateGitSubmodules(nestedValues, features.objectFormat, gitIndex, depth+1, total, nestedAncestors); err != nil {
		return "", err
	}
	return pathValue, nil
}

type gitFeaturesShape struct{ objectFormat string }

func validateGitFeatures(object map[string]any) (gitFeaturesShape, error) {
	if err := requireExactMembers("GitFeatures", object,
		"object_format", "filemode", "symlinks", "case_sensitive", "precompose_unicode", "sparse_checkout",
		"sparse_patterns_blob_id", "sparse_patterns_blob_descriptor_id", "required_filter_names", "lfs_required"); err != nil {
		return gitFeaturesShape{}, err
	}
	objectFormat, err := requireEnum(object, "object_format", "sha1", "sha256")
	if err != nil {
		return gitFeaturesShape{}, err
	}
	for _, name := range []string{"filemode", "symlinks", "case_sensitive", "precompose_unicode", "lfs_required"} {
		if _, err := requireBool(object, name); err != nil {
			return gitFeaturesShape{}, err
		}
	}
	sparse, err := requireBool(object, "sparse_checkout")
	if err != nil {
		return gitFeaturesShape{}, err
	}
	firstPresent := object["sparse_patterns_blob_id"] != nil
	secondPresent := object["sparse_patterns_blob_descriptor_id"] != nil
	if firstPresent {
		if err := requireDigest(object, "sparse_patterns_blob_id"); err != nil {
			return gitFeaturesShape{}, err
		}
	}
	if secondPresent {
		if err := requireDigest(object, "sparse_patterns_blob_descriptor_id"); err != nil {
			return gitFeaturesShape{}, err
		}
	}
	if sparse != (firstPresent && secondPresent) || firstPresent != secondPresent {
		return gitFeaturesShape{}, invalidIdentity("GitFeatures sparse pattern IDs must both be non-null exactly when sparse_checkout is true")
	}
	filters, err := requireArray(object, "required_filter_names", 64)
	if err != nil {
		return gitFeaturesShape{}, err
	}
	if err := validateSortedUniqueStrings(filters, "required_filter_names"); err != nil {
		return gitFeaturesShape{}, err
	}
	return gitFeaturesShape{objectFormat: objectFormat}, nil
}

func validateGitHeadObjectFormat(head gitHeadShape, objectFormat string) error {
	if head.oid == "" {
		return nil
	}
	if _, err := scalar.ParseGitOIDForObjectFormat(head.oid, objectFormat); err != nil {
		return invalidIdentity("GitHead oid: %v", err)
	}
	return nil
}

func validateGitIndexObjectFormat(index gitIndexShape, objectFormat string) error {
	for position, entry := range index.entries {
		if _, err := scalar.ParseGitOIDForObjectFormat(entry.oid, objectFormat); err != nil {
			return invalidIdentity("GitIndexEntry[%d] oid: %v", position, err)
		}
	}
	return nil
}

// validateMigrationProvenance validates only the immutable identity
// contribution from Section 17.3. Publishing, local-reference advancement,
// retention, and configuration migration remain outside this package.
func validateMigrationProvenance(object map[string]any) error {
	extensionsValue, ok := object["extensions"]
	if !ok {
		return nil
	}
	extensions, ok := extensionsValue.(map[string]any)
	if !ok {
		return invalidIdentity("extensions must be a JSON object")
	}
	return validateMigrationExtensionObject(extensions)
}

func validateMigrationExtensionObject(extensions map[string]any) error {
	if err := validateExtensionsObject(extensions); err != nil {
		return err
	}
	value, ok := extensions["works.relux.ax.migrated-from"]
	if !ok {
		return nil
	}
	provenance, err := requireObjectValue(value, `extensions["works.relux.ax.migrated-from"]`)
	if err != nil {
		return err
	}
	if err := requireExactMembers("migration provenance", provenance, "schema_id", "schema_version", "object_id"); err != nil {
		return err
	}
	if _, err := requireUTF8String(provenance, "schema_id"); err != nil {
		return err
	}
	version, err := requireString(provenance, "schema_version")
	if err != nil {
		return err
	}
	if !semverPattern.MatchString(version) {
		return invalidIdentity("migration provenance schema_version must be canonical semver")
	}
	return requireDigest(provenance, "object_id")
}

func validateExtensionsObject(extensions map[string]any) error {
	if len(extensions) > 64 {
		return invalidIdentity("extensions contains %d members, maximum is 64", len(extensions))
	}
	keys := make([]string, 0, len(extensions))
	for key := range extensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(key) < 3 || len(key) > 253 || !reverseDNSPattern.MatchString(key) {
			return invalidIdentity("extensions key %q is not a 3..253 character lowercase reverse-DNS name", key)
		}
		if err := validateExtensionValue(extensions[key], 0); err != nil {
			return invalidIdentity("extensions[%q]: %v", key, err)
		}
	}

	encoded, err := json.Marshal(extensions)
	if err != nil {
		return invalidIdentity("serialize extensions object: %v", err)
	}
	canonical, err := Canonicalize(encoded)
	if err != nil {
		return invalidIdentity("canonicalize extensions object: %v", err)
	}
	if len(canonical) > 65_536 {
		return invalidIdentity("canonical extensions object is %d bytes, maximum is 65536", len(canonical))
	}
	return nil
}

func validateExtensionValue(value any, depth int) error {
	switch typed := value.(type) {
	case nil, bool, json.Number:
		return nil
	case string:
		if !utf8.ValidString(typed) {
			return invalidIdentity("string value must be valid UTF-8")
		}
		return nil
	case []any:
		if depth == 4 {
			return invalidIdentity("value exceeds maximum nesting depth 4")
		}
		for _, member := range typed {
			if err := validateExtensionValue(member, depth+1); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		if depth == 4 {
			return invalidIdentity("value exceeds maximum nesting depth 4")
		}
		for key, member := range typed {
			if !utf8.ValidString(key) {
				return invalidIdentity("object key must be valid UTF-8")
			}
			if err := validateExtensionValue(member, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return invalidIdentity("value uses unsupported JSON type %T", value)
	}
}

func objectString(object map[string]any, name string) (string, bool) {
	value, ok := object[name].(string)
	return value, ok
}

func requireBoundedString(object map[string]any, name string, minimum, maximum int) (string, error) {
	value, err := requireString(object, name)
	if err != nil {
		return "", err
	}
	length := utf8.RuneCountInString(value)
	if length < minimum || length > maximum {
		return "", invalidIdentity("member %s must contain %d..%d Unicode characters", name, minimum, maximum)
	}
	return value, nil
}

func requireRelativePath(object map[string]any, name string) error {
	_, err := requireRelativePathValue(object, name)
	return err
}

func requireRelativePathValue(object map[string]any, name string) (string, error) {
	value, err := requireString(object, name)
	if err != nil {
		return "", err
	}
	if _, err := scalar.ParseRelativePath(value); err != nil {
		return "", invalidIdentity("member %s: %v", name, err)
	}
	return value, nil
}

func requireDotOrRelativePath(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if value == "." {
		return nil
	}
	if _, err := scalar.ParseRelativePath(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

func requireSortedUniquePaths(object map[string]any, name string, maximum int) error {
	values, err := requireArray(object, name, maximum)
	if err != nil {
		return err
	}
	previous := ""
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			return invalidIdentity("member %s[%d] must be a path string", name, index)
		}
		if _, err := scalar.ParseRelativePath(text); err != nil {
			return invalidIdentity("member %s[%d]: %v", name, index, err)
		}
		if index > 0 && text <= previous {
			return invalidIdentity("member %s must be strictly path-sorted and unique", name)
		}
		previous = text
	}
	return nil
}

func requireBool(object map[string]any, name string) (bool, error) {
	value, ok := object[name]
	if !ok {
		return false, invalidIdentity("identity input requires member %s", name)
	}
	boolean, ok := value.(bool)
	if !ok {
		return false, invalidIdentity("member %s must be a boolean", name)
	}
	return boolean, nil
}

func nullableString(object map[string]any, name string) (string, bool, error) {
	value, ok := object[name]
	if !ok {
		return "", false, invalidIdentity("identity input requires member %s", name)
	}
	if value == nil {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok || text == "" || !utf8.ValidString(text) {
		return "", false, invalidIdentity("member %s must be null or a non-empty UTF-8 string", name)
	}
	return text, true, nil
}

func requireGitOID(object map[string]any, name string) (string, error) {
	value, err := requireString(object, name)
	if err != nil {
		return "", err
	}
	if _, err := scalar.ParseGitOID(value); err != nil {
		return "", invalidIdentity("member %s: %v", name, err)
	}
	return value, nil
}

func requireGitOIDForFormat(object map[string]any, name, objectFormat string) (string, error) {
	value, err := requireString(object, name)
	if err != nil {
		return "", err
	}
	if _, err := scalar.ParseGitOIDForObjectFormat(value, objectFormat); err != nil {
		return "", invalidIdentity("member %s: %v", name, err)
	}
	return value, nil
}

func requireNullableGitOID(object map[string]any, name string) (string, bool, error) {
	value, present, err := nullableString(object, name)
	if err != nil || !present {
		return value, present, err
	}
	if _, err := scalar.ParseGitOID(value); err != nil {
		return "", false, invalidIdentity("member %s: %v", name, err)
	}
	return value, true, nil
}

func requireNullableGitRef(object map[string]any, name string) error {
	value, present, err := nullableString(object, name)
	if err != nil || !present {
		return err
	}
	if _, err := scalar.ParseGitRef(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

func requireSanitizedGitURL(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if _, err := scalar.ParseSanitizedGitURL(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

func requireExactMembers(name string, object map[string]any, members ...string) error {
	allowed := make(map[string]struct{}, len(members))
	for _, member := range members {
		allowed[member] = struct{}{}
	}
	unknown := make([]string, 0)
	for member := range object {
		if _, ok := allowed[member]; !ok {
			unknown = append(unknown, member)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return invalidIdentity("%s contains unknown member %q", name, unknown[0])
	}
	for _, member := range members {
		if _, ok := object[member]; !ok {
			return invalidIdentity("%s is missing required member %q", name, member)
		}
	}
	return nil
}

func requireObject(object map[string]any, name string) (map[string]any, error) {
	value, ok := object[name]
	if !ok {
		return nil, invalidIdentity("identity input requires member %s", name)
	}
	return requireObjectValue(value, name)
}

func requireObjectValue(value any, name string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, invalidIdentity("%s must be a JSON object", name)
	}
	return object, nil
}

func requireArray(object map[string]any, name string, maximum int) ([]any, error) {
	value, ok := object[name]
	if !ok {
		return nil, invalidIdentity("identity input requires member %s", name)
	}
	array, ok := value.([]any)
	if !ok {
		return nil, invalidIdentity("member %s must be an array", name)
	}
	if len(array) > maximum {
		return nil, invalidIdentity("member %s exceeds maximum length %d", name, maximum)
	}
	return array, nil
}

func requireString(object map[string]any, name string) (string, error) {
	text, err := requireUTF8String(object, name)
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", invalidIdentity("member %s must be a non-empty UTF-8 string", name)
	}
	return text, nil
}

func requireUTF8String(object map[string]any, name string) (string, error) {
	value, ok := object[name]
	if !ok {
		return "", invalidIdentity("identity input requires member %s", name)
	}
	text, ok := value.(string)
	if !ok || !utf8.ValidString(text) {
		return "", invalidIdentity("member %s must be a UTF-8 string", name)
	}
	return text, nil
}

func requireExactString(object map[string]any, name, expected string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if value != expected {
		return invalidIdentity("member %s is %q, want %q", name, value, expected)
	}
	return nil
}

func requireEnum(object map[string]any, name string, allowed ...string) (string, error) {
	value, err := requireString(object, name)
	if err != nil {
		return "", err
	}
	if _, err := scalar.ParseClosedEnum(value, allowed...); err != nil {
		return "", invalidIdentity("member %s: %v", name, err)
	}
	return value, nil
}

func requireUint(object map[string]any, name string, maximum uint64) (uint64, error) {
	value, ok := object[name]
	if !ok {
		return 0, invalidIdentity("identity input requires member %s", name)
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, invalidIdentity("member %s must be an unsigned integral JSON number", name)
	}
	literal := number.String()
	if strings.ContainsAny(literal, ".eE") || strings.HasPrefix(literal, "-") {
		return 0, invalidIdentity("member %s must be an unsigned integral JSON number", name)
	}
	parsed, err := strconv.ParseUint(literal, 10, 64)
	if err != nil || parsed > maximum {
		return 0, invalidIdentity("member %s must lie in [0, %d]", name, maximum)
	}
	return parsed, nil
}

func requireDigest(object map[string]any, name string) error {
	value, err := requireString(object, name)
	if err != nil {
		return err
	}
	if _, err := scalar.ParseDigest(value); err != nil {
		return invalidIdentity("member %s: %v", name, err)
	}
	return nil
}

func requireNullableDigest(object map[string]any, name string) error {
	value, ok := object[name]
	if !ok {
		return invalidIdentity("identity input requires member %s", name)
	}
	if value == nil {
		return nil
	}
	return requireDigest(object, name)
}

func requireNullableDigestPresence(object map[string]any, name string) (bool, error) {
	value, ok := object[name]
	if !ok {
		return false, invalidIdentity("identity input requires member %s", name)
	}
	if value == nil {
		return false, nil
	}
	if err := requireDigest(object, name); err != nil {
		return false, err
	}
	return true, nil
}

func requireUUIDv7(object map[string]any, name string) (string, error) {
	value, err := requireString(object, name)
	if err != nil {
		return "", err
	}
	if _, err := scalar.ParseUUIDv7(value); err != nil {
		return "", invalidIdentity("member %s: %v", name, err)
	}
	return value, nil
}

func requireNullableUUIDv7(object map[string]any, name string) (string, bool, error) {
	value, present, err := nullableString(object, name)
	if err != nil || !present {
		return value, present, err
	}
	if _, err := scalar.ParseUUIDv7(value); err != nil {
		return "", false, invalidIdentity("member %s: %v", name, err)
	}
	return value, true, nil
}

// validateSortedDigests enforces the bare `sorted` phrase: bytewise
// non-descending order, with duplicates ADMITTED. Section 1.6 defines
// `sorted unique T[n..m]` as the compound phrase meaning "bytewise canonical
// ordering and no duplicate", and the document uses bare `sorted` where it does
// not require uniqueness. Refusing a duplicate here would invent a constraint
// the pinned specification does not declare; the phrase-to-validator mapping in
// array_order_inventory_test.go is what keeps the two apart mechanically.
func validateSortedDigests(values []any, name string) error {
	previous := ""
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			return invalidIdentity("member %s[%d] must be a digest string", name, index)
		}
		if _, err := scalar.ParseDigest(text); err != nil {
			return invalidIdentity("member %s[%d]: %v", name, index, err)
		}
		if index > 0 && text < previous {
			return invalidIdentity("member %s must be sorted", name)
		}
		previous = text
	}
	return nil
}

func validateSortedUniqueDigests(values []any, name string) error {
	previous := ""
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			return invalidIdentity("member %s[%d] must be a digest string", name, index)
		}
		if _, err := scalar.ParseDigest(text); err != nil {
			return invalidIdentity("member %s[%d]: %v", name, index, err)
		}
		if index > 0 && text <= previous {
			return invalidIdentity("member %s must be strictly sorted and unique", name)
		}
		previous = text
	}
	return nil
}

func validateSortedUniqueStrings(values []any, name string) error {
	previous := ""
	for index, value := range values {
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) {
			return invalidIdentity("member %s[%d] must be a UTF-8 string", name, index)
		}
		if index > 0 && text <= previous {
			return invalidIdentity("member %s must be strictly sorted and unique", name)
		}
		previous = text
	}
	return nil
}
