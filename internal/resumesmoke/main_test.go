package resumesmoke

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

// TestMain gates the helper processes first: when the marker names
// a helper, this binary serves it and exits before the suite runs,
// so a helper child never re-enters the suite or spawns
// grandchildren.
func TestMain(main *testing.M) {
	switch os.Getenv("RESUMESMOKE_HELPER") {
	case "fake-provider":
		runFakeProviderHelper()
	case "crash-store":
		runCrashStoreHelper()
	}
	os.Exit(main.Run())
}

// runFakeProviderHelper serves one provider-protocol frame as the
// fake adapter executable: it reads the request, loads the canned
// body file the parent named for the operation, and prints the
// success envelope with the echoed request ID. Unknown operations
// exit nonzero with no frame, which the host refuses as a process
// failure rather than mistaking for a result.
func runFakeProviderHelper() {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper read: %v\n", err)
		os.Exit(1)
	}
	var frame struct {
		RequestID string          `json:"request_id"`
		Operation string          `json:"operation"`
		Body      json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(stdin, &frame); err != nil {
		fmt.Fprintf(os.Stderr, "helper frame: %v\n", err)
		os.Exit(1)
	}
	path := ""
	switch frame.Operation {
	case "probe":
		path = os.Getenv("RESUMESMOKE_PROBE")
	case "identify-session":
		path = os.Getenv("RESUMESMOKE_IDENTIFY")
	case "resume":
		path = os.Getenv("RESUMESMOKE_RESUME")
	default:
		fmt.Fprintf(os.Stderr, "helper unexpected operation %q\n", frame.Operation)
		os.Exit(1)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper body: %v\n", err)
		os.Exit(1)
	}
	response := map[string]any{
		"protocol":         "urn:ax:protocol:provider",
		"protocol_version": "2.0.0",
		"request_id":       frame.RequestID,
		"ok":               true,
		"body":             json.RawMessage(body),
	}
	rendered, err := json.Marshal(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper response: %v\n", err)
		os.Exit(1)
	}
	rendered = append(rendered, '\n')
	if _, err := os.Stdout.Write(rendered); err != nil {
		fmt.Fprintf(os.Stderr, "helper write: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// runCrashStoreHelper installs one record and dies mid-install: the
// AfterWrite hook signals readiness, then the process sleeps so the
// parent can kill it between the write and the fsync. Exit codes
// are diagnostics; the kill is the test.
func runCrashStoreHelper() {
	input, err := os.ReadFile(os.Getenv("RESUMESMOKE_INPUT"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper input: %v\n", err)
		os.Exit(1)
	}
	ready := os.Getenv("RESUMESMOKE_READY")
	hooks := &StoreHooks{AfterWrite: func(string) error {
		if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "helper ready: %v\n", err)
			os.Exit(1)
		}
		time.Sleep(60 * time.Second)
		return nil
	}}
	if _, err := Store(os.Getenv("RESUMESMOKE_DIR"), input, hooks); err != nil {
		fmt.Fprintf(os.Stderr, "helper store: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
