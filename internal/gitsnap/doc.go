// Package gitsnap captures the live Git repository and index state that
// Sections 10.4 (kind = git WorkspaceSnapshotMember) and 12.1-12.3 of the
// pinned specification require: repository identity, sanitized remotes,
// HEAD/ref mode, worktree metadata, index stages and flags, staged and
// unstaged deltas, and file modes.
//
// The production entry point is Capture. It drives the Git executable
// through the Runner seam (ExecGitRunner in production, a scripted fake in
// tests). It validates repository identity and remote names in characters,
// URLs/refs/index OIDs/index paths through internal/scalar, repository-relative
// cwd, and the closed index/feature domains. This internal result includes
// supplementary native worktree paths and raw delta OIDs; it is not a complete
// canonical wire member. See testdata/string-domain-audit.md for every string
// field and the validation boundary.
//
// ContentOptions enables working-copy bytes, ignored-policy selection, symlink
// targets, recursive submodules and sparse pattern blobs at the same Capture
// entry. The existing localstore, secprim and canonicaljson owners perform
// immutable installation, guarded file opens and descriptor/entry validation.
// Without options Capture retains the accepted repository/index observation API;
// a nil Content is explicitly not a complete workspace capture. AssembleProvisional adds
// pack/raw-index delivery and root/child assembly below coordinated capture.
// Runtime quiescence and checkpoint publication remain unimplemented. See testdata/content-acceptance.md for scope, policy and evidence bounds.
//
// ExecGitRunner overrides inherited optional-lock settings and gives each
// porcelain diff a private index copy, removed when the command returns.
// This preserves real index bytes even when Git refreshes cached stat data.
// A killed process can leave disposable OS-temp files, never a partially
// published snapshot. Git configuration and the Runner are trusted inputs;
// this is not a sandbox for arbitrary external filter programs.
//
// Capture compares HEAD mode/ref/OID, actual index path/file identity/digest
// and version, logical entries and both raw delta streams. These sentinels
// detect observed changes inside their read windows; they do not provide an
// atomic transaction or detect change-and-revert between reads. With content
// options, included files are hashed and re-read. AssembleProvisional verifies
// repository objects, builds manifest closure and repeats the recursive observations;
// held quiescence remains the runtime coordinator responsibility. A refusal returns
// no partial result; retry
// after quiescence is supported.
//
// Refusal codes reuse the specification vocabulary (workspace_conflict,
// incompatible_schema, integrity_failure, unsafe_path,
// capability_unavailable) as diagnostic strings. Mapping them into
// axerror envelopes belongs to the CLI owner, which does not exist yet;
// this package mints no axerror object.
//
// Census blind spot: the gate census in census_test.go counts literal
// refuse(Gate...) call sites in non-test production files after stripping
// comments and string literals. A refusal reached through an aliased
// function variable would be missed by that instrument. The package
// upholds the countermeasure by construction: refuse is always invoked
// directly, never aliased, and a control in the census test proves an
// aliased call is invisible to the instrument so the bound stays stated.
package gitsnap
