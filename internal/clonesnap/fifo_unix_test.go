//go:build darwin || linux

package clonesnap

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// A FIFO member refuses before any payload byte is read: the Guard
// opens it non-blocking and refuses its special shape with the
// member named. The test completes without a writer, which is the
// proof the open never stalled.

func TestCaptureRefusesFIFO(t *testing.T) {
	storeRoot := t.TempDir()
	fifo := filepath.Join(storeRoot, "stash-fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "stash-fifo", Class: "durable_payload", Required: true},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, "stash-fifo")
}

func TestCaptureRefusesOptionalFIFO(t *testing.T) {
	// An OPTIONAL FIFO member still refuses with the member
	// named: presence routes every special to its Guard open,
	// required or not.
	storeRoot := t.TempDir()
	fifo := filepath.Join(storeRoot, "stash-fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	storeSink := openTestStore(t)
	request := validCaptureRequest(t, storeRoot, storeSink)
	request.Plan = []PlanItem{
		{NativeKey: "stash-fifo", Class: "durable_payload", Required: false},
	}
	result, err := Capture(request)
	if err == nil {
		t.Fatalf("Capture() = %+v, want a refusal", result)
	}
	requireRefusalDetail(t, err, "stash-fifo")
}
