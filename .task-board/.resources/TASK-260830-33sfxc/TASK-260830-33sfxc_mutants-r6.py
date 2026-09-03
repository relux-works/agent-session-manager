#!/usr/bin/env python3
"""Round-6 mutation harness for the TASK-260830-33sfxc compatibility gates.

Each mutant narrows or removes exactly one gate, is verified applied and
compiled, and is then measured by the tests that must kill it.
"""
import subprocess, sys, os, json

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))

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
    # --- added by the round-4 rework: a figure published beside a measurement
    # --- must be DERIVED from it. Three instances of the class in one artifact.
    # The exact drift the round-3 reviewer found: the registry holds 74 unique
    # acceptance-case ids and tracecheck prints 74; the README said 73.
    ("readme-ownership-acceptance-cases-restored-to-73", "README.md",
     "normative section keys, 74",
     "normative section keys, 73",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    # The other five figures of the same sentence were correct today and
    # unmeasured for exactly the same reason, so each is mutated too.
    ("readme-ownership-contract-rows-drifted", "README.md",
     "owners for all 60 current",
     "owners for all 61 current",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    ("readme-ownership-normative-section-keys-drifted", "README.md",
     "contract rows, 36 pinned",
     "contract rows, 35 pinned",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    ("readme-ownership-section-bindings-drifted", "README.md",
     "cases, 49 exact section bindings",
     "cases, 48 exact section bindings",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    ("readme-ownership-unowned-sections-drifted", "README.md",
     "coverage, 2 disclosed unowned sections",
     "coverage, 3 disclosed unowned sections",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    ("readme-ownership-fixture-identities-drifted", "README.md",
     "and 30 exact fixture identities",
     "and 29 exact fixture identities",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    ("readme-ownership-compatibility-subset-drifted", "README.md",
     "an owned 55-contract subset",
     "an owned 56-contract subset",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    # The unpinned-figure guard. A figure added to the paragraph that no phrase
    # measures must be reported, not ignored - otherwise the next number to
    # arrive beside the measurement is unmeasured again.
    ("readme-ownership-unmeasured-figure-added", "README.md",
     "Appendix D anchors.",
     "Appendix D anchors, across 12 owning packages.",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    # NARROWING the pin itself rather than the prose: dropping one row from the
    # published-figure list must not quietly shrink the comparison. The guard
    # above is what turns that into a red.
    ("readme-ownership-pin-narrowed-by-one-figure", "internal/traceability/readme_figures_pin_test.go",
     '\t{"current contract rows", func(report Report) int { return report.Contracts }},\n',
     "",
     "./internal/traceability", "TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport"),
    # --- round-4, reviewer Observation A: the refusal calls its own count
    # --- "measured rather than advice" and nothing read the number.
    ("refusal-fan-in-count-replaced-by-a-constant", "internal/cliresult/client.go",
     "\t\texitStatus, len(fanIn[exitStatus]), errorVersion)",
     "\t\texitStatus, len(fanIn[exitStatus])*0+1, errorVersion)",
     "./internal/cliresult", "TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice"),
    ("refusal-fan-in-count-replaced-by-advice", "internal/cliresult/client.go",
     '\treturn fmt.Sprintf(\n\t\t"exit status %d is assigned to %d registered Structured Error %s codes, so the status alone identifies no failure",\n\t\texitStatus, len(fanIn[exitStatus]), errorVersion)',
     '\t_ = fanIn\n\treturn fmt.Sprintf(\n\t\t"exit status %d is assigned to many registered Structured Error %s codes, so the status alone identifies no failure",\n\t\texitStatus, errorVersion)',
     "./internal/cliresult", "TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice"),
    # NARROWING, not deletion: the count stays measured at exit 6, which is the
    # single assertion the round-3 reviewer proposed as the minimal fix. A
    # one-row test would survive this; the sweep is what kills it.
    ("refusal-fan-in-count-narrowed-to-one-status", "internal/cliresult/client.go",
     "\treturn fmt.Sprintf(\n\t\t\"exit status %d is assigned to %d registered Structured Error %s codes, so the status alone identifies no failure\",\n\t\texitStatus, len(fanIn[exitStatus]), errorVersion)",
     "\tcount := len(fanIn[exitStatus])\n\tif exitStatus != 6 {\n\t\tcount = 1\n\t}\n\treturn fmt.Sprintf(\n\t\t\"exit status %d is assigned to %d registered Structured Error %s codes, so the status alone identifies no failure\",\n\t\texitStatus, count, errorVersion)",
     "./internal/cliresult", "TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice"),
    # --- round-4, reviewer Observation B: retry independence was proven at exit
    # --- 12 only, and the one exit-15 envelope any test drove already declared
    # --- retryable = true, so a fabricated branch returned the right answer.
    ("retry-forced-true-at-one-status", "internal/cliresult/client.go",
     "func (reading *Reading) Retryable() bool {\n\treturn reading.failure != nil && reading.failure.Retryable()\n}",
     "func (reading *Reading) Retryable() bool {\n\tif reading.exitStatus == 15 {\n\t\treturn true\n\t}\n\treturn reading.failure != nil && reading.failure.Retryable()\n}",
     "./internal/cliresult", "TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus"),
    ("retry-forced-true-over-a-status-class", "internal/cliresult/client.go",
     "func (reading *Reading) Retryable() bool {\n\treturn reading.failure != nil && reading.failure.Retryable()\n}",
     "func (reading *Reading) Retryable() bool {\n\tif reading.exitStatus == 15 || reading.exitStatus == 20 || reading.exitStatus == 21 || reading.exitStatus == 22 {\n\t\treturn true\n\t}\n\treturn reading.failure != nil && reading.failure.Retryable()\n}",
     "./internal/cliresult", "TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus"),
    # NARROWING: the document decides at exit 12 - exactly where the existing
    # row drives - and the status decides everywhere else.
    ("retry-narrowed-to-the-one-status-already-proven", "internal/cliresult/client.go",
     "func (reading *Reading) Retryable() bool {\n\treturn reading.failure != nil && reading.failure.Retryable()\n}",
     "func (reading *Reading) Retryable() bool {\n\tif reading.exitStatus == 12 {\n\t\treturn reading.failure != nil && reading.failure.Retryable()\n\t}\n\treturn true\n}",
     "./internal/cliresult", "TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus"),
    # The enumerator the round-4 sweeps draw on, narrowed below the interrupted
    # row. The asserted counts are what make that a red.
    ("round-4-failure-status-enumeration-narrowed", "internal/cliresult/client_test.go",
     "\tfor status := 0; status <= 255; status++ {",
     "\tfor status := 0; status <= 20; status++ {",
     "./internal/cliresult", "TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus"),
    ("stderr-member-added-to-the-observation", "internal/cliresult/client.go",
     "type InvocationOutput struct {\n\t// Stdout is the exact bytes the invocation wrote to its standard output.\n\tStdout []byte",
     "type InvocationOutput struct {\n\t// Stdout is the exact bytes the invocation wrote to its standard output.\n\tStdout []byte\n\t// Stderr is the mutant this member exists to be caught by.\n\tStderr []byte",
     "./internal/cliresult", "TestMachineReadingCannotSeeStderr"),
    # --- round-5: the reviewer's blocking finding. The command-agreement guard
    # --- was proven at one ordered pair, and two narrowings survived the whole
    # --- repository suite. Both are reproduced here as permanent rows.
    ("command-agreement-narrowed-to-spare-one-invoked-command", "internal/cliresult/client.go",
     "\tif result.Command() != command {",
     "\tif result.Command() != command && command != CommandList {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # The finding itself: any document CLAIMING to be a takeover result - the one
    # body in this contract carrying adoption and authority semantics - admitted
    # for an invocation that never ran it.
    ("command-agreement-narrowed-to-admit-forged-takeover-documents", "internal/cliresult/client.go",
     "\tif result.Command() != command {",
     "\tif result.Command() != command && result.Command() != CommandTakeover {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # The guard narrowed to exactly the coverage it used to have: enforced only
    # when the invocation is doctor, which is the single pair the old row drove.
    ("command-agreement-narrowed-to-the-single-pair-previously-covered", "internal/cliresult/client.go",
     "\tif result.Command() != command {",
     "\tif command == CommandDoctor && result.Command() != command {",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # The sweep's own enumerator. Dropping the tag the finding named must redden
    # rather than quietly shrink the cross product.
    ("command-agreement-sweep-enumerator-narrowed-in-the-test", "internal/cliresult/client_test.go",
     "\tfor _, command := range implemented {\n\t\tdocuments[command] = mustEmittedSuccess(t, command)\n\t}",
     "\tfor _, command := range implemented {\n\t\tif command == CommandTakeover {\n\t\t\tcontinue\n\t\t}\n\t\tdocuments[command] = mustEmittedSuccess(t, command)\n\t}",
     "./internal/cliresult", "TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus"),
    # The grid's denominator. Measuring the guard's share over the implemented
    # tags instead of over the whole registered vocabulary turns a stated bound
    # into a silently smaller claim.
    ("command-agreement-grid-denominator-narrowed-in-the-test", "internal/cliresult/client_test.go",
     "\tregistered := Commands()\n\timplemented := ImplementedCommands()",
     "\tregistered := ImplementedCommands()\n\timplemented := ImplementedCommands()",
     "./internal/cliresult", "TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary"),
    # --- round-5, reviewer Observation A, taken rather than left as a bound.
    # --- Both duplicate-member gates were proven only on hand-built documents,
    # --- so a payload-SIZE narrowing refused every fixture and admitted a large
    # --- document. Payload size is attacker-controlled on every path either
    # --- decoder is reachable from.
    ("axerror-common-data-model-gate-narrowed-by-payload-size", "internal/axerror/decode.go",
     "\tif err := requireCommonDataModel(data); err != nil {",
     "\tif err := requireCommonDataModel(data); len(data) < 4096 && err != nil {",
     "./internal/axerror", "TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember"),
    ("cliresult-discriminator-gate-narrowed-by-payload-size", "internal/cliresult/client.go",
     "\tif _, err := canonicaljson.Canonicalize(stdout); err != nil {",
     "\tif _, err := canonicaljson.Canonicalize(stdout); len(stdout) < 4096 && err != nil {",
     "./internal/cliresult", "TestADuplicateSchemaMemberCannotSelectTheBranchInALargeDocument"),
    # The padding those two new rows depend on. A fixture that stopped exceeding
    # the short-document range must redden on its own length assertion instead of
    # passing as a smaller row.
    ("large-document-padding-narrowed-in-the-test", "internal/axerror/datamodel_test.go",
     "\tfor index := 0; index < 63; index++ {",
     "\tfor index := 0; index < 3; index++ {",
     "./internal/axerror", "TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember"),
    ("large-result-padding-narrowed-in-the-test", "internal/cliresult/datamodel_test.go",
     "\tfor index := 0; index < 12; index++ {",
     "\tfor index := 0; index < 2; index++ {",
     "./internal/cliresult", "TestADuplicateSchemaMemberCannotSelectTheBranchInALargeDocument"),
    # --- added by the round-6 rework that closed the reviewer's F1: the
    # --- code-to-exit-status agreement in axerror.decodeBody had delete-only
    # --- coverage (1 of 752 ordered pairs at 1.0.0), and relabelling a code out
    # --- of its own exit class disarmed the exit-keyed retryability refusal.
    ("code-to-exit-agreement-dropped", "internal/axerror/decode.go",
     "\t\tif exitCode != expectedExit {",
     "\t\tif false && exitCode != expectedExit {",
     "./internal/axerror", "TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry"),
    # NARROWING: the gate is restricted to exactly the one code its previous
    # single row of coverage drove. This is the mutant the reviewer measured as
    # SURVIVED across all thirteen packages.
    ("code-to-exit-agreement-narrowed-to-the-single-code-previously-covered", "internal/axerror/decode.go",
     "\t\tif exitCode != expectedExit {",
     "\t\tif exitCode != expectedExit && code == \"observation_gap\" {",
     "./internal/axerror", "TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry"),
    ("code-to-exit-agreement-narrowed-to-spare-policy-refused", "internal/axerror/decode.go",
     "\t\tif exitCode != expectedExit {",
     "\t\tif exitCode != expectedExit && code != \"policy_refused\" {",
     "./internal/axerror", "TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry"),
    # The same narrowing on the code the reviewer drove all the way through
    # cliresult.Read to a forged retry claim, measured by the test that closes
    # that bypass at the production entry point rather than at the decoder.
    ("code-to-exit-agreement-narrowed-to-spare-authentication-failed", "internal/axerror/decode.go",
     "\t\tif exitCode != expectedExit {",
     "\t\tif exitCode != expectedExit && code != \"authentication_failed\" {",
     "./internal/cliresult", "TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead"),
    ("code-to-exit-agreement-narrowed-to-one-version", "internal/axerror/decode.go",
     "\t\tif exitCode != expectedExit {",
     "\t\tif exitCode != expectedExit && version == Version100 {",
     "./internal/axerror", "TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry"),
    # The reviewer's non-blocking observation, closed in the same pass: the
    # registered-status admission in decodeExitStatus was driven at four sampled
    # values, so sparing one arbitrary unregistered status survived.
    ("error-decoder-exit-status-gate-narrowed-to-spare-one-status", "internal/axerror/decode.go",
     "\tif !IsFailureExitStatus(status) {",
     "\tif !IsFailureExitStatus(status) && status != 42 {",
     "./internal/axerror", "TestTheExitStatusAdmissionIsSweptOverEveryByteValue"),
    ("error-decoder-exit-status-gate-dropped", "internal/axerror/decode.go",
     "\tif !IsFailureExitStatus(status) {",
     "\tif false && !IsFailureExitStatus(status) {",
     "./internal/axerror", "TestTheExitStatusAdmissionIsSweptOverEveryByteValue"),
    # The sweeps are measured too. An enumerator that quietly stops covering the
    # interrupted row, and a loop that quietly stops covering the registry, must
    # redden on their own asserted figures rather than shrink in silence.
    ("agreement-sweep-enumerator-narrowed-in-the-test", "internal/axerror/agreement_test.go",
     "\t\tif status == successExit {",
     "\t\tif status == successExit || status == 130 {",
     "./internal/axerror", "TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry"),
    ("agreement-sweep-code-loop-narrowed-in-the-test", "internal/axerror/agreement_test.go",
     "\t\tfor _, code := range codes {",
     "\t\tfor _, code := range codes[:1] {",
     "./internal/axerror", "TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry"),
    ("relabel-destination-loop-narrowed-in-the-test", "internal/cliresult/relabel_test.go",
     "\t\t\tfor _, destination := range permittedDestinations {",
     "\t\t\tfor _, destination := range permittedDestinations[:1] {",
     "./internal/cliresult", "TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead"),
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

# NARROWING names the mutants that leave enforcement in place for PART of the
# domain - a restricted class, one of several redundant copies, a reordering, or
# a published claim kept and given a specific wrong value - rather than removing
# the check or the claim outright. A gate whose only evidence is that deleting it
# reddens something has been shown to exist, not to hold.
#
# The definition is the round-3 one, stated explicitly here instead of counted by
# hand: applied to the round-3 harness it selects exactly the fourteen that round
# published, so the round-4 figure extends the same class rather than redefining
# it.
NARROWING = {
    "fan-in-narrowed-to-one-code-per-status",
    "cliresult-member-rule-restored-before-the-identity-check",
    "axerror-closed-decode-restored-before-the-identity-check",
    "axerror-common-data-model-gate-narrowed-to-one-version",
    "axerror-gate-adopts-its-own-canonical-bytes",
    "axerror-gate-narrowed-past-the-identity-check",
    "cliresult-discriminator-data-model-gate-removed",
    "readme-fan-in-row-restored-to-its-published-error",
    "logbook-fan-in-figure-restored-to-its-published-error",
    "exit-status-equality-narrowed-to-registered-codes",
    "exit-status-equality-narrowed-to-spare-one-status",
    "success-at-a-failure-status-narrowed-past-the-interrupted-row",
    "success-at-a-failure-status-narrowed-past-an-ordinary-status",
    "failure-status-enumeration-narrowed-below-the-interrupted-row",
    "readme-ownership-acceptance-cases-restored-to-73",
    "readme-ownership-contract-rows-drifted",
    "readme-ownership-normative-section-keys-drifted",
    "readme-ownership-section-bindings-drifted",
    "readme-ownership-unowned-sections-drifted",
    "readme-ownership-fixture-identities-drifted",
    "readme-ownership-compatibility-subset-drifted",
    "readme-ownership-pin-narrowed-by-one-figure",
    "refusal-fan-in-count-narrowed-to-one-status",
    "retry-forced-true-at-one-status",
    "retry-forced-true-over-a-status-class",
    "retry-narrowed-to-the-one-status-already-proven",
    "round-4-failure-status-enumeration-narrowed",
    "command-agreement-narrowed-to-spare-one-invoked-command",
    "command-agreement-narrowed-to-admit-forged-takeover-documents",
    "command-agreement-narrowed-to-the-single-pair-previously-covered",
    "command-agreement-sweep-enumerator-narrowed-in-the-test",
    "command-agreement-grid-denominator-narrowed-in-the-test",
    "axerror-common-data-model-gate-narrowed-by-payload-size",
    "cliresult-discriminator-gate-narrowed-by-payload-size",
    "large-document-padding-narrowed-in-the-test",
    "large-result-padding-narrowed-in-the-test",
    "code-to-exit-agreement-narrowed-to-the-single-code-previously-covered",
    "code-to-exit-agreement-narrowed-to-spare-policy-refused",
    "code-to-exit-agreement-narrowed-to-spare-authentication-failed",
    "code-to-exit-agreement-narrowed-to-one-version",
    "error-decoder-exit-status-gate-narrowed-to-spare-one-status",
    "agreement-sweep-enumerator-narrowed-in-the-test",
    "agreement-sweep-code-loop-narrowed-in-the-test",
    "relabel-destination-loop-narrowed-in-the-test",
}
declared = {name for name, *_ in MUTANTS}
unknown = sorted(NARROWING - declared)

verify = run(["go", "test", "./internal/cliresult", "./internal/axerror", "./internal/traceability", "-count=1"])
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
    "narrowing": len(NARROWING),
    "narrowing_names_not_declared": unknown,
    "restored_suite_exit_code": verify.returncode,
    "results": results,
}
print(json.dumps(report, indent=2))
sys.exit(0 if killed + subsumed == len(MUTANTS) and verify.returncode == 0 and not unknown else 1)
