# BUG-260902-32iy1u: stop-inferring-semver-on-environment-tuple

## Description
EnvironmentTuple adapter_version is validated against a three-part SemVer regex that the pinned spec never declares for it, so a legal prerelease version is refused.

Record-schema branch, internal/canonicaljson/closed_shapes.go:29 declares ^(0|[1-9][0-9]*)\.(...)\.(...)$ and applies it at :751. The identical regex is used again at :1739 for migration provenance.

SPEC.md:3627-3631 declares the EnvironmentTuple members without a type. The word SemVer appears only on the Adapter Manifest row at :3610, which is a different object. So the constraint is inferred across schemas by field-name similarity, which SS1.6 explicitly forbids.

Consequence: 1.2.3-rc.1 is refused although nothing in the pinned contract rejects it. Even if SemVer were intended, this pattern is not SemVer 2.0.0 - it omits prerelease and build metadata entirely.

This is the invented-constraint class, cross-schema variety: the rule is not wrong because SemVer is wrong, it is wrong because the contract does not say it here.

## Scope
Normative scope: §10.1 EnvironmentTuple, SPEC.md:3627-3631 and :3610.

## Acceptance Criteria
Either the constraint is removed and the members are presence-and-type validated as declared, or SemVer 2.0.0 is adopted deliberately with the pinned line that authorises it quoted verbatim and prerelease and build metadata accepted. Whichever is chosen is recorded as an explicit decision rather than left implied. A case proves 1.2.3-rc.1 behaves the way the decision says, at the production identity entries, and reddens if the behaviour flips.
