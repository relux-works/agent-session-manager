package config

import . "os"

// censusDotImportWitness is the executable witness for a dot-imported
// standard-library mutation used as a function value.
func censusDotImportWitness(path string, data []byte) error {
	return WriteFile(path, data, 0o600)
}
