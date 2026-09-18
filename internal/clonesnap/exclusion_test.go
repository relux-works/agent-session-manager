package clonesnap

import (
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

func itoa(value int) string { return strconv.Itoa(value) }

// Exclusion evidence: the always-excluded classes seal as excluded
// rows with stable reasons, and their bytes never reach a blob, a
// manifest, a log line, or an error string — because the walk never
// opens them.

var exclusionSecrets = map[string]string{
	"credential":     "snap-secret-credential-7f3a",
	"machine_auth":   "snap-secret-machine-auth-9b1e",
	"runtime_state":  "snap-secret-runtime-state-4c8d",
	"transient_lock": "snap-secret-transient-lock-2e6f",
}

func exclusionStore(t *testing.T) (string, []PlanItem) {
	t.Helper()
	root := t.TempDir()
	writeStoreFile(t, root, "store/blob-a", []byte("alpha-payload-bytes"))
	plan := []PlanItem{{NativeKey: "store/blob-a", Class: "durable_payload", Required: true}}
	keys := []string{"store/auth-token", "store/lock-file", "store/runtime-sock", "store/session-key"}
	classes := []string{"credential", "transient_lock", "runtime_state", "machine_auth"}
	for index, key := range keys {
		writeStoreFile(t, root, key, []byte(exclusionSecrets[classes[index]]))
		plan = append(plan, PlanItem{NativeKey: key, Class: classes[index], Required: true})
	}
	// Sorted: auth-token, blob-a, lock-file, runtime-sock, session-key.
	ordered := []PlanItem{plan[1], plan[0], plan[2], plan[3], plan[4]}
	return root, ordered
}

func TestCaptureExcludesSecrets(t *testing.T) {
	storeRoot, plan := exclusionStore(t)
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = plan
	result := mustCapture(t, request)

	byKey := map[string]clonebundle.CaptureItem{}
	for _, item := range result.Capture.Items {
		byKey[item.NativeItemKey] = item
	}
	expectReason := map[string]string{
		"store/auth-token":   "credential_excluded",
		"store/session-key":  "machine_auth_excluded",
		"store/runtime-sock": "runtime_state_excluded",
		"store/lock-file":    "transient_lock_excluded",
	}
	for key, reason := range expectReason {
		item, ok := byKey[key]
		if !ok {
			t.Fatalf("excluded candidate %q has no item", key)
		}
		if item.Included() {
			t.Fatalf("excluded candidate %q sealed as included", key)
		}
		if item.ExclusionReason == nil || *item.ExclusionReason != reason {
			t.Fatalf("excluded candidate %q reason = %v, want %q", key, item.ExclusionReason, reason)
		}
		if item.BlobDescriptorID != nil || item.ByteCount != nil {
			t.Fatalf("excluded candidate %q carries content members", key)
		}
	}
	for _, entry := range result.Raw.Entries {
		if entry.NativeItemKey != "store/blob-a" {
			t.Fatalf("raw entry %q is not the included member", entry.NativeItemKey)
		}
	}

	secrets := []string{}
	for _, secret := range exclusionSecrets {
		secrets = append(secrets, secret)
	}
	scanSecrets := func(context string, haystacks ...string) {
		t.Helper()
		for _, haystack := range haystacks {
			for _, secret := range secrets {
				if strings.Contains(haystack, secret) {
					t.Fatalf("%s carries excluded bytes %q", context, secret)
				}
			}
		}
	}
	for _, blob := range listStoredBlobs(t, storeSink) {
		scanSecrets("stored blob", string(blob.bytes))
	}
	scanSecrets("raw manifest", string(result.RawManifest))
	scanSecrets("capture manifest", string(result.CaptureManifest))
	scanSecrets("workspace binding", string(result.WorkspaceBinding))
	scanSecrets("capture log", strings.Join(result.Logs, "\n"))

	// A refusal elsewhere in the run must not smuggle excluded
	// bytes into its error string either.
	writeStoreFile(t, storeRoot, "store/zz-extra", []byte("unplanned-bytes"))
	second, err := Capture(validCaptureRequest(t, storeRoot, openTestStore(t)))
	if err == nil {
		t.Fatalf("Capture() = %+v, want an unplanned-member refusal", second)
	}
	scanSecrets("refusal error", err.Error())
}

func TestCaptureLogRecordsDigestsOnly(t *testing.T) {
	storeRoot := t.TempDir()
	secret := "snap-secret-included-payload-5a1c"
	payload := "prefix-" + secret + "-suffix"
	writeStoreFile(t, storeRoot, "store/blob-a", []byte(payload))
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{{NativeKey: "store/blob-a", Class: "durable_payload", Required: true}}
	result := mustCapture(t, request)

	joined := strings.Join(result.Logs, "\n")
	if strings.Contains(joined, secret) {
		t.Fatalf("capture log carries payload bytes:\n%s", joined)
	}
	digest := result.Raw.Entries[0].BlobID.String()
	if !strings.Contains(joined, digest) {
		t.Fatalf("capture log misses the blob digest %q:\n%s", digest, joined)
	}
	if !strings.Contains(joined, "application/octet-stream") {
		t.Fatalf("capture log misses the media type:\n%s", joined)
	}
	if want := "size=" + itoa(len(payload)); !strings.Contains(joined, want) {
		t.Fatalf("capture log misses the byte size %q:\n%s", want, joined)
	}
}
