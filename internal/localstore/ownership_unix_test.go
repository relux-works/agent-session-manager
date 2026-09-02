//go:build !windows

package localstore

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestOwnerVerifierAcceptsExactCurrentOwnerAndRefusesForeignOrUnavailableIdentity(t *testing.T) {
	t.Parallel()

	currentUID := uint32(os.Geteuid())
	foreignUID := currentUID + 1
	if foreignUID == currentUID {
		foreignUID--
	}

	tests := []struct {
		name    string
		mode    os.FileMode
		want    os.FileMode
		system  any
		wantErr error
	}{
		{name: "exact current owner file", mode: 0o600, want: 0o600, system: &syscall.Stat_t{Uid: currentUID}},
		{name: "exact current owner directory", mode: os.ModeDir | 0o700, want: 0o700, system: &syscall.Stat_t{Uid: currentUID}},
		{name: "foreign owner", mode: 0o600, want: 0o600, system: &syscall.Stat_t{Uid: foreignUID}, wantErr: ErrUnsafeOwnership},
		{name: "ownership unavailable", mode: 0o600, want: 0o600, system: struct{}{}, wantErr: ErrUnsafeOwnership},
		{name: "group-readable file", mode: 0o640, want: 0o600, system: &syscall.Stat_t{Uid: currentUID}, wantErr: ErrUnsafeOwnership},
		{name: "group-accessible directory", mode: os.ModeDir | 0o750, want: 0o700, system: &syscall.Stat_t{Uid: currentUID}, wantErr: ErrUnsafeOwnership},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			info := syntheticFileInfo{mode: test.mode, system: test.system}
			if err := verifyOwnerFileInfo(info, test.want); !errors.Is(err, test.wantErr) {
				t.Fatalf("verifyOwnerFileInfo() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

type syntheticFileInfo struct {
	mode   os.FileMode
	system any
}

func (info syntheticFileInfo) Name() string       { return "fixture" }
func (info syntheticFileInfo) Size() int64        { return 0 }
func (info syntheticFileInfo) Mode() os.FileMode  { return info.mode }
func (info syntheticFileInfo) ModTime() time.Time { return time.Time{} }
func (info syntheticFileInfo) IsDir() bool        { return info.mode.IsDir() }
func (info syntheticFileInfo) Sys() any           { return info.system }
