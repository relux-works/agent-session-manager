package secprim

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsEnvName(t *testing.T) {
	t.Parallel()
	admitted := []string{"_", "A", "a", "PATH", "AX_CONFIG", "a1", "_0", "HOME", strings.Repeat("x", 128)}
	for _, name := range admitted {
		if !IsEnvName(name) {
			t.Errorf("IsEnvName(%q) = false, want true", name)
		}
	}
	refused := []string{
		"", "9LIVES", "-X", "A B", "A=B", "A:B", "a/b", "a.b", "a-b",
		"ünï", "A\nB", "A\x00B", strings.Repeat("x", 129),
	}
	for _, name := range refused {
		if IsEnvName(name) {
			t.Errorf("IsEnvName(%q) = true, want false", name)
		}
	}
}

func lookupFixture(variables map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := variables[name]
		return value, ok
	}
}

func TestBuildEnv(t *testing.T) {
	t.Parallel()
	environment, err := BuildEnv(
		[]string{"PATH", "HOME"},
		map[string]string{"AX_WORKSPACE": "/stage/ws"},
		lookupFixture(map[string]string{"PATH": "/usr/bin:/bin", "HOME": "/home/ax", "SECRET": "nope"}),
	)
	if err != nil {
		t.Fatalf("BuildEnv = %v", err)
	}
	want := []string{"AX_WORKSPACE=/stage/ws", "HOME=/home/ax", "PATH=/usr/bin:/bin"}
	if !reflect.DeepEqual(environment, want) {
		t.Fatalf("BuildEnv = %q, want sorted %q", environment, want)
	}
}

func TestBuildEnvSkipsAbsentNames(t *testing.T) {
	t.Parallel()
	// An unset variable is a legitimate absence: the build skips it
	// instead of failing or substituting a default.
	environment, err := BuildEnv([]string{"MISSING"}, nil, lookupFixture(nil))
	if err != nil {
		t.Fatalf("BuildEnv = %v", err)
	}
	if len(environment) != 0 {
		t.Fatalf("BuildEnv = %q, want an empty environment", environment)
	}
}

func TestBuildEnvRefuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		allowed  []string
		literals map[string]string
		lookup   func(string) (string, bool)
		want     string
	}{
		{
			name: "bad allowed name", allowed: []string{"9X"},
			literals: nil, lookup: lookupFixture(nil),
			want: "secprim unsafe environment: env name: 9X",
		},
		{
			name: "duplicate allowed", allowed: []string{"A", "A"},
			literals: nil, lookup: lookupFixture(nil),
			want: "secprim unsafe environment: env allowlist duplicate: A",
		},
		{
			name: "bad literal key", allowed: nil,
			literals: map[string]string{"a-b": "x"}, lookup: lookupFixture(nil),
			want: "secprim unsafe environment: env name: a-b",
		},
		{
			name: "literal collides", allowed: []string{"FOO"},
			literals: map[string]string{"FOO": "x"}, lookup: lookupFixture(map[string]string{"FOO": "y"}),
			want: "secprim unsafe environment: env literal collision: FOO",
		},
		{
			name: "inherited NUL", allowed: []string{"FOO"},
			literals: nil, lookup: lookupFixture(map[string]string{"FOO": "a\x00b"}),
			want: "secprim unsafe environment: env value NUL: FOO",
		},
		{
			name: "literal NUL", allowed: nil,
			literals: map[string]string{"FOO": "a\x00b"}, lookup: lookupFixture(nil),
			want: "secprim unsafe environment: env value NUL: FOO",
		},
		{
			name: "literal encoding", allowed: nil,
			literals: map[string]string{"FOO": "a\xffb"}, lookup: lookupFixture(nil),
			want: "secprim unsafe environment: env value encoding: FOO",
		},
		{
			name: "inherited encoding", allowed: []string{"FOO"},
			literals: nil, lookup: lookupFixture(map[string]string{"FOO": "a\xffb"}),
			want: "secprim unsafe environment: env value encoding: FOO",
		},
		{
			name: "nil lookup", allowed: nil,
			literals: nil, lookup: nil,
			want: "secprim unsafe environment: env lookup missing: no parent environment reader",
		},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			t.Parallel()
			_, err := BuildEnv(kase.allowed, kase.literals, kase.lookup)
			requireRefusal(t, err, ErrUnsafeEnv, kase.want)
		})
	}
}

func TestBuildEnvAdmitsEmptyValues(t *testing.T) {
	t.Parallel()
	// FOO= is a meaningful empty assignment, not a missing one.
	environment, err := BuildEnv(nil, map[string]string{"FOO": ""}, lookupFixture(nil))
	if err != nil {
		t.Fatalf("BuildEnv = %v", err)
	}
	if !reflect.DeepEqual(environment, []string{"FOO="}) {
		t.Fatalf("BuildEnv = %q, want [FOO=]", environment)
	}
}
