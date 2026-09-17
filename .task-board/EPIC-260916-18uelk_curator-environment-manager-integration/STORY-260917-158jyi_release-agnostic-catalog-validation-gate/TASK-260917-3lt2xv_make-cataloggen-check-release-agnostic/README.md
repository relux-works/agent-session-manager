# TASK-260917-3lt2xv: make-cataloggen-check-release-agnostic

## Description
Add an adopted-release mode to cataloggen that derives the catalog metadata and normative lock inputs from the code-level current release, switch validation command 21 in task-board.config.json to that form, and prove byte-identical output against the explicit invocation plus refusals for missing or mismatched inputs.

## Scope
internal/catalog/cmd/cataloggen, internal/cataloggen, internal/catalog (read-only consumers of Current/ForRelease), task-board.config.json command 21, README tool row, LOGBOOK.

## Acceptance Criteria
go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check exits 0 on trunk and produces byte-identical output to the explicit v0.6.0 invocation; explicit -metadata/-contracts still work and mixing them with -adopted is refused; a missing metadata or lock file for the current release, or a lock whose release differs from the current release, is refused with a distinct message; task-board.config.json command 21 is the -adopted form; every other validation command unchanged; tests drive the production main through its entry with positives and refusals and a narrowing mutant per refusal; no catalog content or generated output changes.
