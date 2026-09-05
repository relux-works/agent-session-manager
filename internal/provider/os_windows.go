//go:build windows

package provider

import (
	"fmt"
	"os"
)

// fileOwnerUID refuses owner attestation on native Windows: no UID model
// exists there, and an unapproved owner must fail closed. External
// executables are therefore undiscoverable on native Windows until a
// Windows owner model lands; built-in adapters are unaffected because they
// carry no executable trust facts.
func fileOwnerUID(info os.FileInfo) (uint32, error) {
	return 0, fmt.Errorf("provider owner attestation: native Windows owner model is not implemented")
}
