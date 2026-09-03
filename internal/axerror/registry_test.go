package axerror

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// pinnedVersionCodeCounts is the measured denominator of this package's code
// registry, taken from the reviewed catalog projection of the pinned Section
// 15.3 tables. It is written down so that a coverage assertion cannot pass by
// measuring an empty registry against itself: 0 of 0 is not 109 of 109.
var pinnedVersionCodeCounts = map[Version]int{
	Version100: 47,
	Version110: 66,
	Version120: 94,
	Version130: 109,
}

const pinnedTotalCodeCount = 109

// TestExitCodeForMatchesThePinnedRegistry measures the registry against every
// catalog row and reports the ratio it measured. Both directions are checked:
// every catalog row resolves through ExitCodeFor to the exit status the catalog
// assigns, and every version admits exactly the number of codes the pinned
// tables carry.
func TestExitCodeForMatchesThePinnedRegistry(test *testing.T) {
	rows := catalog.Current().Errors
	if len(rows) != pinnedTotalCodeCount {
		test.Fatalf("catalog carries %d Structured Error codes, pinned count is %d", len(rows), pinnedTotalCodeCount)
	}
	resolved := 0
	for _, row := range rows {
		for _, rawVersion := range row.ContractVersions {
			version := Version(rawVersion)
			status, err := ExitCodeFor(version, Code(row.Code))
			if err != nil {
				test.Fatalf("ExitCodeFor(%s, %s): %v", version, row.Code, err)
			}
			if status != row.ExitCode {
				test.Fatalf("ExitCodeFor(%s, %s) = %d, catalog assigns %d", version, row.Code, status, row.ExitCode)
			}
			if !IsFailureExitStatus(status) {
				test.Fatalf("code %s resolves to exit %d, which is not a registered failure status", row.Code, status)
			}
			resolved++
		}
	}
	expected := 0
	for _, version := range Versions() {
		codes, err := CodesFor(version)
		if err != nil {
			test.Fatalf("CodesFor(%s): %v", version, err)
		}
		want := pinnedVersionCodeCounts[version]
		if len(codes) != want {
			test.Fatalf("Structured Error %s registers %d codes, pinned count is %d", version, len(codes), want)
		}
		expected += want
	}
	if resolved != expected {
		test.Fatalf("resolved %d of %d registered code-version pairs", resolved, expected)
	}
	test.Logf("code-to-exit registry coverage: %d/%d registered code-version pairs", resolved, expected)
}

// TestExitCodeForRefusesVersionAndCodeDrift narrows the registry gate rather
// than deleting it. Each case would be admitted by a validator that checked
// only membership in the union of all versions, only the code grammar, or only
// the version string's shape.
func TestExitCodeForRefusesVersionAndCodeDrift(test *testing.T) {
	cases := []struct {
		name    string
		version Version
		code    Code
		wantErr error
	}{
		{
			name:    "code added by a later minor is not admitted by an earlier one",
			version: Version100,
			code:    "terminal_backend_not_found",
			wantErr: ErrUnregisteredCode,
		},
		{
			name:    "clone code is not admitted by the base version",
			version: Version100,
			code:    "source_not_quiescent",
			wantErr: ErrUnregisteredCode,
		},
		{
			name:    "directory code is not admitted by the clone version",
			version: Version110,
			code:    "observation_gap",
			wantErr: ErrUnregisteredCode,
		},
		{
			name:    "terminal code is not admitted by the directory version",
			version: Version120,
			code:    "terminal_backend_stale_generation",
			wantErr: ErrUnregisteredCode,
		},
		{
			name:    "well-formed but unminted code is refused",
			version: Version130,
			code:    "realm_broker_unavailable",
			wantErr: ErrUnregisteredCode,
		},
		{
			name:    "unregistered minor is refused",
			version: Version("1.4.0"),
			code:    "not_found",
			wantErr: ErrUnsupportedVersion,
		},
		{
			name:    "unsupported major is refused",
			version: Version("2.0.0"),
			code:    "not_found",
			wantErr: ErrUnsupportedVersion,
		},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			if _, err := ExitCodeFor(item.version, item.code); !errors.Is(err, item.wantErr) {
				test.Fatalf("ExitCodeFor(%s, %s) error = %v, want %v", item.version, item.code, err, item.wantErr)
			}
		})
	}
}

// TestExitStatusRegistryIsClosed checks the Section 15.2 table itself: every
// row the pinned document states is present with its meaning, success is not a
// failure class, and a status the table does not carry is reported unknown
// rather than given a plausible meaning.
func TestExitStatusRegistryIsClosed(test *testing.T) {
	pinned := map[int]string{
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
	if len(exitMeanings) != len(pinned) {
		test.Fatalf("exit registry carries %d rows, the pinned table has %d", len(exitMeanings), len(pinned))
	}
	for status, meaning := range pinned {
		got, ok := ExitStatusMeaning(status)
		if !ok {
			test.Fatalf("exit status %d is missing from the registry", status)
		}
		if got != meaning {
			test.Fatalf("exit status %d means %q, pinned meaning is %q", status, got, meaning)
		}
	}
	if IsFailureExitStatus(0) {
		test.Fatal("success status 0 is reported as a failure class")
	}
	for _, absent := range []int{1, 18, 19, 20, 129, 131, -1} {
		if _, ok := ExitStatusMeaning(absent); ok {
			test.Fatalf("exit status %d is not in the pinned table but the registry carries it", absent)
		}
		if IsFailureExitStatus(absent) {
			test.Fatalf("exit status %d is not registered but is reported as a failure class", absent)
		}
	}
}

// TestBuildCodeRegistryRefusesDriftedCatalogRows narrows the load-time gate.
// Each row would be admitted by a builder that only copied the catalog.
func TestBuildCodeRegistryRefusesDriftedCatalogRows(test *testing.T) {
	valid := catalog.ErrorCode{
		Code:             "not_found",
		ExitCode:         4,
		ContractID:       Schema,
		ContractVersions: []string{"1.0.0"},
	}
	if _, err := buildCodeRegistry([]catalog.ErrorCode{valid}); err != nil {
		test.Fatalf("valid row was refused: %v", err)
	}
	cases := []struct {
		name string
		rows []catalog.ErrorCode
	}{
		{name: "empty catalog", rows: nil},
		{name: "foreign contract", rows: []catalog.ErrorCode{withContract(valid, "urn:ax:schema:cli-result")}},
		{name: "unregistered exit status", rows: []catalog.ErrorCode{withExit(valid, 18)}},
		{name: "success exit status", rows: []catalog.ErrorCode{withExit(valid, 0)}},
		{name: "no version", rows: []catalog.ErrorCode{withVersions(valid)}},
		{name: "unregistered version", rows: []catalog.ErrorCode{withVersions(valid, "1.4.0")}},
		{name: "unsupported major", rows: []catalog.ErrorCode{withVersions(valid, "2.0.0")}},
		{name: "code grammar", rows: []catalog.ErrorCode{withCode(valid, "Not_Found")}},
		{name: "empty code", rows: []catalog.ErrorCode{withCode(valid, "")}},
		{name: "duplicate code", rows: []catalog.ErrorCode{valid, valid}},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			if _, err := buildCodeRegistry(item.rows); err == nil {
				test.Fatal("drifted catalog rows were admitted")
			}
		})
	}
}

func withContract(row catalog.ErrorCode, id catalog.ContractID) catalog.ErrorCode {
	row.ContractID = id
	return row
}

func withExit(row catalog.ErrorCode, status int) catalog.ErrorCode {
	row.ExitCode = status
	return row
}

func withVersions(row catalog.ErrorCode, versions ...string) catalog.ErrorCode {
	row.ContractVersions = versions
	return row
}

func withCode(row catalog.ErrorCode, code catalog.ErrorName) catalog.ErrorCode {
	row.Code = code
	return row
}

// TestMustBuildCodeRegistryFailsClosedOnADriftedCatalog pins the load-time
// behaviour. The registry is package state built at initialisation, so a
// drifted catalog has to stop the program rather than leave a partial registry
// in place for every later caller to resolve codes against.
func TestMustBuildCodeRegistryFailsClosedOnADriftedCatalog(test *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			test.Fatal("a drifted catalog produced a usable registry")
		}
		if !strings.Contains(fmt.Sprint(recovered), "structured error registry is invalid") {
			test.Fatalf("panic did not name the registry failure: %v", recovered)
		}
	}()
	mustBuildCodeRegistry([]catalog.ErrorCode{{
		Code:             "not_found",
		ExitCode:         0,
		ContractID:       Schema,
		ContractVersions: []string{"1.0.0"},
	}})
}
