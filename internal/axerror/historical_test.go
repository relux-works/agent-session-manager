package axerror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// historicalCorpusDirectory holds frozen Structured Error envelopes, one per
// containing contract that binds a version of this schema. They are checked-in
// bytes rather than objects a test builds: an envelope produced by today's
// constructor would only prove that this package agrees with itself, while the
// compatibility claim is that a document written by an older peer stays
// readable and keeps meaning the same thing.
const historicalCorpusDirectory = "testdata/historical"

// historicalEnvelope is one frozen failure object together with the containing
// contract that selects its version. The contract is the input, never the
// document: Section 15.1 says "the containing protocol version is sufficient to
// select it", so the reader is told which contract it is reading and the
// payload's own schema_version is only ever checked against that answer.
type historicalEnvelope struct {
	file     string
	digest   string
	release  string
	contract ContainingContract
	version  Version

	// refusal is the error a bound reader must return for these bytes, or nil
	// when the envelope must remain readable.
	refusal error

	code           Code
	exitCode       int
	retryable      bool
	codeRegistered bool
	detailKeys     []string
}

const (
	providerContract  = catalog.ContractID("urn:ax:protocol:provider")
	bridgeContract    = catalog.ContractID("urn:ax:protocol:task-board-bridge")
	rpcContract       = catalog.ContractID("urn:ax:protocol:rpc")
	directoryContract = catalog.ContractID("urn:ax:schema:session-directory-query")
	adapterContract   = catalog.ContractID("urn:ax:protocol:session-adapter")
	terminalContract  = catalog.ContractID("urn:ax:protocol:terminal-backend")
)

// historicalErrorCorpus covers every distinct Structured Error version the
// pinned binding table carries, each read through a containing contract that
// binds it.
var historicalErrorCorpus = []historicalEnvelope{
	{
		file:           "error-1.0.0-provider-protocol-error.json",
		digest:         "90a6f740747d87d7f3aa34da0024cec3d49cb3ba2851ebbe1da1bd2bafe0d7ce",
		release:        "v0.1.0",
		contract:       ContainingContract{ID: providerContract, Major: 2},
		version:        Version100,
		code:           "provider_protocol_error",
		exitCode:       13,
		codeRegistered: true,
		detailKeys:     []string{"frame", "provider_id"},
	},
	{
		file:           "error-1.0.0-bridge-incompatible-protocol.json",
		digest:         "94389cacda34ef996a8c53091057f98a2bdd7c2a51d1aebc1c7e6ed098092f08",
		release:        "v0.1.0",
		contract:       ContainingContract{ID: bridgeContract, Major: 1},
		version:        Version100,
		code:           "incompatible_protocol",
		exitCode:       6,
		codeRegistered: true,
		detailKeys:     []string{"observed_major", "supported_major"},
	},
	{
		file:           "error-1.1.0-session-adapter-timeout.json",
		digest:         "f4120b076569620b874687e8951adc780a8a83fb8b784b15bc450599fe97957d",
		release:        "v0.3.0",
		contract:       ContainingContract{ID: adapterContract, Major: 1},
		version:        Version110,
		code:           "session_adapter_timeout",
		exitCode:       13,
		retryable:      true,
		codeRegistered: true,
		detailKeys:     []string{"adapter_id", "timeout_ms"},
	},
	{
		file:           "error-1.2.0-operation-uncertain.json",
		digest:         "c6438c58d5627a7dd9c52e105eca7618e56961380eb3aff46e054dae455d8d1e",
		release:        "v0.4.0",
		contract:       ContainingContract{ID: rpcContract, Major: 3},
		version:        Version120,
		code:           "operation_uncertain",
		exitCode:       12,
		codeRegistered: true,
		detailKeys:     []string{"remediation"},
	},
	{
		// The same envelope with retryable forged to true. Section 15.3 says
		// operation_uncertain "is not retry permission: status/recovery
		// inspection is mandatory", so the claim is refused on the reading side
		// and not merely on the writing side.
		file:     "error-1.2.0-operation-uncertain-forged-retry.json",
		digest:   "dc2add321020d955e5dae04e480d72a591d917cb7554aa00e44366c7d104fce8",
		release:  "v0.4.0",
		contract: ContainingContract{ID: rpcContract, Major: 3},
		version:  Version120,
		refusal:  ErrInvalidStructuredError,
	},
	{
		file:           "error-1.2.0-target-auth-missing.json",
		digest:         "0c3293a856abaf9a7a299ada419dcf49abadbf08176506d122a01162a70a7fef",
		release:        "v0.4.3",
		contract:       ContainingContract{ID: directoryContract, Major: 1},
		version:        Version120,
		code:           "target_auth_missing",
		exitCode:       7,
		codeRegistered: true,
		detailKeys: []string{
			"macos_version", "provider_build", "provider_id", "remediation", "tmux_server_generation",
		},
	},
	{
		file:           "error-1.3.0-terminal-backend-capability-unproven.json",
		digest:         "afe240b2702e7547dd0e6169c87e6e661fc06fbd9838146df5e421f899f0ae13",
		release:        "v0.5.0",
		contract:       ContainingContract{ID: terminalContract, Major: 1},
		version:        Version130,
		code:           "terminal_backend_capability_unproven",
		exitCode:       6,
		codeRegistered: true,
		detailKeys:     []string{"evidence", "terminal_backend_id"},
	},
}

func readHistoricalEnvelope(t *testing.T, entry historicalEnvelope) []byte {
	t.Helper()
	document, err := os.ReadFile(filepath.Join(historicalCorpusDirectory, entry.file))
	if err != nil {
		t.Fatalf("read %q: %v", entry.file, err)
	}
	digest := sha256.Sum256(document)
	if got := hex.EncodeToString(digest[:]); got != entry.digest {
		t.Fatalf("%q digest = %s, want the reviewed %s; a historical envelope is frozen bytes, "+
			"so regenerating it changes the compatibility claim rather than restating it",
			entry.file, got, entry.digest)
	}
	return document
}

// TestHistoricalErrorCorpusIsFrozenAndCoversEveryBoundVersion keeps the corpus
// honest in three directions: every reviewed entry exists with its reviewed
// bytes, every file present is reviewed, and every Structured Error version the
// pinned binding table carries appears at least once. A version that gained a
// binding without gaining a frozen envelope is a compatibility claim nobody
// checked.
func TestHistoricalErrorCorpusIsFrozenAndCoversEveryBoundVersion(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(historicalCorpusDirectory)
	if err != nil {
		t.Fatalf("read corpus directory: %v", err)
	}
	var found []string
	for _, entry := range entries {
		found = append(found, entry.Name())
	}
	var reviewed []string
	covered := make(map[Version]int)
	for _, entry := range historicalErrorCorpus {
		reviewed = append(reviewed, entry.file)
		covered[entry.version]++
		readHistoricalEnvelope(t, entry)
	}
	sort.Strings(found)
	sort.Strings(reviewed)
	if !reflect.DeepEqual(found, reviewed) {
		t.Fatalf("corpus files %v, reviewed entries %v", found, reviewed)
	}

	bound := make(map[Version]struct{})
	for _, contract := range BoundContracts() {
		version, err := BindingFor(contract)
		if err != nil {
			t.Fatalf("BindingFor(%v): %v", contract, err)
		}
		bound[version] = struct{}{}
	}
	for version := range bound {
		if covered[version] == 0 {
			t.Fatalf("Structured Error %s is bound by a containing contract and has no frozen envelope", version)
		}
	}
	if len(bound) != 4 || len(covered) != 4 {
		t.Fatalf("corpus covers %d of %d bound versions, want 4 of 4", len(covered), len(bound))
	}
}

// TestHistoricalErrorEnvelopesRemainReadable decodes every frozen envelope
// through the production bound reader and checks each machine answer against
// the reviewed row.
func TestHistoricalErrorEnvelopesRemainReadable(t *testing.T) {
	t.Parallel()

	for _, entry := range historicalErrorCorpus {
		t.Run(entry.file, func(t *testing.T) {
			document := readHistoricalEnvelope(t, entry)
			failure, err := DecodeBound(entry.contract, document)
			if entry.refusal != nil {
				if !errors.Is(err, entry.refusal) {
					t.Fatalf("DecodeBound(%s) error = %v, want %v", entry.file, err, entry.refusal)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeBound(%s) error = %v, want a readable %s envelope", entry.file, err, entry.release)
			}
			if failure.Version() != entry.version {
				t.Fatalf("Version() = %q, want %q", failure.Version(), entry.version)
			}
			if failure.Code() != entry.code {
				t.Fatalf("Code() = %q, want %q", failure.Code(), entry.code)
			}
			if failure.ExitCode() != entry.exitCode {
				t.Fatalf("ExitCode() = %d, want %d", failure.ExitCode(), entry.exitCode)
			}
			if failure.Retryable() != entry.retryable {
				t.Fatalf("Retryable() = %t, want %t", failure.Retryable(), entry.retryable)
			}
			if failure.CodeRegistered() != entry.codeRegistered {
				t.Fatalf("CodeRegistered() = %t, want %t", failure.CodeRegistered(), entry.codeRegistered)
			}
			if !reflect.DeepEqual(failure.DetailKeys(), entry.detailKeys) {
				t.Fatalf("DetailKeys() = %v, want %v", failure.DetailKeys(), entry.detailKeys)
			}
		})
	}
}

// TestHistoricalErrorEnvelopesAreClassifiedWithoutTheirMessages replaces the
// human text of every readable frozen envelope and requires every machine
// answer to be unchanged. Section 15.1: "messages are for humans; automation
// MUST branch on code and exit_code".
func TestHistoricalErrorEnvelopesAreClassifiedWithoutTheirMessages(t *testing.T) {
	t.Parallel()

	replayed := 0
	for _, entry := range historicalErrorCorpus {
		if entry.refusal != nil {
			continue
		}
		replayed++
		t.Run(entry.file, func(t *testing.T) {
			document := readHistoricalEnvelope(t, entry)
			baseline, err := DecodeBound(entry.contract, document)
			if err != nil {
				t.Fatalf("DecodeBound(baseline): %v", err)
			}
			for _, message := range []string{
				"x",
				"not_found",
				`{"code":"not_found","exit_code":4,"retryable":true}`,
				"ОШИБКА",
			} {
				rewritten := rewriteHistoricalMessage(t, document, message)
				if string(rewritten) == string(document) {
					t.Fatalf("rewriting the message to %q changed no byte", message)
				}
				blinded, err := DecodeBound(entry.contract, rewritten)
				if err != nil {
					t.Fatalf("DecodeBound(message %q): %v", message, err)
				}
				if blinded.Code() != baseline.Code() ||
					blinded.ExitCode() != baseline.ExitCode() ||
					blinded.Retryable() != baseline.Retryable() ||
					blinded.CodeRegistered() != baseline.CodeRegistered() ||
					blinded.Version() != baseline.Version() ||
					!reflect.DeepEqual(blinded.DetailKeys(), baseline.DetailKeys()) {
					t.Fatalf("message %q changed a machine answer: %+v vs %+v", message, blinded, baseline)
				}
				if blinded.Message() != message {
					t.Fatalf("Message() = %q, want the rewritten text", blinded.Message())
				}
			}
		})
	}
	if replayed != 6 {
		t.Fatalf("replayed %d readable envelopes, want the reviewed 6", replayed)
	}
}

// TestABoundReaderNeverAdoptsThePayloadDeclaredVersion offers every frozen
// envelope to a containing contract that binds a different version. Section
// 15.1 forbids the payload from selecting its own version, so each cross pair
// must be refused rather than parsed under the version the document declares.
func TestABoundReaderNeverAdoptsThePayloadDeclaredVersion(t *testing.T) {
	t.Parallel()

	crossed := 0
	for _, entry := range historicalErrorCorpus {
		document := readHistoricalEnvelope(t, entry)
		for _, contract := range BoundContracts() {
			bound, err := BindingFor(contract)
			if err != nil {
				t.Fatalf("BindingFor(%v): %v", contract, err)
			}
			if bound == entry.version {
				continue
			}
			crossed++
			if _, err := DecodeBound(contract, document); !errors.Is(err, ErrVersionMismatch) {
				t.Fatalf("DecodeBound(%s under %s major %d, which binds %s) error = %v, want ErrVersionMismatch",
					entry.file, contract.ID, contract.Major, bound, err)
			}
		}
	}
	if crossed == 0 {
		t.Fatal("no cross-version pair was exercised")
	}
}

// TestCodesByExitStatusMeasuresTheFanIn is the measurement behind the claim
// that an exit status narrows a failure to a class and never identifies it. The
// counts are asserted per registered version rather than described.
func TestCodesByExitStatusMeasuresTheFanIn(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		version   Version
		statuses  int
		codes     int
		ambiguous int
	}{
		{version: Version100, statuses: 17, codes: 47, ambiguous: 14},
		{version: Version110, statuses: 17, codes: 66, ambiguous: 14},
		{version: Version120, statuses: 17, codes: 94, ambiguous: 15},
		{version: Version130, statuses: 17, codes: 109, ambiguous: 15},
	} {
		t.Run(string(test.version), func(t *testing.T) {
			groups, err := CodesByExitStatus(test.version)
			if err != nil {
				t.Fatalf("CodesByExitStatus(%s): %v", test.version, err)
			}
			all, err := CodesFor(test.version)
			if err != nil {
				t.Fatalf("CodesFor(%s): %v", test.version, err)
			}
			total, ambiguous := 0, 0
			for status, group := range groups {
				if status == 0 {
					t.Fatalf("%s groups a code under the success status", test.version)
				}
				if !IsFailureExitStatus(status) {
					t.Fatalf("%s groups a code under unregistered status %d", test.version, status)
				}
				if !sort.SliceIsSorted(group, func(left, right int) bool { return group[left] < group[right] }) {
					t.Fatalf("%s exit %d group %v is unsorted", test.version, status, group)
				}
				total += len(group)
				if len(group) > 1 {
					ambiguous++
				}
			}
			if len(groups) != test.statuses || total != test.codes || ambiguous != test.ambiguous {
				t.Fatalf("%s = %d statuses over %d codes with %d ambiguous, want %d/%d/%d",
					test.version, len(groups), total, ambiguous, test.statuses, test.codes, test.ambiguous)
			}
			if total != len(all) {
				t.Fatalf("%s grouped %d codes and registers %d", test.version, total, len(all))
			}
			// Why CodesByExitStatus cannot reach its own ExitCodeFor failure
			// path: every code CodesFor returns for a version is registered by
			// that version, which is the invariant measured here rather than
			// assumed by the projection.
			for _, code := range all {
				if _, err := ExitCodeFor(test.version, code); err != nil {
					t.Fatalf("%s registers %q and cannot resolve its exit status: %v", test.version, code, err)
				}
			}
		})
	}
}

// TestCodesByExitStatusRefusesAnUnregisteredVersion keeps the projection from
// answering for a version the registry does not carry. An empty map would read
// as "this version assigns no code", which is a different fact from "there is
// no such version".
func TestCodesByExitStatusRefusesAnUnregisteredVersion(t *testing.T) {
	t.Parallel()

	for _, version := range []Version{"1.4.0", "2.0.0", "", "1.0"} {
		if _, err := CodesByExitStatus(version); !errors.Is(err, ErrUnsupportedVersion) {
			t.Fatalf("CodesByExitStatus(%q) error = %v, want ErrUnsupportedVersion", version, err)
		}
	}
}

func rewriteHistoricalMessage(t *testing.T, document []byte, message string) []byte {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(document, &decoded); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}
	decoded["message"] = encoded
	rewritten, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("encode rewritten envelope: %v", err)
	}
	return rewritten
}
