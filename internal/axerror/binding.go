package axerror

import (
	"errors"
	"fmt"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// ErrUnboundContract reports a containing contract major that binds no
// Structured Error version. It is returned rather than a default, because
// guessing a version here is exactly the negotiation the pinned document
// forbids.
var ErrUnboundContract = errors.New("containing contract major binds no structured error version")

// ContainingContract identifies a contract that embeds Structured Error.
type ContainingContract struct {
	// ID is the Section 1.5 schema identifier of the containing contract.
	ID catalog.ContractID
	// Major is the containing contract's major version.
	Major int
}

// staticBindings is the complete Section 15.1, 15.3, and 17.1 binding table.
// Section 17.1 states the rule this table implements: "None negotiates the
// error schema separately. Compatibility is evaluated first for the containing
// protocol and then against its fixed embedded-error validator."
//
// Every row is quoted from the pinned document:
//   - 15.1: "Provider protocol 2.x, task-board bridge 1.x, and Mesh RPC 2.x
//     each embed exactly Structured Error 1.0.0."
//   - 15.1: "Session Adapter 1.0 and session.clone.* bind Structured Error
//     1.1.0."
//   - 14.2: "Legacy commands select CLI Result 1.0.0 and Structured Error
//     1.0.0; every session.clone.* command selects CLI Result 2.0.0 on success
//     and Structured Error 1.1.0 on failure."
//   - 15.3: Structured Error 1.2.0 "is bound by Directory Node 1 and 2, Mesh
//     RPC 3, CLI Result 3, and Directory Query 1".
//   - 15.3: "Terminal Backend Protocol 1.0.0, Provider Protocol 3.0.0, Mesh RPC
//     4.0.0, and CLI Result 4.0.0 statically bind Structured Error 1.3.0."
var staticBindings = map[ContainingContract]Version{
	{ID: "urn:ax:protocol:provider", Major: 2}:               Version100,
	{ID: "urn:ax:protocol:provider", Major: 3}:               Version130,
	{ID: "urn:ax:protocol:task-board-bridge", Major: 1}:      Version100,
	{ID: "urn:ax:protocol:rpc", Major: 2}:                    Version100,
	{ID: "urn:ax:protocol:rpc", Major: 3}:                    Version120,
	{ID: "urn:ax:protocol:rpc", Major: 4}:                    Version130,
	{ID: "urn:ax:protocol:session-adapter", Major: 1}:        Version110,
	{ID: "urn:ax:schema:cli-result", Major: 1}:               Version100,
	{ID: "urn:ax:schema:cli-result", Major: 2}:               Version110,
	{ID: "urn:ax:schema:cli-result", Major: 3}:               Version120,
	{ID: "urn:ax:schema:cli-result", Major: 4}:               Version130,
	{ID: "urn:ax:protocol:session-directory-node", Major: 1}: Version120,
	{ID: "urn:ax:protocol:session-directory-node", Major: 2}: Version120,
	{ID: "urn:ax:schema:session-directory-query", Major: 1}:  Version120,
	{ID: "urn:ax:protocol:terminal-backend", Major: 1}:       Version130,
}

// BindingFor returns the exact Structured Error version that a containing
// contract major statically binds. There is no negotiation path: the version is
// selected by the containing contract alone, as Section 15.1 requires — "the
// containing protocol version is sufficient to select it".
//
// An unregistered contract or major is refused. A caller cannot fall back to a
// neighbouring major, and no default is returned.
func BindingFor(contract ContainingContract) (Version, error) {
	version, bound := staticBindings[contract]
	if !bound {
		return "", fmt.Errorf("%w: %s major %d", ErrUnboundContract, contract.ID, contract.Major)
	}
	return version, nil
}

// BoundContracts returns every containing contract major that binds a
// Structured Error version, in a stable order.
func BoundContracts() []ContainingContract {
	result := make([]ContainingContract, 0, len(staticBindings))
	for contract := range staticBindings {
		result = append(result, contract)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].ID != result[right].ID {
			return result[left].ID < result[right].ID
		}
		return result[left].Major < result[right].Major
	})
	return result
}
