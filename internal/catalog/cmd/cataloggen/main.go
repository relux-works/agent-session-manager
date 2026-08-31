package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	if err := flags.Parse(arguments); err != nil {
		return err
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
	if err := writeIfChanged(*outputPath, generated); err != nil {
		return fmt.Errorf("write catalog: %w", err)
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
