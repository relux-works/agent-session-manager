# STORY-260917-158jyi: release-agnostic-catalog-validation-gate

## Description
Make validation command 21 (cataloggen -check) independent of the adopted specification release so a Story that re-points the catalog to a new pinned revision can pass Change Request construction under the control-root suite. Background: CR construction runs the control root (trunk) validation suite against the candidate; the v0.7.0 binding leaf TASK-260916-n9r71p regenerates catalog_gen.go for v0.7.0 and by design refuses the explicit v0.6.0 lock, so trunk command 21 fails on its candidate (CR-TASK-260916-n9r71p-1 validation failed at command 21). The same collision hit the v0.6.0 adoption (TASK-260908-2tkufa integration outcome).

## Scope
internal/catalog/cmd/cataloggen (an adopted-release mode that derives the metadata and lock inputs from the code-level current release), internal/cataloggen, task-board.config.json validation command 21, README tool row, tests.

## Acceptance Criteria
cataloggen accepts a mode that resolves -metadata and -contracts from the code-level current adopted release and behaves byte-identically to the explicit invocation; explicit flags keep working; missing or mismatched current-release inputs are refused; validation command 21 uses the release-agnostic form and is green on trunk; no catalog content changes.
