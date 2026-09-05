//go:build !windows

package provider

import (
	"fmt"
	"os"
	"syscall"
)

// fileOwnerUID attests the owning UID of a stat result. The host refuses
// trust when attestation is unavailable rather than treating an unknown
// owner as approved.
func fileOwnerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("provider owner attestation: ownership metadata unavailable")
	}
	return stat.Uid, nil
}
