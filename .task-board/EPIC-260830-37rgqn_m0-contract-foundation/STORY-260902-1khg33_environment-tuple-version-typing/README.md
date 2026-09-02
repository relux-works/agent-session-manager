# STORY-260902-1khg33: environment-tuple-version-typing

## Description
EnvironmentTuple adapter_version is validated against a three-part SemVer regex the pinned spec never declares for it, so a legal prerelease version is refused. The constraint is inferred across schemas by field-name similarity, which SS1.6 forbids.

## Scope
Normative scope: §10.1 EnvironmentTuple, SPEC.md:3627-3631 and :3610.

## Acceptance Criteria
Either the constraint is removed and the members are validated as declared, or SemVer 2.0.0 is adopted deliberately with the authorising pinned line quoted verbatim. The choice is recorded as an explicit decision and pinned in both directions.
