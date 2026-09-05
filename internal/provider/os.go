package provider

import (
	"os"
	"path/filepath"
	"strings"
)

// OSSystem is the production System: it reads the host filesystem through
// the os package. It holds no state and starts no process.
type OSSystem struct{}

// ReadDir returns the entry names of dir in filesystem order. Discover
// sorts them, so the order here carries no determinism obligation.
func (OSSystem) ReadDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

// Canonicalize returns the absolute, symlink-resolved target of path.
func (OSSystem) Canonicalize(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

// ownerAttester attests the owning UID of a stat result. It is a variable
// only so seam-failure tests can force the attestation-failure path
// through the production Inspect below: no real stat result reaches that
// path on unix, and without the seam the error branch is untestable.
// Production code never reassigns it.
var ownerAttester = fileOwnerUID

// Inspect stats the symlink target at an already-canonical path.
func (OSSystem) Inspect(canonical string) (FileInfo, error) {
	info, err := os.Stat(canonical)
	if err != nil {
		return FileInfo{}, err
	}
	uid, err := ownerAttester(info)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{IsRegular: info.Mode().IsRegular(), UID: uid}, nil
}

// ReadFile returns the full bytes of the symlink target at an
// already-canonical path.
func (OSSystem) ReadFile(canonical string) ([]byte, error) {
	return os.ReadFile(canonical)
}

// PathDirs returns the host PATH directories in listed order, skipping
// empty entries.
func (OSSystem) PathDirs() []string {
	raw := os.Getenv("PATH")
	if raw == "" {
		return nil
	}
	var dirs []string
	for _, dir := range strings.Split(raw, string(os.PathListSeparator)) {
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// CurrentOperatorPolicy returns the OwnerPolicy for the invoking operator
// with no additional administrator-approved identities. A host that
// recognizes administrator-approved owners supplies them explicitly in
// OwnerPolicy instead.
func CurrentOperatorPolicy() OwnerPolicy {
	return OwnerPolicy{OperatorUID: uint32(os.Geteuid())}
}
