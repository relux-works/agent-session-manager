package canonicaljson

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget(t *testing.T) {
	t.Parallel()

	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve validation test source path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	configBytes, err := os.ReadFile(filepath.Join(repositoryRoot, "task-board.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Spawn struct {
			WorktreeIsolation struct {
				Validation struct {
					Commands []string `json:"commands"`
				} `json:"validation"`
			} `json:"worktree_isolation"`
		} `json:"spawn"`
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatal(err)
	}
	configured := make(map[string]int, len(config.Spawn.WorktreeIsolation.Validation.Commands))
	for _, command := range config.Spawn.WorktreeIsolation.Validation.Commands {
		configured[command]++
	}

	err = filepath.WalkDir(repositoryRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != repositoryRoot && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor" || entry.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		packagePath, pathErr := filepath.Rel(repositoryRoot, filepath.Dir(path))
		if pathErr != nil {
			return pathErr
		}
		packageArgument := "."
		if packagePath != "." {
			packageArgument = "./" + filepath.ToSlash(packagePath)
		}
		for _, declaration := range parsed.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || !strings.HasPrefix(function.Name.Name, "Fuzz") {
				continue
			}
			expected := "go test " + packageArgument + " -run=^$ -fuzz=^" + function.Name.Name + "$ -fuzztime=100x -parallel=1"
			if count := configured[expected]; count != 1 {
				t.Errorf("configured validation contains %q %d times, want exactly once", expected, count)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
