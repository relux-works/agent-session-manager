package sessprofile

// This file projects the Section 2.4 pairs the fixtures require —
// at the reducer/record level only. Takeover, resume, fork,
// materialization, and bridge transactions stay with their owning
// leaves; those transactions call these projections for the pair
// they must carry and these checks to refuse a divergent one with
// integrity_failure. Every check compares an observed pair against
// the derived expectation: falling back to the creation value past
// a change, omitting the source, or naming a losing, missing, or
// non-newest source each refuses with the violated expectation
// named.

// Finalize activations Section 11.6 binds for the profile fields:
// dormant validation carries nulls, either owner-resumed tag
// carries the exact effective pair fixed by the activated
// session/checkpoint, and fork carries the new Session Record's
// projected creation profile with a null new-session source.
const (
	ActivationDormant          = "dormant_validated"
	ActivationDirectResumed    = "direct_owner_resumed"
	ActivationTaskBoardResumed = "task_board_owner_resumed"
)

// ResolveCreationProfile admits one ax start --profile flag value
// as the Session Record creation profile. Exactly standard and
// yolo pass; anything else — including empty — refuses. The pinned
// specification states no absent-flag default, so defaulting
// belongs to the CLI-surface leaf, which passes this entry an
// explicit value; this entry never guesses one.
func ResolveCreationProfile(flag string) (string, error) {
	if flag != ProfileStandard && flag != ProfileYOLO {
		return "", refuse(ErrInvalidProfile, "creation profile %q is not standard|yolo", flag)
	}
	return flag, nil
}

// BundlePair projects the Task-board Bundle pair (Section 9.3): the
// effective profile and nullable source at the exported checkpoint.
// The caller derives want with DeriveForHeads over the exported
// checkpoint's heads; this projection copies it unchanged —
// export-time omission or normalization is forbidden.
func BundlePair(want Pair) Pair {
	return want
}

// CheckBundlePair binds one observed bundle pair to its derived
// expectation: the TB-BUNDLE-PROFILE-CHANGED-POS projection. A
// stale creation value past a change (N1), a null source where the
// closure holds a change (N2), or a losing, missing, or non-newest
// source (N3 — anything but the derived newest) refuses
// integrity_failure before import.
func CheckBundlePair(profile, source string, hasSource bool, want Pair) error {
	return checkObservedPair("bundle", Pair{Profile: profile, Source: source, HasSource: hasSource}, want)
}

// CheckLaunchPair binds one observed launch pair — provider.launched
// or task_board.launched — to its derived expectation: the
// creation pair for the first launch, the newest authoritative
// change at or before a later launch. The caller derives want over
// the governing closure; this check refuses a divergent value or
// source with integrity_failure.
func CheckLaunchPair(observed, want Pair) error {
	return checkObservedPair("launch", observed, want)
}

// CheckResumedPair binds one observed session.resumed pair to the
// effective pair of its referenced checkpoint's own closure. The
// caller derives want with DeriveForHeads over the referenced
// checkpoint's heads — never over the outer-walk prefix — and this
// check refuses a divergent value or source with integrity_failure.
func CheckResumedPair(observed, want Pair) error {
	return checkObservedPair("resume", observed, want)
}

// ForkProjection projects the fork's new Session Record creation
// profile (Sections 2.4/5.2): the source checkpoint's effective
// profile. The new session's profile_source_event_id is null by
// construction — fork creates a new authority boundary — while
// the source event survives only as
// fork.created.source_profile_event_id provenance, which this
// projection never reads as authority.
func ForkProjection(source Pair) string {
	return source.Profile
}

// CheckForkPair binds one observed fork.created pair with its
// source provenance to the fork expectation: the new-session pair
// must equal the newly persisted creation profile with a null
// source, and the provenance must equal the source checkpoint's
// nullable profile event. Either divergence refuses
// integrity_failure.
func CheckForkPair(observed Pair, provenance string, hasProvenance bool, newCreation string, wantSource string, wantHasSource bool) error {
	if err := checkObservedPair("fork", observed, Pair{Profile: newCreation}); err != nil {
		return err
	}
	if hasProvenance != wantHasSource {
		if hasProvenance {
			return refuse(ErrIntegrity, "fork carries source profile event %s, want no source profile event", provenance)
		}
		return refuse(ErrIntegrity, "fork carries no source profile event, want source profile event %s", wantSource)
	}
	if wantHasSource && provenance != wantSource {
		return refuse(ErrIntegrity, "fork carries source profile event %s, want source profile event %s", provenance, wantSource)
	}
	return nil
}

// CheckResumeRequestProfile binds the execution_profile member of
// one plugin resume request body (Section 7.5) to the effective
// profile at the resumed checkpoint. The request body carries no
// source member; the SpawnPlan's profile_mapping binds through
// the provhost mapping resolver instead.
func CheckResumeRequestProfile(observed string, want Pair) error {
	if observed != want.Profile {
		return refuse(ErrIntegrity, "plugin resume carries execution profile %q, want effective profile %q", observed, want.Profile)
	}
	return nil
}

// FinalizePair projects the materialize.finalize profile fields
// (Section 11.6) for one activation: dormant validation carries
// the null pair, either owner-resumed tag carries the exact
// checkpoint pair, and fork carries the new Session Record's
// projected creation profile with a null new-session source. An
// unknown activation — or a fork under dormant validation, which
// finalizes no owner — refuses.
func FinalizePair(activation string, checkpoint Pair, forkNewCreation string, isFork bool) (Pair, error) {
	switch activation {
	case ActivationDormant:
		if isFork {
			return Pair{}, refuse(ErrDerivation, "dormant validation finalizes no fork owner")
		}
		return Pair{}, nil
	case ActivationDirectResumed, ActivationTaskBoardResumed:
		if isFork {
			if forkNewCreation != ProfileStandard && forkNewCreation != ProfileYOLO {
				return Pair{}, refuse(ErrInvalidProfile, "fork finalize carries creation profile %q, want standard|yolo", forkNewCreation)
			}
			return Pair{Profile: forkNewCreation}, nil
		}
		return checkpoint, nil
	default:
		return Pair{}, refuse(ErrDerivation, "finalize activation %q is not a Section 11.6 activation", activation)
	}
}

// CheckFinalizeParams binds observed materialize.finalize profile
// fields to the projected expectation. A passive finalization that
// carries a profile, or an owner-resumed one that diverges from
// the activated pair, refuses integrity_failure.
func CheckFinalizeParams(profile string, hasProfile bool, source string, hasSource bool, want Pair) error {
	observedProfile := ""
	if hasProfile {
		observedProfile = profile
	}
	wantProfile := ""
	if want.Profile != "" {
		wantProfile = want.Profile
	}
	if hasProfile != (want.Profile != "") || observedProfile != wantProfile {
		if hasProfile {
			return refuse(ErrIntegrity, "finalize carries execution profile %q, want %q", profile, wantProfile)
		}
		return refuse(ErrIntegrity, "finalize carries no execution profile, want %q", wantProfile)
	}
	if hasSource != want.HasSource {
		if hasSource {
			return refuse(ErrIntegrity, "finalize carries profile source %s, want no profile source", source)
		}
		return refuse(ErrIntegrity, "finalize carries no profile source, want newest authoritative source %s", want.Source)
	}
	if want.HasSource && source != want.Source {
		return refuse(ErrIntegrity, "finalize carries profile source %s, want newest authoritative source %s", source, want.Source)
	}
	return nil
}

// CheckBridgeProfile binds one observed bridge profile — the
// task-board launch profile member or the bridge resume --profile
// flag (Section 9.2) — to the effective profile at the exported
// checkpoint. The bridge surface carries no source member; the
// bundle and the journaled events carry the pair.
func CheckBridgeProfile(observed string, want Pair) error {
	if observed != want.Profile {
		return refuse(ErrIntegrity, "bridge resume carries profile %q, want effective profile %q", observed, want.Profile)
	}
	return nil
}

// checkObservedPair binds one observed pair to its derived
// expectation under the named context. The profile value must
// equal the derived effective profile, so falling back to the
// creation value past a change refuses; the source presence must
// match, so a first launch with any source — or a later pair with
// none — refuses; and a present source must equal the derived
// newest, so a missing, out-of-closure, or non-newest reference
// refuses with the violated expectation named.
func checkObservedPair(context string, observed, want Pair) error {
	if observed.Profile != want.Profile {
		return refuse(ErrIntegrity, "%s carries execution profile %q, want effective profile %q", context, observed.Profile, want.Profile)
	}
	if observed.HasSource != want.HasSource {
		if observed.HasSource {
			return refuse(ErrIntegrity, "%s carries profile source %s, want no profile source", context, observed.Source)
		}
		return refuse(ErrIntegrity, "%s carries no profile source, want newest authoritative source %s", context, want.Source)
	}
	if want.HasSource && observed.Source != want.Source {
		return refuse(ErrIntegrity, "%s carries profile source %s, want newest authoritative source %s", context, observed.Source, want.Source)
	}
	return nil
}
