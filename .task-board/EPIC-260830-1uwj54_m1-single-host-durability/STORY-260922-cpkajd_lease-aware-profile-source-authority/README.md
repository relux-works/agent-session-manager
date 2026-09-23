# STORY-260922-cpkajd: lease-aware-profile-source-authority

## Description
SPEC v0.7.0 2.4 requires that losing-lease or ambiguous events never change the effective profile. BUG-260917-3lddu0 closed that property through append admission only (route b of its rework brief): sessprofile.Derive takes (record, events) and cannot see the lease store, so with the append gate disabled as an instrument a losing-lease profile.changed still yields yolo with the sandbox bypass at Projector.Project. This Story gives the property an owner independent of the append gate.

## Scope
internal/sessprofile derivation-side authority over the profile source, plus whatever lease-store or provenance input it needs. The append-admission gate landed by BUG-260917-3lddu0 stays as it is.

## Acceptance Criteria
A profile.changed event authored under a non-winning or ambiguous lease is never the effective profile source at the derivation entries (LoadProfile, Projector.Project, SetProfile from-end, axpane.deriveProfile), proven with the append gate disabled as an instrument; the property has its own tests and its own narrowing mutant; the registry binds the clause to the owner that actually implements it. The never-minted higher-epoch class (an event whose epoch exceeds the winner, admitted today on trunk and candidate alike) is decided here, since SPEC 2.4 names ambiguous events too.
