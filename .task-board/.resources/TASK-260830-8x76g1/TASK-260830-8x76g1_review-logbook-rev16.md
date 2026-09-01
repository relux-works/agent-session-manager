# Logbook — TASK-260830-8x76g1 CR rev16 review (RUN-260901-5205ba)

## Finding worth carrying forward: Section 10.4 "overlapping" is undefined entry-locally

Section 10.4 requires that entries contain "no duplicate, overlapping, or
destination-case-colliding path". Duplicate and case collision are enforced in
`validateManifestEntries`; overlapping is not. A `workspace_tree` Transfer Manifest
with entries `{"path":"a","type":"file"}` and `{"path":"a/b","type":"file"}` — a
regular file carrying a descendant — is attested by both public identity entries.
Verified directly at `CalculateObjectIdentity`.

This was NOT raised as a finding, because the pinned text does not support an
entry-local reading: directory entries are legitimately prefixes of their contents
(the spec's own fragment is `{"path":"src","type":"directory"}`), the partition
concept is named separately as "path-disjoint child manifests" (SPEC.md:5211), and
no `TM-*` normative fixture requires the refusal. If a spec clarification later
defines entry-local overlap, it attaches at `validateManifestEntries` next to the
existing `simpleFoldKey` collision set, and `constraint-enumeration.md:213` is the
row to update.

## Method note: how a 181-mutant claim was bound without re-running 181 mutants

Re-executing both full sweeps costs hours and, per the rev15 reviewer, filled the
root volume with 54 GB of Go build cache. A cheaper binding is sound when the delta
is test-only:

1. Regenerate the mutant corpus from the candidate production file and require
   byte-identity with the producer's corpus (binds *what* was mutated).
2. Require the producer harness's recorded baseline hash to equal the candidate's
   production file (binds *against what*).
3. Execute a stratified sample across both corpora and require row-for-row agreement
   (binds *the classifier*). 16/16 matched here.
4. Note monotonicity: a revision that only adds a test cannot turn a KILLED mutant
   into a SURVIVED one, so the prior reviewed tally can only improve.

Report which mutants were executed and which were bound by argument. Do not present
the bound ones as re-run.

## Method note: a golden identity should be reproduced outside the implementation

`cross-platform-identity.golden.json` was re-derived by a reviewer-written Python
RFC 8785 canonicalizer (UTF-16 code-unit key ordering, ECMAScript string escaping),
omitting `record_id` and hashing. Both representations — LF/ordered and
CRLF/reverse-ordered with an empty key and a U+10000 surrogate pair — reproduce
`sha256:b3cfa25a...` exactly. Without this the golden only proves the Go code agrees
with itself.
