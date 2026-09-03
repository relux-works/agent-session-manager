package axerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// registeredFailureStatusesFromTheMeaningTable enumerates the Section 15.2
// failure statuses by asking ExitStatusMeaning about every byte value rather
// than by asking IsFailureExitStatus.
//
// The choice is deliberate. The sweep below measures the admission decision
// decodeExitStatus makes, and that decision IS a call to IsFailureExitStatus.
// An oracle built from the same predicate would move with the mutant and the
// sweep would agree with whatever the predicate had been narrowed to. Reading
// the meaning table through the other accessor keeps the two independent, and
// the asserted row count is what catches a mutation that moves both - a status
// added to or removed from exitMeanings changes this count, and the count is
// the Section 15.2 table's eighteen rows less the one success row.
func registeredFailureStatusesFromTheMeaningTable(test *testing.T) []int {
	test.Helper()

	var statuses []int
	for status := 0; status <= 255; status++ {
		if _, registered := ExitStatusMeaning(status); !registered {
			continue
		}
		if status == successExit {
			continue
		}
		statuses = append(statuses, status)
	}
	sort.Ints(statuses)
	const registeredFailureStatusCount = 17
	if len(statuses) != registeredFailureStatusCount {
		test.Fatalf("the Section 15.2 table carries %d failure statuses %v, want %d",
			len(statuses), statuses, registeredFailureStatusCount)
	}
	return statuses
}

// sweptStructuredErrorDocument builds one otherwise-conforming Structured Error
// document carrying exactly the code and exit status a sweep row names.
//
// Everything else about the document is held constant, so the only reason a row
// can be refused is the pair it carries. In particular retryable is false, which
// disarms RetryabilityRefusal for every row: a sweep whose refusals came from the
// retryability gate would measure that gate rather than the code-to-exit-status
// agreement, and the two are separately mutable.
func sweptStructuredErrorDocument(test *testing.T, version Version, code Code, exitStatus int) []byte {
	test.Helper()

	details := map[string]any{}
	if code == "target_auth_missing" {
		// decodeBody requires these five keys for this one code. That check
		// runs after the guard under test, so it cannot mask a refusal - but
		// without them the agreeing row for this code would fail its positive
		// control for an unrelated reason.
		for _, key := range targetAuthMissingKeys {
			details[key] = "measured"
		}
	}
	encoded, err := json.Marshal(map[string]any{
		"schema":         Schema,
		"schema_version": string(version),
		"code":           string(code),
		"message":        "swept by the code-to-exit-status agreement measurement",
		"exit_code":      exitStatus,
		"retryable":      false,
		"details":        details,
	})
	if err != nil {
		test.Fatalf("marshal %s %q at exit %d: %v", version, code, exitStatus, err)
	}
	return encoded
}

// TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry states, as
// a measured ratio over every registered code of every registered version, how
// much of the "a document may not name a code and an exit status from different
// Section 15.2 classes" refusal the one guard in decodeBody actually owns.
//
// The class had one row of coverage: TestDecodeRefusesClosedShapeViolations
// drives observation_gap at exit 5 instead of its own exit 9. One code, one
// wrong status, out of 752 ordered (code, wrong-status) pairs at 1.0.0 alone.
// Narrowing the guard to `exitCode != expectedExit && code == "observation_gap"`
// - restricting it to exactly the one code that single row drives - passed all
// thirteen packages, as did sparing policy_refused or authentication_failed
// individually. Only deleting the guard outright reddened anything, which
// proved it reachable and said nothing about the class.
//
// The gap was a bypass, not only an unmeasured ratio. This guard is the sole
// enforcement of the pairing: nothing downstream re-derives the exit status
// from the code. And the retryability refusal this package publishes keys on
// the exit status for three whole classes (retryabilityRefusalsByExitStatus:
// 7 authorization, 16 refusal, 130 interrupt), so relabelling a code out of its
// own class disarms it. Under the narrowing above, an authentication_failed
// document rewritten to exit 9 and carrying retryable: true was admitted, and
// cliresult.Read handed a machine client a safe-retry claim on an authorization
// failure. TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead
// drives that exact shape through the production entry point.
//
// The sweep here drives every registered code of every registered version at
// every registered Section 15.2 failure status. The agreeing status is required
// to be admitted, and each of the other sixteen is required to be refused with
// this guard's own sentence naming both numbers - so a pair settled by
// decodeExitStatus, by the closed member set, or by the retryability gate is
// not counted as coverage of this guard. Every figure is derived from CodesFor
// and CodesByExitStatus, and the ordered-pair totals are pinned per version, so
// the loop cannot go vacuous or quietly shrink.
func TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry(test *testing.T) {
	test.Parallel()

	statuses := registeredFailureStatusesFromTheMeaningTable(test)

	// The measured denominator, per version: registered codes x wrong statuses.
	// These are the same registry sizes TestCodesByExitStatusMeasuresTheFanIn
	// pins, multiplied by the sixteen statuses that are not a given code's own.
	wrongPairsByVersion := map[Version]int{
		Version100: 752,
		Version110: 1056,
		Version120: 1504,
		Version130: 1744,
	}
	if len(wrongPairsByVersion) != len(Versions()) {
		test.Fatalf("pinned %d ordered-pair totals for %d registered versions",
			len(wrongPairsByVersion), len(Versions()))
	}

	swept := make(map[int]bool, len(statuses))
	for _, status := range statuses {
		swept[status] = true
	}

	admitted, byGuard := 0, 0
	for _, version := range Versions() {
		codes, err := CodesFor(version)
		if err != nil {
			test.Fatalf("CodesFor(%s): %v", version, err)
		}
		if len(codes) == 0 {
			test.Fatalf("%s registers no code, so this sweep would measure nothing", version)
		}

		// The denominator is re-derived from the projection this leaf added,
		// not restated: every code the version registers has to appear in
		// exactly one exit group, and every group's status has to be one of
		// the statuses swept below.
		groups, err := CodesByExitStatus(version)
		if err != nil {
			test.Fatalf("CodesByExitStatus(%s): %v", version, err)
		}
		grouped := 0
		for status, group := range groups {
			if !swept[status] {
				test.Fatalf("%s groups %d codes under exit %d, which is not a swept failure status",
					version, len(group), status)
			}
			grouped += len(group)
		}
		if grouped != len(codes) {
			test.Fatalf("%s registers %d codes and groups %d of them by exit status",
				version, len(codes), grouped)
		}

		versionAdmitted, versionByGuard := 0, 0
		for _, code := range codes {
			expectedExit, err := ExitCodeFor(version, code)
			if err != nil {
				test.Fatalf("%s registers %q and cannot resolve its exit status: %v", version, code, err)
			}
			for _, status := range statuses {
				document := sweptStructuredErrorDocument(test, version, code, status)
				failure, err := Decode(version, document)

				if status == expectedExit {
					if err != nil {
						test.Fatalf("Decode(%s, %q at its own exit %d) was refused: %v",
							version, code, status, err)
					}
					if failure.Code() != code || failure.ExitCode() != status {
						test.Fatalf("Decode(%s, %q at exit %d) read back %q at exit %d",
							version, code, status, failure.Code(), failure.ExitCode())
					}
					if !failure.CodeRegistered() {
						test.Fatalf("Decode(%s, %q) reported a registered code as unrecognized",
							version, code)
					}
					versionAdmitted++
					continue
				}

				if !errors.Is(err, ErrInvalidStructuredError) {
					test.Fatalf("Decode(%s, %q at exit %d) error = %v, want ErrInvalidStructuredError",
						version, code, status, err)
				}
				sentence := fmt.Sprintf("maps to exit %d, document carries %d", expectedExit, status)
				if !strings.Contains(err.Error(), sentence) {
					test.Fatalf("Decode(%s, %q at exit %d) was refused by something other than the "+
						"code-to-exit-status agreement: %v", version, code, status, err)
				}
				versionByGuard++
			}
		}

		if versionAdmitted != len(codes) {
			test.Fatalf("%s admitted %d of its %d codes at their own exit status",
				version, versionAdmitted, len(codes))
		}
		wantWrong := wrongPairsByVersion[version]
		if versionByGuard != wantWrong || versionByGuard != len(codes)*(len(statuses)-1) {
			test.Fatalf("%s: the guard refused %d ordered (code, wrong-status) pairs, want %d pinned "+
				"and %d derived from %d codes x %d statuses",
				version, versionByGuard, wantWrong, len(codes)*(len(statuses)-1), len(codes), len(statuses))
		}
		admitted += versionAdmitted
		byGuard += versionByGuard
	}

	if admitted == 0 || byGuard == 0 {
		test.Fatalf("the sweep classified %d admissions and %d refusals; one of its arms drove nothing",
			admitted, byGuard)
	}
	total := admitted + byGuard
	wantTotal := 0
	for _, version := range Versions() {
		codes, err := CodesFor(version)
		if err != nil {
			test.Fatalf("CodesFor(%s): %v", version, err)
		}
		wantTotal += len(codes) * len(statuses)
	}
	if total != wantTotal {
		test.Fatalf("classified %d of %d ordered (version, code, status) rows", total, wantTotal)
	}
}

// TestTheExitStatusAdmissionIsSweptOverEveryByteValue turns decodeExitStatus's
// registered-status admission into a measured ratio over its whole input domain.
//
// It was driven at four sampled values - {0, 1, 18, 99} - so narrowing the gate
// to `!IsFailureExitStatus(status) && status != 42` admitted an unregistered
// exit status through every reader in this package and passed all thirteen
// packages. The complement of a four-value sample is not a bound.
//
// The document carries a code the registry does not register, on purpose. That
// selects decodeBody's Section 15.3 unknown-code branch, which skips the
// code-to-exit-status agreement entirely, so the only decision left in the row
// is the one being measured: an exit status is admitted if and only if the
// Section 15.2 table registers it as a failure. With a registered code the
// agreement guard would refuse sixteen of the seventeen registered statuses and
// this sweep would measure that guard a second time instead.
//
// The path is remotely reachable without a process at all: DecodeBound reads
// peer-supplied provider, bridge, RPC, session-adapter and terminal-backend
// envelopes, where no exit status corroborates the member and the document's
// own number is the only evidence there is.
func TestTheExitStatusAdmissionIsSweptOverEveryByteValue(test *testing.T) {
	test.Parallel()

	const unregistered Code = "observation_horizon_lost"
	if _, err := ExitCodeFor(Version120, unregistered); !errors.Is(err, ErrUnregisteredCode) {
		test.Fatalf("ExitCodeFor(1.2.0, %q) = %v, want ErrUnregisteredCode: this sweep needs a code "+
			"that takes the unknown-code branch, and this one is now registered", unregistered, err)
	}

	statuses := registeredFailureStatusesFromTheMeaningTable(test)
	wanted := make(map[int]bool, len(statuses))
	for _, status := range statuses {
		wanted[status] = true
	}

	admitted := 0
	for status := 0; status <= 255; status++ {
		document := sweptStructuredErrorDocument(test, Version120, unregistered, status)
		failure, err := Decode(Version120, document)

		if wanted[status] {
			if err != nil {
				test.Fatalf("Decode(exit_code %d) was refused, and the Section 15.2 table registers "+
					"that status as a failure: %v", status, err)
			}
			if failure.ExitCode() != status {
				test.Fatalf("Decode(exit_code %d) read back exit %d", status, failure.ExitCode())
			}
			if failure.CodeRegistered() {
				test.Fatalf("Decode(exit_code %d) reported %q as a registered code", status, unregistered)
			}
			admitted++
			continue
		}

		if !errors.Is(err, ErrUnregisteredExit) {
			test.Fatalf("Decode(exit_code %d) error = %v, want ErrUnregisteredExit: the Section 15.2 "+
				"table assigns that status no failure meaning", status, err)
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("exit_code %d", status)) {
			test.Fatalf("Decode(exit_code %d) refused without naming the status it refused: %v", status, err)
		}
	}
	if admitted != len(statuses) {
		test.Fatalf("over 0..255 the reader admitted %d exit statuses, and the Section 15.2 table "+
			"registers %d failures", admitted, len(statuses))
	}

	// The domain is int32, not a byte. A value outside 0..255 can never be a
	// process exit status, but nothing stops a peer writing one into the member,
	// and DecodeBound reads that member with no process behind it.
	for _, status := range []int{-1, -130, 256, 1000, 65536, 2147483647} {
		document := sweptStructuredErrorDocument(test, Version120, unregistered, status)
		if _, err := Decode(Version120, document); !errors.Is(err, ErrUnregisteredExit) {
			test.Fatalf("Decode(exit_code %d) error = %v, want ErrUnregisteredExit", status, err)
		}
	}
}
