#!/usr/bin/env python3
"""Mutation harness for the TASK-260830-33sfxc compatibility gates.

Each mutant narrows or removes exactly one gate, is verified applied and
compiled, and is then measured by the tests that must kill it.
"""
import subprocess, sys, os, json

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))

MUTANTS = [
    ("exit-code-equality-dropped", "internal/cliresult/client.go",
     "\tif failure.ExitCode() != output.ExitStatus {",
     "\tif false {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    ("absence-collapsed-into-read-failure", "internal/cliresult/client.go",
     'return "", fmt.Errorf("%w: %s", ErrAbsentDocument, exitStatusIsNotEnough(exitStatus, errorVersion))',
     'return "", fmt.Errorf("%w: %s", ErrUnreadableDocument, exitStatusIsNotEnough(exitStatus, errorVersion))',
     "./internal/cliresult", "TestReadDistinguishesAbsenceFromAReadFailure"),
    ("success-admitted-at-a-failure-status", "internal/cliresult/client.go",
     "\tif output.ExitStatus != SuccessExitStatus {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "\tif false {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # Declared subsumed below: with this guard removed, the exit_code equality
    # check refuses the same input, because decodeExitStatus never admits a
    # Structured Error carrying exit_code 0.
    ("failure-admitted-at-exit-zero", "internal/cliresult/client.go",
     "\tif output.ExitStatus == SuccessExitStatus {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a Structured Error, which reports failure, and the process exited 0\",",
     "\tif false {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a Structured Error, which reports failure, and the process exited 0\",",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    ("unregistered-exit-status-admitted", "internal/cliresult/client.go",
     "\tif output.ExitStatus != SuccessExitStatus && !axerror.IsFailureExitStatus(output.ExitStatus) {",
     "\tif output.ExitStatus != SuccessExitStatus && output.ExitStatus == 0 {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    ("command-tag-agreement-dropped", "internal/cliresult/client.go",
     "\tif result.Command() != command {",
     "\tif false {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    ("second-document-on-stdout-admitted", "internal/cliresult/client.go",
     "\tif err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {",
     "\tif err := decoder.Decode(&trailing); false && !errors.Is(err, io.EOF) {",
     "./internal/cliresult", "TestReadDistinguishesAbsenceFromAReadFailure"),
    ("error-version-taken-from-a-fixed-default", "internal/cliresult/client.go",
     "\terrorVersion, err := axerror.BindingFor(axerror.ContainingContract{ID: Schema, Major: major})",
     "\terrorVersion, err := axerror.BindingFor(axerror.ContainingContract{ID: Schema, Major: major*0 + 1})",
     "./internal/cliresult", "TestReadSelectsTheErrorVersionFromTheCommandNotTheDocument"),
    ("fan-in-narrowed-to-one-code-per-status", "internal/axerror/registry.go",
     "\t\tresult[exitStatus] = append(result[exitStatus], code)",
     "\t\tif len(result[exitStatus]) == 0 {\n\t\t\tresult[exitStatus] = append(result[exitStatus], code)\n\t\t}",
     "./internal/axerror", "TestCodesByExitStatusMeasuresTheFanIn"),
    ("unregistered-version-answered-with-an-empty-group", "internal/axerror/registry.go",
     "\tcodes, err := CodesFor(version)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tresult := make(map[int][]Code)",
     "\tcodes, _ := CodesFor(version)\n\tresult := make(map[int][]Code)",
     "./internal/axerror", "TestCodesByExitStatusRefusesAnUnregisteredVersion"),
    ("historical-fixture-regenerated", "internal/cliresult/testdata/historical/error-1.0.0-workspace-conflict.json",
     '"message": "destination differs from its last materialized checkpoint"',
     '"message": "destination differs from the checkpoint"',
     "./internal/cliresult", "TestHistoricalEnvelopesRemainReadable"),
    ("cliresult-member-rule-restored-before-the-identity-check", "internal/cliresult/decode.go",
     "\tif err := verifyEnvelopeIdentity(document, version); err != nil {\n\t\treturn nil, err\n\t}\n\tif err := verifyClosedMembers(document); err != nil {\n\t\treturn nil, err\n\t}",
     "\tif err := verifyClosedMembers(document); err != nil {\n\t\treturn nil, err\n\t}\n\tif err := verifyEnvelopeIdentity(document, version); err != nil {\n\t\treturn nil, err\n\t}",
     "./internal/cliresult", "TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule"),
    ("axerror-closed-decode-restored-before-the-identity-check", "internal/axerror/decode.go",
     "\tif err := verifyEnvelopeIdentity(identity, version); err != nil {\n\t\treturn nil, err\n\t}\n\tdocument, err := decodeClosedDocument(data)\n\tif err != nil {\n\t\treturn nil, err\n\t}",
     "\tdocument, err := decodeClosedDocument(data)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif err := verifyEnvelopeIdentity(identity, version); err != nil {\n\t\treturn nil, err\n\t}",
     "./internal/axerror", "TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule"),
    # --- added by the rework that closed the duplicate-member bypass -------
    ("axerror-common-data-model-gate-removed", "internal/axerror/decode.go",
     "\tif err := requireCommonDataModel(data); err != nil {\n\t\treturn nil, err\n\t}\n",
     "",
     "./internal/axerror", "TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember"),
    # NARROWING, not deletion: the gate still exists and still refuses, for one
    # version out of four. A delete-only mutant would prove the call is reached
    # and say nothing about the class it covers.
    ("axerror-common-data-model-gate-narrowed-to-one-version", "internal/axerror/decode.go",
     "\tif err := requireCommonDataModel(data); err != nil {",
     "\tif err := requireCommonDataModel(data); version == Version100 && err != nil {",
     "./internal/axerror", "TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember"),
    # The widening the discarded canonical bytes exist to prevent: adopting the
    # transform launders 1e1 into 10 before decodeExitStatus reads the token.
    ("axerror-gate-adopts-its-own-canonical-bytes", "internal/axerror/decode.go",
     "\tif err := requireCommonDataModel(data); err != nil {\n\t\treturn nil, err\n\t}",
     "\tif canonical, gateErr := canonicaljson.Canonicalize(data); gateErr != nil {\n\t\treturn nil, gateErr\n\t} else {\n\t\tdata = canonical\n\t}",
     "./internal/axerror", "TestTheCanonicalGateDoesNotLaunderTheExitStatusToken"),
    ("axerror-gate-narrowed-past-the-identity-check", "internal/axerror/decode.go",
     "\tif err := requireCommonDataModel(data); err != nil {\n\t\treturn nil, err\n\t}\n\tidentity, err := parseEnvelopeIdentity(data)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif err := verifyEnvelopeIdentity(identity, version); err != nil {\n\t\treturn nil, err\n\t}",
     "\tidentity, err := parseEnvelopeIdentity(data)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif err := verifyEnvelopeIdentity(identity, version); err != nil {\n\t\treturn nil, err\n\t}\n\tif err := requireCommonDataModel(data); err != nil {\n\t\treturn nil, err\n\t}",
     "./internal/axerror", "TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact"),
    ("cliresult-discriminator-data-model-gate-removed", "internal/cliresult/client.go",
     "\tif _, err := canonicaljson.Canonicalize(stdout); err != nil {",
     "\tif _, err := canonicaljson.Canonicalize(stdout); false && err != nil {",
     "./internal/cliresult", "TestADuplicateSchemaMemberCannotSelectTheBranch"),
    ("absence-aliased-onto-the-read-failure-sentinel", "internal/cliresult/client.go",
     '\tErrAbsentDocument = errors.New("invocation wrote no document to stdout")',
     "\tErrAbsentDocument = ErrUnreadableDocument",
     "./internal/cliresult", "TestReadDistinguishesAbsenceFromAReadFailure"),
    ("readme-fan-in-row-restored-to-its-published-error", "README.md",
     "| `1.2.0` | 17 | 94 | 15 | 14 codes at exit 16 |",
     "| `1.2.0` | 17 | 94 | 15 | 12 codes at exit 6 |",
     "./internal/axerror", "TestREADMEFanInTableIsDerivedFromTheMeasuredProjection"),
    ("readme-fan-in-row-deleted", "README.md",
     "| `1.3.0` | 17 | 109 | 15 | 17 codes at exit 6 |\n",
     "",
     "./internal/axerror", "TestREADMEFanInTableIsDerivedFromTheMeasuredProjection"),
    ("logbook-fan-in-figure-restored-to-its-published-error", "LOGBOOK.md",
     "1.3.0 reaches 109 codes with 17 codes at exit 6",
     "1.3.0 reaches 109 codes with 15 codes at exit 6",
     "./internal/axerror", "TestLogbookFanInFiguresAreDerivedFromTheMeasuredProjection"),
    # --- added by the round-3 rework that closed the two surviving narrowing
    # --- mutants the reviewer measured against the Section 14.2 outcome gates.
    # F1. The gate is present, reachable and green, and this narrowing was
    # equally green: for a code the registry does not carry, axerror.decodeBody
    # runs no exit cross-check at all, so readFailure is the SOLE enforcement of
    # the Section 14.2 equality on exactly that class.
    ("exit-status-equality-narrowed-to-registered-codes", "internal/cliresult/client.go",
     "\tif failure.ExitCode() != output.ExitStatus {",
     "\tif failure.CodeRegistered() && failure.ExitCode() != output.ExitStatus {",
     "./internal/cliresult", "TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry"),
    # F1 again, narrowed by status rather than by registration: sparing one
    # arbitrary registered failure status. A single-sample row would survive it.
    ("exit-status-equality-narrowed-to-spare-one-status", "internal/cliresult/client.go",
     "\tif failure.ExitCode() != output.ExitStatus {",
     "\tif output.ExitStatus != 12 && failure.ExitCode() != output.ExitStatus {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # F2. Exit 130 is the Section 15.2 signal-interruption row, the one status
    # at which a success document plausibly reaches stdout before the process
    # dies, so sparing it is a realistic bypass rather than an artificial one.
    ("success-at-a-failure-status-narrowed-past-the-interrupted-row", "internal/cliresult/client.go",
     "\tif output.ExitStatus != SuccessExitStatus {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "\tif output.ExitStatus != SuccessExitStatus && output.ExitStatus != 130 {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # F2 again at an ordinary failure status, so the row is not proven only at
    # the one status the finding named.
    ("success-at-a-failure-status-narrowed-past-an-ordinary-status", "internal/cliresult/client.go",
     "\tif output.ExitStatus != SuccessExitStatus {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "\tif output.ExitStatus != SuccessExitStatus && output.ExitStatus != 7 {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # The enumerator both new loops draw from. Narrowing its range drops exit
    # 130 out of every loop that uses it; the asserted count is what makes that
    # a red rather than a quietly smaller sweep.
    ("failure-status-enumeration-narrowed-below-the-interrupted-row", "internal/cliresult/client_test.go",
     "\tfor status := 0; status <= 255; status++ {",
     "\tfor status := 0; status <= 20; status++ {",
     "./internal/cliresult", "TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry"),
    ("stderr-member-added-to-the-observation", "internal/cliresult/client.go",
     "type InvocationOutput struct {\n\t// Stdout is the exact bytes the invocation wrote to its standard output.\n\tStdout []byte",
     "type InvocationOutput struct {\n\t// Stdout is the exact bytes the invocation wrote to its standard output.\n\tStdout []byte\n\t// Stderr is the mutant this member exists to be caught by.\n\tStderr []byte",
     "./internal/cliresult", "TestMachineReadingCannotSeeStderr"),
]

def run(cmd, **kw):
    return subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, **kw)

results = []
for name, path, old, new, pkg, test in MUTANTS:
    full = os.path.join(ROOT, path)
    original = open(full).read()
    count = original.count(old)
    if count != 1:
        results.append({"mutant": name, "status": "NOT-APPLIED",
                        "detail": f"marker occurs {count} times in {path}, want exactly 1"})
        continue
    mutated = original.replace(old, new, 1)
    open(full, "w").write(mutated)
    try:
        # Confirm the mutant is PRESENT before believing a green or a red, and
        # assert the MUTATED text rather than only the absence of the original:
        # a rewrite that silently produced the original bytes back would
        # otherwise be measured as a surviving mutant.
        onDisk = open(full).read()
        if onDisk != mutated or onDisk == original:
            results.append({"mutant": name, "status": "NOT-APPLIED",
                            "detail": "the file on disk is not the mutated text"})
            continue
        if new and new not in onDisk:
            results.append({"mutant": name, "status": "NOT-APPLIED",
                            "detail": "the mutated text is absent from the file on disk"})
            continue
        if not new and old in onDisk:
            results.append({"mutant": name, "status": "NOT-APPLIED",
                            "detail": "the deleted text is still present in the file on disk"})
            continue
        build = run(["go", "build", "./..."])
        if build.returncode != 0:
            results.append({"mutant": name, "status": "NOT-COMPILED", "detail": build.stderr.strip()[:400]})
            continue
        measured = run(["go", "test", pkg, "-count=1", "-run", test])
        results.append({
            "mutant": name,
            "gate": path,
            "killed_by": test,
            "status": "KILLED" if measured.returncode != 0 else "SURVIVED",
            "exit_code": measured.returncode,
        })
    finally:
        open(full, "w").write(original)

verify = run(["go", "test", "./internal/cliresult", "./internal/axerror", "-count=1"])
SUBSUMED = {
    "failure-admitted-at-exit-zero":
        "subsumed by the exit_code equality check in readFailure: a Structured Error can never carry "
        "exit_code 0, because decodeExitStatus refuses the success status, so the same input is refused "
        "one guard later with ErrOutcomeDisagreement",
}
for r in results:
    if r["status"] == "SURVIVED" and r["mutant"] in SUBSUMED:
        r["status"] = "SUBSUMED"
        r["subsumed_by"] = SUBSUMED[r["mutant"]]
killed = sum(1 for r in results if r["status"] == "KILLED")
subsumed = sum(1 for r in results if r["status"] == "SUBSUMED")
report = {
    "mutants": len(MUTANTS),
    "killed": killed,
    "declared_subsumed": subsumed,
    "unexplained_survivors": [r["mutant"] for r in results if r["status"] not in ("KILLED", "SUBSUMED")],
    "restored_suite_exit_code": verify.returncode,
    "results": results,
}
print(json.dumps(report, indent=2))
sys.exit(0 if killed + subsumed == len(MUTANTS) and verify.returncode == 0 else 1)
