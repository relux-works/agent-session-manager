package resumesmoke

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// This file installs smoke records durably: content-addressed,
// no-replace, fsynced files the crash tests kill around. The
// discipline follows the session repository model — an exclusive
// create first, so append-only is a filesystem fact under a second
// process rather than a check-then-act race; identical bytes on an
// existing path are the idempotent replay, and disagreeing bytes
// refuse for quarantine instead of replacing evidence.
//
// The filename carries the record digest, so one record has exactly
// one path: a retry installs the same bytes at the same path, and a
// torn write at that path disagrees with the retry and refuses
// until the operator quarantines it. Only verified records install:
// invalid bytes refuse before any file is created.

// StoreHooks are the test-only crash points of the install path:
// BeforeCreate runs before the exclusive create, AfterWrite runs
// after the bytes land but before the fsync. Production passes nil.
type StoreHooks struct {
	BeforeCreate func(path string) error
	AfterWrite   func(path string) error
}

// Store verifies the record bytes and installs them at their
// content-addressed path under the directory, creating nothing
// else. It returns the installed path. An identical retry returns
// the same path; disagreeing bytes at that path refuse.
func Store(directory string, record []byte, hooks *StoreHooks) (string, error) {
	decoded, err := VerifyRecord(record)
	if err != nil {
		return "", err
	}
	name := "native-resume-smoke-" + strings.TrimPrefix(decoded.RecordID, "sha256:") + ".json"
	path := filepath.Join(directory, name)
	if hooks != nil && hooks.BeforeCreate != nil {
		if err := hooks.BeforeCreate(path); err != nil {
			return "", err
		}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if !os.IsExist(err) {
			return "", fmt.Errorf("create smoke record: %w", err)
		}
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("read installed smoke record: %w", readErr)
		}
		if !bytes.Equal(existing, record) {
			return "", fmt.Errorf("installed smoke record disagrees; quarantine %q before retry", path)
		}
		return path, nil
	}
	if _, err := file.Write(record); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write smoke record: %w", err)
	}
	if hooks != nil && hooks.AfterWrite != nil {
		if err := hooks.AfterWrite(path); err != nil {
			_ = file.Close()
			return "", err
		}
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("fsync smoke record: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close smoke record: %w", err)
	}
	if err := syncDirectory(directory); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads one installed record and verifies it: malformed,
// tampered, or inconsistent bytes refuse, never decode into a
// record the caller could mistake for evidence.
func Load(path string) (Record, []byte, error) {
	var empty Record
	bytes, err := os.ReadFile(path)
	if err != nil {
		return empty, nil, fmt.Errorf("read smoke record: %w", err)
	}
	record, err := VerifyRecord(bytes)
	if err != nil {
		return empty, nil, err
	}
	return record, bytes, nil
}

// syncDirectory fsyncs the directory so the installed name is
// durable alongside the file bytes.
func syncDirectory(directory string) error {
	dir, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open smoke record directory: %w", err)
	}
	defer func() { _ = dir.Close() }()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("fsync smoke record directory: %w", err)
	}
	return nil
}
