package hostchannel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// forbiddenCalls pins the no-durable-write, no-launch, no-listen contract
// at the source level. A channel run opens no files for writing, starts no
// processes, accepts no sockets, and never consults the system trust pool.
// The runtime census below proves the same property behaviorally; a mutant
// that evades this token list must still face that suite.
var forbiddenCalls = map[string][]string{
	"os":      {"WriteFile", "Create", "OpenFile", "Mkdir", "MkdirAll", "MkdirTemp", "CreateTemp", "Remove", "RemoveAll", "Rename", "Chmod", "Chtimes", "Truncate", "NewFile"},
	"io":      {"WriteFile"},
	"os/exec": {"Command", "CommandContext"},
	"net":     {"Listen", "ListenPacket", "Dial", "DialTimeout", "DialContext"},
	"syscall": {"Open", "Creat", "Write", "Mkdir", "Rename", "Unlink"},
	"x509":    {"SystemCertPool"},
}

func TestStaticWriteCensus(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	entries, err := os.ReadDir(filepath.Dir(self))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(filepath.Dir(self), name)
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			for _, banned := range forbiddenCalls[receiver.Name] {
				if selector.Sel.Name == banned {
					t.Errorf("%s calls forbidden %s.%s", name, receiver.Name, banned)
				}
			}
			return true
		})
	}
}

func snapshotTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	tree := map[string][]byte{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		tree[rel] = bytes
		return nil
	}); err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	return tree
}

func tempHCEntries(t *testing.T) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatalf("ReadDir(temp): %v", err)
	}
	names := map[string]struct{}{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "hc-") {
			names[entry.Name()] = struct{}{}
		}
	}
	return names
}

func TestRuntimeWriteCensus(t *testing.T) {
	f := newFixture(t)
	// A stale tripwire from an earlier mutant run must not poison this one.
	evict := filepath.Join(os.TempDir(), "hc-evict")
	_ = os.Remove(evict)
	beforeA := snapshotTree(t, f.dirA)
	beforeB := snapshotTree(t, f.dirB)
	beforeTemp := tempHCEntries(t)
	trustBefore, err := os.ReadFile(f.storeB.TrustPath())
	if err != nil {
		t.Fatalf("ReadFile(trust): %v", err)
	}

	var ran int
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, mutate func(func() error) error) (json.RawMessage, error) {
		if err := mutate(func() error { ran++; return nil }); err != nil {
			return nil, err
		}
		return json.RawMessage(`{}`), nil
	}
	clientConn, serverConn := net.Pipe()
	served := serveAsync(serverConn, f.serverConfig(handler))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	for i := 0; i < 2; i++ {
		response, err := client.Call("health.get", json.RawMessage(`{}`))
		if err != nil || !response.OK() {
			t.Fatalf("Call %d = %+v, %v", i, response, err)
		}
	}
	if ran != 2 {
		t.Fatalf("boundary ran %d times", ran)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
	clientConn.Close()
	serverConn.Close()

	for name, before := range beforeA {
		after, err := os.ReadFile(filepath.Join(f.dirA, name))
		if err != nil || !bytes.Equal(after, before) {
			t.Fatalf("state A file %s changed", name)
		}
	}
	for name, before := range beforeB {
		after, err := os.ReadFile(filepath.Join(f.dirB, name))
		if err != nil || !bytes.Equal(after, before) {
			t.Fatalf("state B file %s changed", name)
		}
	}
	if after := snapshotTree(t, f.dirA); len(after) != len(beforeA) {
		t.Fatal("state A file set changed")
	}
	if after := snapshotTree(t, f.dirB); len(after) != len(beforeB) {
		t.Fatal("state B file set changed")
	}
	trustAfter, err := os.ReadFile(f.storeB.TrustPath())
	if err != nil {
		t.Fatalf("ReadFile(trust): %v", err)
	}
	if !bytes.Equal(trustAfter, trustBefore) {
		t.Fatal("trust store changed during the session")
	}
	for name := range tempHCEntries(t) {
		if _, ok := beforeTemp[name]; !ok {
			t.Fatalf("channel created temp entry %s", name)
		}
	}
	if _, err := os.Stat(evict); !os.IsNotExist(err) {
		t.Fatalf("channel created the eviction tripwire: %v", err)
	}
}
