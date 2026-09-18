package axpane

import "fmt"

// Wrapper errors are static, narrow causes: they name the failed
// member and the violated rule, never credentials, generations,
// native references, or terminal output. Landed-gate errors pass
// through as the decision Cause unchanged; these exist only for the
// wrapper's own composition arms.

type errInvalidArguments struct{ detail string }

func (err errInvalidArguments) Error() string { return "invalid_arguments: " + err.detail }

type errWrapperMode string

func (mode errWrapperMode) Error() string {
	return fmt.Sprintf("wrapper mode %q is not launch or restore", string(mode))
}

type errUnknownSession string

func (session errUnknownSession) Error() string {
	return fmt.Sprintf("logical session %q is unknown", string(session))
}

type errIdempotencyChanged string

func (session errIdempotencyChanged) Error() string {
	return fmt.Sprintf("bootstrap operation for session %q changed inside the bootstrap window", string(session))
}

type errMaterializationState struct {
	phase    string
	admitted bool
}

func (err errMaterializationState) Error() string {
	if !err.admitted {
		return "required materialization is not admitted"
	}
	return fmt.Sprintf("required materialization is in phase %q, want committed", err.phase)
}

func errMaterializationNotValid(phase string, admitted bool) error {
	return errMaterializationState{phase: phase, admitted: admitted}
}

type errMaterializationStaleValue struct{}

func (err errMaterializationStaleValue) Error() string {
	return "required materialization is sourced from a superseded checkpoint"
}

func errMaterializationStale() error { return errMaterializationStaleValue{} }

type errCheckpoint struct{ detail string }

func (err errCheckpoint) Error() string { return err.detail }

func errCheckpointAbsent() error {
	return errCheckpoint{detail: "resume requires a checkpoint and none is admitted"}
}

func errCheckpointMismatch() error {
	return errCheckpoint{detail: "admitted checkpoint identity does not match the required checkpoint"}
}

func errCheckpointNoPublished() error {
	return errCheckpoint{detail: "resume requires a published checkpoint and the chain names none"}
}

func errCheckpointNotNewest() error {
	return errCheckpoint{detail: "required checkpoint is not the newest published checkpoint"}
}

func errSuccessorNullFold() error {
	return errCheckpoint{detail: "winning lease carries a checkpoint and the chain publishes none"}
}

func errMaterializationNoCheckpoint() error {
	return errMaterializationSource{detail: "required materialization names no published checkpoint to bind"}
}

type errMaterializationSource struct{ detail string }

func (err errMaterializationSource) Error() string { return err.detail }

func errCheckpointForeign() error {
	return errCheckpoint{detail: "checkpoint names another session"}
}

func errCheckpointMembers() error {
	return errCheckpoint{detail: "checkpoint carries no readable bound members"}
}

type errIdentity struct{ detail string }

func (err errIdentity) Error() string { return err.detail }

func errIdentityForeign() error {
	return errIdentity{detail: "provider identity names another session"}
}

func errIdentityMembers() error {
	return errIdentity{detail: "provider identity carries no readable bound members"}
}

type errSmoke struct{ detail string }

func (err errSmoke) Error() string { return err.detail }

func errSmokeVerdict(verdict any) error {
	return errSmoke{detail: fmt.Sprintf("provider-auth smoke verdict is %v, want pass", verdict)}
}

func errSmokeTuple() error {
	return errSmoke{detail: "provider-auth smoke binds another probed build"}
}

func errSmokeTargetBuild() error {
	return errSmoke{detail: "provider-auth smoke target binds another probed build"}
}

func errSmokeTargetGeneration() error {
	return errSmoke{detail: "provider-auth smoke target binds a stale server generation"}
}

func errSmokeTargetOS() error {
	return errSmoke{detail: "provider-auth smoke target binds another OS version"}
}

type errRealmValue struct{ detail string }

func (err errRealmValue) Error() string { return err.detail }

func errNoRealmEvidence() error {
	return errRealmValue{detail: "no admitted credential realm evidence authorizes the caller"}
}

func errRealmGenerationStale() error {
	return errRealmValue{detail: "caller realm generation is stale"}
}

func errRealmBinding() error {
	return errRealmValue{detail: "credential realm evidence binds another host binding or provider build"}
}

type errHeadlessValue struct{}

func (err errHeadlessValue) Error() string {
	return "non-interactive create requires headless_creation"
}

func errHeadlessRequired() error { return errHeadlessValue{} }

type errUnknownCaller string

func (caller errUnknownCaller) Error() string {
	return fmt.Sprintf("caller realm %q is not foreground or background", string(caller))
}
