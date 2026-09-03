package axerror

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// Schema is the exact Structured Error schema identifier of Section 15.1.
const Schema = "urn:ax:schema:error"

// Version is one registered Structured Error version. The set is closed: the
// pinned Section 1.5 registry lists exactly these four, and a value outside it
// is refused rather than coerced to the nearest neighbour.
type Version string

const (
	// Version100 is bound by Provider protocol 2.x, task-board bridge 1.x,
	// Mesh RPC 2.x, and CLI Result 1.0.0.
	Version100 Version = "1.0.0"
	// Version110 is bound by Session Adapter 1.0, session.clone.*, and
	// CLI Result 2.0.0.
	Version110 Version = "1.1.0"
	// Version120 is bound by Directory Node 1 and 2, Mesh RPC 3, CLI Result 3,
	// and Directory Query 1.
	Version120 Version = "1.2.0"
	// Version130 is bound by Terminal Backend Protocol 1.0.0, Provider Protocol
	// 3.0.0, Mesh RPC 4.0.0, and CLI Result 4.0.0.
	Version130 Version = "1.3.0"
)

// Code is one stable lower-snake-case Structured Error code.
type Code string

var (
	// ErrUnsupportedVersion reports a Structured Error version outside the
	// pinned registry, including any major other than 1.
	ErrUnsupportedVersion = errors.New("unsupported structured error version")

	// ErrUnregisteredCode reports a code that the requested version does not
	// register. A writer may never mint one; see RetryabilityRefusal and
	// Decode for the reader's separate tolerance rule.
	ErrUnregisteredCode = errors.New("unregistered structured error code")

	// ErrUnregisteredExit reports an exit status outside the closed Section
	// 15.2 registry, or the success status 0 in a failure object.
	ErrUnregisteredExit = errors.New("unregistered structured error exit code")
)

// versionOrder is the registry order of the four registered versions. Index
// position is the only ordering used; the strings are never compared
// lexically, because "1.10.0" would sort before "1.2.0" if they were.
var versionOrder = []Version{Version100, Version110, Version120, Version130}

// codePattern is the Section 15.1 constraint on code, verbatim: "string[1..128]
// | Stable lower-snake-case registry value".
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

const maxCodeLength = 128

// exitMeanings is the Section 15.2 exit-code registry, closed. The table is
// stated in the pinned document without RFC 2119 keywords, so no clause-level
// coverage is claimed for it; it is reproduced here because every code-to-exit
// mapping in Section 15.3 has to resolve against it.
//
// The row count is deliberately not written here in words.
// TestExitStatusRegistryMatchesThePinnedTableRowForRow measures it from
// internal/specdoc and compares the statuses row for row, because the previous
// comment said "nineteen" against a table with eighteen body rows and that
// number was copied into README prose and a landed traceability comment before
// anything compared the word to the table.
var exitMeanings = map[int]string{
	0:   "Requested operation succeeded",
	2:   "Usage, invalid flag, or interactive choice required",
	3:   "Configuration or local precondition invalid",
	4:   "Session, checkpoint, peer, provider, or task-board identity not found",
	5:   "Workspace/native-store conflict; no silent overwrite",
	6:   "Capability unsupported, unknown, conditional, or incompatible",
	7:   "Authentication/authorization/allowlist failure",
	8:   "SSH/RPC transport failure; staging can be resumable",
	9:   "Integrity, schema, hash, or unsafe-path failure",
	10:  "Ownership/lease/fencing failure",
	11:  "Busy, quiesce timeout, or graceful stop timeout",
	12:  "Staging/materialization/rollback failure",
	13:  "Provider plugin/process failure",
	14:  "Task-board bridge/bundle/validation failure",
	15:  "Partial success: immutable sync succeeded but a requested materialization or peer did not",
	16:  "Explicit policy refusal, including missing destructive confirmation",
	17:  "Contract/schema migration required",
	130: "Interrupted by operator signal before a clean response; inspect authority before retry",
}

// successExit is the one registered status that is not a failure class. A
// Structured Error is a failure object, so it can never carry it.
const successExit = 0

type registryEntry struct {
	exitCode int
	versions map[Version]struct{}
}

// codeRegistry is derived from the reviewed catalog rather than retyped here,
// so a code added to the pinned registry cannot exist in one place only.
var codeRegistry = mustBuildCodeRegistry(catalog.Current().Errors)

func mustBuildCodeRegistry(rows []catalog.ErrorCode) map[Code]registryEntry {
	built, err := buildCodeRegistry(rows)
	if err != nil {
		panic(fmt.Sprintf("structured error registry is invalid: %v", err))
	}
	return built
}

// buildCodeRegistry projects the catalog error rows into the code-to-exit
// registry. It refuses a row that does not belong to the Structured Error
// contract, names an exit status outside Section 15.2, claims success, carries
// no version, names an unregistered version, or repeats a code, so a drifted
// catalog fails at load rather than silently widening the admitted set.
func buildCodeRegistry(rows []catalog.ErrorCode) (map[Code]registryEntry, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("catalog carries no Structured Error rows")
	}
	registry := make(map[Code]registryEntry, len(rows))
	for _, row := range rows {
		code := Code(row.Code)
		if string(row.ContractID) != Schema {
			return nil, fmt.Errorf("code %q belongs to contract %q, not %s", code, row.ContractID, Schema)
		}
		if err := validateCodeGrammar(code); err != nil {
			return nil, err
		}
		if _, ok := exitMeanings[row.ExitCode]; !ok {
			return nil, fmt.Errorf("code %q maps to exit %d, which Section 15.2 does not register", code, row.ExitCode)
		}
		if row.ExitCode == successExit {
			return nil, fmt.Errorf("code %q maps to the success exit status", code)
		}
		if len(row.ContractVersions) == 0 {
			return nil, fmt.Errorf("code %q registers no Structured Error version", code)
		}
		versions := make(map[Version]struct{}, len(row.ContractVersions))
		for _, raw := range row.ContractVersions {
			version := Version(raw)
			if !isRegisteredVersion(version) {
				return nil, fmt.Errorf("code %q names unregistered version %q", code, raw)
			}
			versions[version] = struct{}{}
		}
		if _, duplicate := registry[code]; duplicate {
			return nil, fmt.Errorf("code %q is registered twice", code)
		}
		registry[code] = registryEntry{exitCode: row.ExitCode, versions: versions}
	}
	return registry, nil
}

func validateCodeGrammar(code Code) error {
	if len(code) == 0 || len(code) > maxCodeLength {
		return fmt.Errorf("%w: code %q is not 1..%d bytes", ErrUnregisteredCode, code, maxCodeLength)
	}
	if !codePattern.MatchString(string(code)) {
		return fmt.Errorf("%w: code %q is not lower snake case", ErrUnregisteredCode, code)
	}
	return nil
}

func isRegisteredVersion(version Version) bool {
	for _, candidate := range versionOrder {
		if candidate == version {
			return true
		}
	}
	return false
}

// Versions returns the registered Structured Error versions in registry order.
func Versions() []Version {
	return append([]Version(nil), versionOrder...)
}

// ExitCodeFor returns the exact Section 15.2 exit status that version assigns
// to code. It refuses an unregistered version, a code the pinned registry does
// not carry, and a code that a later version added but the requested one does
// not register, so a 1.3.0-only TerminalBackend code cannot be emitted inside a
// 1.0.0 envelope.
func ExitCodeFor(version Version, code Code) (int, error) {
	if !isRegisteredVersion(version) {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
	entry, known := codeRegistry[code]
	if !known {
		return 0, fmt.Errorf("%w: %q is not a Structured Error code", ErrUnregisteredCode, code)
	}
	if _, admitted := entry.versions[version]; !admitted {
		return 0, fmt.Errorf("%w: %q is not registered by Structured Error %s", ErrUnregisteredCode, code, version)
	}
	return entry.exitCode, nil
}

// ExitStatusMeaning returns the Section 15.2 meaning of a registered exit
// status. The second result is false for any status the registry does not
// carry, so an unregistered status is reported as unknown rather than given a
// plausible meaning.
func ExitStatusMeaning(status int) (string, bool) {
	meaning, ok := exitMeanings[status]
	return meaning, ok
}

// IsFailureExitStatus reports whether status is a registered Section 15.2
// status other than success. Every Structured Error carries one of these.
func IsFailureExitStatus(status int) bool {
	if status == successExit {
		return false
	}
	_, ok := exitMeanings[status]
	return ok
}

// CodesFor returns every code the given version registers, sorted. It is the
// measured denominator of this package's registry coverage; callers that want
// a prose summary instead have to state the ratio themselves.
func CodesFor(version Version) ([]Code, error) {
	if !isRegisteredVersion(version) {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
	var result []Code
	for code, entry := range codeRegistry {
		if _, ok := entry.versions[version]; ok {
			result = append(result, code)
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result, nil
}

// CodesByExitStatus groups a version's registered codes by the Section 15.2
// status they map to, each group sorted.
//
// It exists to make one compatibility fact measurable instead of asserted: the
// mapping from code to exit status is many-to-one, so the status a process
// exits with narrows a failure to a class and never identifies it. A client
// that branches on the status alone is choosing between the codes in one of
// these groups without evidence, and the group sizes say how many. Callers that
// want a prose summary have to state the ratio themselves.
//
// A status the version registers no code for is absent from the result rather
// than present and empty: "no code maps here" is a fact about the registry, and
// an empty group would read as one the caller could still branch on.
func CodesByExitStatus(version Version) (map[int][]Code, error) {
	codes, err := CodesFor(version)
	if err != nil {
		return nil, err
	}
	result := make(map[int][]Code)
	for _, code := range codes {
		exitStatus, err := ExitCodeFor(version, code)
		if err != nil {
			return nil, err
		}
		result[exitStatus] = append(result[exitStatus], code)
	}
	return result, nil
}

// retryabilityRefusalsByExitStatus records the exit classes whose whole meaning
// contradicts the Section 15.1 definition of retryable: "True only when the
// identical request may safely be retried without new authority or
// confirmation". Each entry quotes the Section 15.2 row that licenses it.
var retryabilityRefusalsByExitStatus = map[int]string{
	7:   `exit 7 is "Authentication/authorization/allowlist failure"; the identical request cannot succeed without new authority`,
	16:  `exit 16 is "Explicit policy refusal, including missing destructive confirmation"; the identical request cannot succeed without new confirmation`,
	130: `exit 130 is "Interrupted by operator signal before a clean response; inspect authority before retry"`,
}

// retryabilityRefusalsByCode records the three codes the pinned document
// disqualifies individually, each with the sentence that disqualifies it.
// These are quoted, not inferred: no code is added here because retrying it
// merely looks unwise.
var retryabilityRefusalsByCode = map[Code]string{
	"operation_uncertain":               `Section 15.3: "operation_uncertain is not retry permission: status/recovery inspection is mandatory"`,
	"terminal_backend_stale_generation": `Section 15.3: terminal_backend_stale_generation "requires status/recovery before retry and is never retryable solely because the caller can submit the same bytes"`,
	"transaction_unknown":               `Section 15.1: "Transaction unknown is a parked ambiguous effect, never success or absence"`,
}

// RetryabilityRefusal reports why retryable = true is forbidden for a code, and
// whether such a refusal exists. Both the code and its exit status are
// consulted: an unknown code still retains its envelope's exit class, so a
// reader that cannot resolve the code can still refuse a forged retry claim on
// an authorization, refusal, or interrupt class.
//
// The complement is deliberately not a permission. This function never reports
// that retrying is safe; it reports only where the pinned document forbids the
// claim. Every other code carries whatever the constructing call site declared,
// and this package makes no assertion about it.
func RetryabilityRefusal(code Code, exitStatus int) (string, bool) {
	if reason, forbidden := retryabilityRefusalsByCode[code]; forbidden {
		return reason, true
	}
	if reason, forbidden := retryabilityRefusalsByExitStatus[exitStatus]; forbidden {
		return reason, true
	}
	return "", false
}
