package hostchannel

import (
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/peeridentity"
)

// ServeArgv is the exact remote responder invocation of Section 11.10.1.
// The version member must equal config.HostChannelVersion; a test pins
// that equality so the two cannot drift.
var ServeArgv = []string{"ax", "rpc", "serve", "--stdio", "--host-channel", "1.0.0"}

// LaunchArgv extends the Section 11.1 fixed SSH command of a configured
// target with the explicit host-channel flag. It starts no process.
func LaunchArgv(target peeridentity.Target) ([]string, error) {
	argv, err := target.RPCArgv()
	if err != nil {
		return nil, err
	}
	return append(argv, "--host-channel", "1.0.0"), nil
}

// LaunchForConfig selects the host-channel launch for one configured peer
// under Configuration 4 only: the snapshot must be a version-4 document
// carrying a host-channel credential binding. Older documents refuse;
// there is no legacy fallback on this path.
func LaunchForConfig(snapshot config.Snapshot, selector string) ([]string, error) {
	loaded, ok := snapshot.Configuration()
	if !ok {
		return nil, fmt.Errorf("%w: no configuration selected", ErrInvalidConfig)
	}
	if loaded.SourceVersion != config.Version4 {
		return nil, fmt.Errorf("%w: host channel requires Configuration 4", ErrInvalidConfig)
	}
	if loaded.Value.Mesh.HostChannel == nil {
		return nil, fmt.Errorf("%w: Configuration 4 carries no host-channel binding", ErrInvalidConfig)
	}
	directory, err := peeridentity.FromSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	target, err := directory.Resolve(selector)
	if err != nil {
		return nil, err
	}
	return LaunchArgv(target)
}
