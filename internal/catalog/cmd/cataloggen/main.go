package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/cataloggen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("cataloggen", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	metadataPath := flags.String("metadata", "", "reviewed catalog metadata JSON")
	contractsPath := flags.String("contracts", "", "verified normative contract lock")
	outputPath := flags.String("output", "", "generated Go output")
	check := flags.Bool("check", false, "verify generated Go output without rewriting it")
	adopted := flags.Bool("adopted", false, "derive metadata and contract lock from the adopted release")
	root := flags.String("root", ".", "repository root the adopted inputs are resolved against")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *adopted {
		return runAdopted(*root, *outputPath, *check, *metadataPath != "", *contractsPath != "")
	}
	if *metadataPath == "" || *contractsPath == "" || *outputPath == "" {
		return fmt.Errorf("-metadata, -contracts, and -output are required")
	}

	metadata, err := os.ReadFile(*metadataPath)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}
	contracts, err := os.ReadFile(*contractsPath)
	if err != nil {
		return fmt.Errorf("read contract lock: %w", err)
	}
	generated, err := cataloggen.Generate(metadata, contracts)
	if err != nil {
		return fmt.Errorf("generate catalog: %w", err)
	}
	if *check {
		if err := checkUnchanged(*outputPath, generated); err != nil {
			return fmt.Errorf("check catalog: %w", err)
		}
		return nil
	}
	if err := writeIfChanged(*outputPath, generated); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
}

// runAdopted derives the catalog metadata and normative lock inputs from the
// code-level adopted release and otherwise behaves exactly like the explicit
// invocation: same Generate call, same -check/-output semantics.
func runAdopted(root, output string, check, explicitMetadata, explicitContracts bool) error {
	if explicitMetadata || explicitContracts {
		return fmt.Errorf("-adopted cannot be combined with -metadata or -contracts")
	}
	if output == "" {
		return fmt.Errorf("-output is required with -adopted")
	}
	metadataPath := filepath.Join(root, "internal", "catalog", "catalog."+string(catalog.Adopted)+".json")
	contractsPath := filepath.Join(root, "internal", "specpin", string(catalog.Adopted)+".lock.json")

	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("read adopted metadata %q: %w", metadataPath, err)
	}
	contracts, err := os.ReadFile(contractsPath)
	if err != nil {
		return fmt.Errorf("read adopted contract lock %q: %w", contractsPath, err)
	}
	if err := checkAdoptedLockRelease(contracts); err != nil {
		return err
	}
	generated, err := cataloggen.Generate(metadata, contracts)
	if err != nil {
		return fmt.Errorf("generate catalog: %w", err)
	}
	if check {
		if err := checkUnchanged(output, generated); err != nil {
			return fmt.Errorf("check catalog: %w", err)
		}
		return nil
	}
	if err := writeIfChanged(output, generated); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
}

// adoptedLockRelease extracts the source release claim of a contract lock
// without verifying it; verification stays the job of Generate.
type adoptedLockRelease struct {
	Source struct {
		Release string `json:"release"`
	} `json:"source"`
}

// checkAdoptedLockRelease refuses a lock whose source release differs from
// the adopted release before generation. A lock that does not even decode
// falls through to Generate, which reports the verification failure.
func checkAdoptedLockRelease(contracts []byte) error {
	var lock adoptedLockRelease
	if err := json.Unmarshal(contracts, &lock); err != nil {
		return nil
	}
	if lock.Source.Release != string(catalog.Adopted) {
		return fmt.Errorf("adopted contract lock release %q differs from adopted release %q", lock.Source.Release, string(catalog.Adopted))
	}
	return nil
}

func checkUnchanged(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(existing, content) {
		return fmt.Errorf("generated catalog is stale; run go generate ./internal/catalog")
	}
	return nil
}

func writeIfChanged(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, content) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".catalog-gen-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
