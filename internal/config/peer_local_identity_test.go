package config

import (
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"strings"
	"testing"
)

func TestLoadRefusesLocalHostAsPeer(t *testing.T) {
	doc := strings.Replace(string(peerDocumentWithSSHArgs()), testPeerID, testHostID, 1)
	_, err := loadConfigDocument([]byte(doc), scalar.PlatformMacOS, nil)
	requireConfigClause(t, err, "mesh.peers duplicate host_id")
}
