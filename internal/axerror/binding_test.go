package axerror

import (
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// TestBindingForPinsEveryDeclaredContainingContract checks the static binding
// table against the pinned rows one by one and reports the measured ratio. The
// table is the whole of the "the containing protocol version is sufficient to
// select it" rule: nothing else in this package chooses a version.
func TestBindingForPinsEveryDeclaredContainingContract(test *testing.T) {
	pinned := map[ContainingContract]Version{
		{ID: "urn:ax:protocol:provider", Major: 2}:               Version100,
		{ID: "urn:ax:protocol:task-board-bridge", Major: 1}:      Version100,
		{ID: "urn:ax:protocol:rpc", Major: 2}:                    Version100,
		{ID: "urn:ax:schema:cli-result", Major: 1}:               Version100,
		{ID: "urn:ax:protocol:session-adapter", Major: 1}:        Version110,
		{ID: "urn:ax:schema:cli-result", Major: 2}:               Version110,
		{ID: "urn:ax:protocol:session-directory-node", Major: 1}: Version120,
		{ID: "urn:ax:protocol:session-directory-node", Major: 2}: Version120,
		{ID: "urn:ax:protocol:rpc", Major: 3}:                    Version120,
		{ID: "urn:ax:schema:cli-result", Major: 3}:               Version120,
		{ID: "urn:ax:schema:session-directory-query", Major: 1}:  Version120,
		{ID: "urn:ax:protocol:terminal-backend", Major: 1}:       Version130,
		{ID: "urn:ax:protocol:provider", Major: 3}:               Version130,
		{ID: "urn:ax:protocol:rpc", Major: 4}:                    Version130,
		{ID: "urn:ax:schema:cli-result", Major: 4}:               Version130,
	}
	bound := BoundContracts()
	if len(bound) != len(pinned) {
		test.Fatalf("binding table carries %d rows, the pinned set has %d", len(bound), len(pinned))
	}
	for contract, want := range pinned {
		got, err := BindingFor(contract)
		if err != nil {
			test.Fatalf("BindingFor(%s major %d): %v", contract.ID, contract.Major, err)
		}
		if got != want {
			test.Fatalf("%s major %d binds %s, the pinned binding is %s", contract.ID, contract.Major, got, want)
		}
	}
	// Every bound contract identifier is a real Section 1.5 contract, so a typo
	// cannot create a binding to a schema that does not exist.
	known := make(map[catalog.ContractID]struct{})
	for _, contract := range catalog.Current().Contracts {
		known[contract.ID] = struct{}{}
	}
	for _, contract := range bound {
		if _, ok := known[contract.ID]; !ok {
			test.Fatalf("binding table names %q, which the v0.5.0 contract registry does not carry", contract.ID)
		}
	}
	test.Logf("static binding coverage: %d/%d pinned containing-contract majors", len(bound), len(pinned))
}

// TestBindingForRefusesUnboundContractMajor narrows the refusal. Each case is a
// contract or major that the pinned document does not bind, including
// neighbours of bound rows, so a table that fell back to the nearest major or
// to a default version would be caught.
func TestBindingForRefusesUnboundContractMajor(test *testing.T) {
	cases := []ContainingContract{
		{ID: "urn:ax:protocol:provider", Major: 1},
		{ID: "urn:ax:protocol:provider", Major: 4},
		{ID: "urn:ax:protocol:rpc", Major: 1},
		{ID: "urn:ax:protocol:rpc", Major: 5},
		{ID: "urn:ax:protocol:task-board-bridge", Major: 2},
		{ID: "urn:ax:schema:cli-result", Major: 5},
		{ID: "urn:ax:protocol:session-adapter", Major: 2},
		{ID: "urn:ax:schema:error", Major: 1},
		{ID: "urn:ax:schema:session-record", Major: 3},
		{ID: "urn:ax:protocol:provider", Major: 0},
	}
	for _, contract := range cases {
		if _, err := BindingFor(contract); !errors.Is(err, ErrUnboundContract) {
			test.Fatalf("%s major %d was bound to a version: %v", contract.ID, contract.Major, err)
		}
	}
}

// TestDecodeBoundSelectsTheVersionFromTheContainerNotThePayload proves that the
// embedded version is chosen by the containing contract. A payload that names a
// different registered version is refused rather than accepted on its own word.
func TestDecodeBoundSelectsTheVersionFromTheContainerNotThePayload(test *testing.T) {
	document := []byte(`{
		"schema": "urn:ax:schema:error",
		"schema_version": "1.3.0",
		"code": "not_found",
		"message": "no such session",
		"exit_code": 4,
		"retryable": false,
		"details": {}
	}`)
	provider2 := ContainingContract{ID: "urn:ax:protocol:provider", Major: 2}
	if _, err := DecodeBound(provider2, document); !errors.Is(err, ErrVersionMismatch) {
		test.Fatalf("a provider 2 envelope accepted a 1.3.0 error payload: %v", err)
	}
	provider3 := ContainingContract{ID: "urn:ax:protocol:provider", Major: 3}
	failure, err := DecodeBound(provider3, document)
	if err != nil {
		test.Fatalf("DecodeBound(provider 3): %v", err)
	}
	if failure.Version() != Version130 {
		test.Fatalf("decoded version is %s, the provider 3 binding is 1.3.0", failure.Version())
	}
	if _, err := DecodeBound(ContainingContract{ID: "urn:ax:protocol:provider", Major: 9}, document); !errors.Is(err, ErrUnboundContract) {
		test.Fatalf("an unbound containing major produced an error object: %v", err)
	}
}
