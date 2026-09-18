package clonesnap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// This file builds the closed WorkspaceBinding from real repository
// state: the logical workspace, the cwd normalized beneath the
// Workspace Group root, sorted unique repository remote
// fingerprints, the branch, and the head, index, and working-tree
// digests. The cwd never grants filesystem authority: absolute
// paths, dot segments, and escapes past the root refuse, and the
// recorded relative path must name a directory that exists. The
// binding seals through the landed DecodeWorkspaceBinding, so what
// this entry emits is exactly what the closed shape admits.

// WorkspaceRequest is one workspace checkpoint candidate: the
// Workspace Group root and the observed repository facts. Cwd is an
// absolute path or a root-relative path; RemoteFingerprints must
// arrive sorted unique; the branch and digests are nullable.
type WorkspaceRequest struct {
	LogicalWorkspaceID string
	WorkspaceRoot      string
	Cwd                string
	RemoteFingerprints []string
	Branch             *string
	HeadDigest         *string
	IndexDigest        *string
	WorkingTreeDigest  *string
	Platform           scalar.Platform
}

// CheckpointWorkspace builds one closed WorkspaceBinding from the
// observed repository state and returns its canonical bytes. The
// workspace root must be an existing directory opened without
// following a trailing symlink; the cwd must normalize beneath it;
// every digest and identifier validates through the landed decoder.
func CheckpointWorkspace(request WorkspaceRequest) ([]byte, error) {
	workspaceID, err := scalar.ParseUUIDv7(request.LogicalWorkspaceID)
	if err != nil {
		return nil, invalid("workspace binding logical_workspace_id is not a UUIDv7: %v", err)
	}
	root, err := secprim.OpenNoFollowDir(request.WorkspaceRoot)
	if err != nil {
		return nil, invalid("workspace root %q: %v", request.WorkspaceRoot, err)
	}
	_ = root.Close()
	relative, err := workspaceCwdRelative(request.Platform, request.WorkspaceRoot, request.Cwd)
	if err != nil {
		return nil, err
	}
	fingerprints, err := workspaceFingerprints(request.RemoteFingerprints)
	if err != nil {
		return nil, err
	}
	branch, err := workspaceNullableText(request.Branch, "branch", 1, 1024)
	if err != nil {
		return nil, err
	}
	head, err := workspaceNullableDigest(request.HeadDigest, "head_digest")
	if err != nil {
		return nil, err
	}
	index, err := workspaceNullableDigest(request.IndexDigest, "index_digest")
	if err != nil {
		return nil, err
	}
	tree, err := workspaceNullableDigest(request.WorkingTreeDigest, "working_tree_digest")
	if err != nil {
		return nil, err
	}
	object := map[string]any{
		"logical_workspace_id":           workspaceID.String(),
		"cwd_relative":                   relative,
		"repository_remote_fingerprints": fingerprints,
		"branch":                         branch,
		"head_digest":                    head,
		"index_digest":                   index,
		"working_tree_digest":            tree,
		"extensions":                     map[string]any{},
	}
	plain, err := json.Marshal(object)
	if err != nil {
		return nil, invalid("serialize workspace binding: %v", err)
	}
	sealed, err := canonicaljson.Canonicalize(plain)
	if err != nil {
		return nil, invalid("canonicalize workspace binding: %v", err)
	}
	if _, err := clonebundle.DecodeWorkspaceBinding(sealed); err != nil {
		return nil, invalid("sealed workspace binding refused: %v", err)
	}
	return sealed, nil
}

// workspaceCwdRelative normalizes the observed cwd beneath the
// Workspace Group root. An absolute cwd must sit inside the root; a
// relative cwd must already be normalized; absolute paths outside
// the root, parent segments, and dot segments refuse. The resolved
// directory must exist, so the checkpoint records observed state,
// not an unanchored claim.
func workspaceCwdRelative(platform scalar.Platform, root, cwd string) (string, error) {
	if cwd == "" {
		return "", invalid("workspace binding cwd_relative is empty")
	}
	if _, err := scalar.ParseAbsolutePath(platform, cwd); err == nil {
		relative, err := filepath.Rel(root, cwd)
		if err != nil {
			return "", invalid("workspace binding cwd %q is not beneath the workspace root", cwd)
		}
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", invalid("workspace binding cwd %q escapes the workspace root", cwd)
		}
		cwd = filepath.ToSlash(relative)
	}
	if cwd == "." {
		cwd = "."
	} else {
		parsed, err := scalar.ParseRelativePath(cwd)
		if err != nil {
			return "", invalid("workspace binding cwd_relative %q is not normalized beneath the workspace root: %v", cwd, err)
		}
		cwd = parsed.String()
	}
	absolute := root
	if cwd != "." {
		absolute = filepath.Join(root, filepath.FromSlash(cwd))
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", invalid("workspace binding cwd_relative %q does not name an observed directory: %v", cwd, err)
	}
	if !info.IsDir() {
		return "", invalid("workspace binding cwd_relative %q does not name an observed directory", cwd)
	}
	return cwd, nil
}

// workspaceFingerprints admits the sorted unique repository remote
// fingerprints: at most 128 digests in strict bytewise order. An
// unsorted, duplicated, over-long, or over-count row refuses; the
// entry never repairs caller order.
func workspaceFingerprints(fingerprints []string) ([]any, error) {
	if len(fingerprints) > 128 {
		return nil, invalid("workspace binding carries %d repository remote fingerprints, maximum is 128", len(fingerprints))
	}
	previous := ""
	values := make([]any, 0, len(fingerprints))
	for index, fingerprint := range fingerprints {
		digest, err := scalar.ParseDigest(fingerprint)
		if err != nil {
			return nil, invalid("workspace binding repository_remote_fingerprints[%d] is not a digest: %v", index, err)
		}
		if index > 0 && fingerprint <= previous {
			return nil, invalid("workspace binding carries unsorted or duplicate repository remote fingerprints")
		}
		previous = fingerprint
		values = append(values, digest.String())
	}
	return values, nil
}

func workspaceNullableText(value *string, name string, minimum, maximum int) (any, error) {
	if value == nil {
		return nil, nil
	}
	// The Section 1.6 string measure counts characters, not bytes:
	// the bound delegates to the environ owner, never len.
	if length := environ.StringLength(*value); length < minimum || length > maximum {
		return nil, invalid("workspace binding %s is not a string[%d..%d]", name, minimum, maximum)
	}
	return *value, nil
}

func workspaceNullableDigest(value *string, name string) (any, error) {

	if value == nil {
		return nil, nil
	}
	digest, err := scalar.ParseDigest(*value)
	if err != nil {
		return nil, invalid("workspace binding %s is not a digest: %v", name, err)
	}
	return digest.String(), nil
}
