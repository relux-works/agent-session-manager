package config

import censusOS "os"

// censusAliasImportWitness executes the alias-import shape used by the
// write-path plants against a real file, so the static result has a live
// behavioral witness rather than a token-only source assertion.
func censusAliasImportWitness(path string, data []byte) error {
	return censusOS.WriteFile(path, data, 0o600)
}
