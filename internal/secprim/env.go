package secprim

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// IsEnvName reports whether name satisfies the environment-name grammar
// shared by launch allowlists and SpawnPlan admission:
// [A-Za-z_][A-Za-z0-9_]{0,127}. internal/provhost delegates its wire-plan
// name check to this function, so the two packages cannot drift on what a
// name is; the byte-counted plan bounds around it stay in provhost.
func IsEnvName(name string) bool {
	if len(name) < 1 || len(name) > 128 {
		return false
	}
	for index := 0; index < len(name); index++ {
		char := name[index]
		valid := char == '_' ||
			char >= 'A' && char <= 'Z' ||
			char >= 'a' && char <= 'z' ||
			index > 0 && char >= '0' && char <= '9'
		if !valid {
			return false
		}
	}
	return true
}

// BuildEnv builds the exact environment of a child process from a minimal
// allowlist plus operator literals (Section 16.7 "minimal environment
// allowlists"):
//
//   - allowed names the inherited variables the child may see. Each must
//     satisfy IsEnvName; duplicates are refused rather than silently
//     collapsed, because a duplicated allowlist entry is a configuration
//     error. A name absent from lookup is skipped, not failed: an unset
//     variable is a legitimate absence, while a failed read must never
//     masquerade as one, so lookup reports only what it read.
//   - literals carries explicit key=value pairs (workspace-derived values,
//     SpawnPlan literals). Keys must satisfy IsEnvName and must be disjoint
//     from allowed: a literal that shadows an inherited name would make the
//     child's environment depend on map order rather than on policy.
//   - values must be NUL-free valid UTF-8. Empty values are admitted:
//     FOO= is a meaningful empty assignment, not a missing one.
//
// The result is sorted by name and rendered k=v, so identical inputs build
// byte-identical environments. lookup abstracts the parent environment for
// tests; production passes os.LookupEnv.
//
// No length bound is enforced here: the Section 5.1 literal bounds apply
// to the SpawnPlan wire object in internal/provhost. No default allowlist
// is declared here either: with no provider implementation in this
// repository to derive runtime needs from, a baked-in list would be an
// invented product policy, so the operator supplies the list explicitly
// and ExecRunner inherits the parent environment until one is supplied.
func BuildEnv(allowed []string, literals map[string]string, lookup func(string) (string, bool)) ([]string, error) {
	if lookup == nil {
		return nil, failEnv("env lookup missing", "no parent environment reader")
	}
	seen := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		if !IsEnvName(name) {
			return nil, failEnv("env name", name)
		}
		if seen[name] {
			return nil, failEnv("env allowlist duplicate", name)
		}
		seen[name] = true
	}
	for key := range literals {
		if !IsEnvName(key) {
			return nil, failEnv("env name", key)
		}
		if seen[key] {
			return nil, failEnv("env literal collision", key)
		}
	}
	values := make(map[string]string, len(allowed)+len(literals))
	for _, name := range allowed {
		value, present := lookup(name)
		if !present {
			continue
		}
		if strings.ContainsRune(value, 0) {
			return nil, failEnv("env value NUL", name)
		}
		if !utf8.ValidString(value) {
			return nil, failEnv("env value encoding", name)
		}
		values[name] = value
	}
	for key, value := range literals {
		if strings.ContainsRune(value, 0) {
			return nil, failEnv("env value NUL", key)
		}
		if !utf8.ValidString(value) {
			return nil, failEnv("env value encoding", key)
		}
		values[key] = value
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	environment := make([]string, 0, len(names))
	for _, name := range names {
		environment = append(environment, name+"="+values[name])
	}
	return environment, nil
}
