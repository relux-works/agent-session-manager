package sessquery

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestRev15AliasAwareRecordConsumptionCensus keeps the sealed checkpoint
// inventory semantic rather than lexical. Go aliases are the same type as
// their right-hand side for production purposes, including when aliases are
// chained or nested in a value type. Every compiling alternate producer must
// therefore be rejected by the same package-wide census that protects the
// canonical checkpointSeal and checkpointSealToken objects.
func TestRev15AliasAwareRecordConsumptionCensus(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	packageDir := filepath.Dir(file)
	if err := rev12CheckRecordConsumptionSource(packageDir); err != nil {
		t.Fatalf("real production census baseline rejected: %v", err)
	}

	plants := []struct {
		name string
		src  string
	}{
		{
			name: "alias_seal",
			src: `package sessquery

type review15Alias = checkpointSeal

func review15AliasMint() any { return new(review15Alias) }
`,
		},
		{
			name: "alias_token",
			src: `package sessquery

type review15TokenAlias = checkpointSealToken

func review15AliasMint() any { return new(review15TokenAlias) }
`,
		},
		{
			name: "chained_alias",
			src: `package sessquery

type review15Alias = checkpointSeal
type review15ChainedAlias = review15Alias

func review15ChainedAliasMint() any { return new(review15ChainedAlias) }
`,
		},
		{
			name: "container_of_alias",
			src: `package sessquery

type review15SealAlias = checkpointSeal
type review15TokenAlias = checkpointSealToken
type review15ContainerAlias = map[string]struct {
	seals  []*review15SealAlias
	tokens []review15TokenAlias
}

func review15ContainerAliasValue() any {
	return review15ContainerAlias(nil)
}
`,
		},
		{
			name: "function_signature_alias",
			src: `package sessquery

type review15SealAlias = checkpointSeal
type review15SignatureAlias = func() *review15SealAlias

var review15SignatureValue review15SignatureAlias
`,
		},
	}

	for _, plant := range plants {
		plant := plant
		t.Run(plant.name, func(t *testing.T) {
			dir := t.TempDir()
			rev12CopyProductionSources(t, packageDir, dir)
			if err := os.WriteFile(filepath.Join(dir, "review15_plant.go"), []byte(plant.src), 0o600); err != nil {
				t.Fatal(err)
			}
			err := rev12CheckRecordConsumptionSource(dir)
			if err == nil {
				t.Fatalf("record-consumption census admitted %s", plant.name)
			}
			if strings.Contains(err.Error(), "type-check") {
				t.Fatalf("%s did not reach the semantic census: %v", plant.name, err)
			}
			t.Logf("rejected %s: %v", plant.name, err)
		})
	}
}
