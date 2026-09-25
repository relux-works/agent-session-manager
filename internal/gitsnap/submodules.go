package gitsnap

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

func (s *contentState) submodules(ctx context.Context, runner Runner, root string, links map[string]IndexEntry, head Head, remotes []Remote, prefix string, depth int) ([]Submodule, error) {
	result := []Submodule{}
	if len(links) == 0 {
		return result, nil
	}
	if depth >= 16 || s.count+len(links) > 256 {
		return nil, refuse(GateSubmoduleState, "submodule depth or total exceeds protocol bound")
	}
	s.count += len(links)
	// Read only the working .gitmodules mapping. --no-includes prevents local
	// include directives from turning this logical mapping into arbitrary config.
	file, err := secprim.OpenNoFollowFile(filepath.Join(root, ".gitmodules"))
	if err != nil {
		return nil, refuse(GateSubmoduleState, "submodule mapping is missing or unsafe")
	}
	file.Close()
	raw, fail := runOK(ctx, runner, root, "config", "--no-includes", "--file", ".gitmodules", "-z", "--get-regexp", `^submodule\..*\.(path|url)$`)
	if fail != nil {
		return nil, refuse(GateSubmoduleState, "submodule mapping read failed")
	}
	if len(raw) == 0 || raw[len(raw)-1] != 0 {
		return nil, refuse(GateSubmoduleState, "partial submodule mapping")
	}
	type mapping struct{ name, url string }
	byKey := map[string]*mapping{}
	for _, record := range strings.Split(string(raw[:len(raw)-1]), "\x00") {
		key, value, ok := strings.Cut(record, "\n")
		if !ok {
			return nil, refuse(GateSubmoduleState, "malformed submodule mapping")
		}
		dot := strings.LastIndexByte(key, '.')
		if dot < 0 {
			return nil, refuse(GateSubmoduleState, "malformed submodule mapping key")
		}
		section, field := key[:dot], key[dot+1:]
		m := byKey[section]
		if m == nil {
			m = &mapping{}
			byKey[section] = m
		}
		switch field {
		case "path":
			if m.name != "" {
				return nil, refuse(GateSubmoduleState, "duplicate submodule path mapping")
			}
			m.name = value
		case "url":
			if m.url != "" {
				return nil, refuse(GateSubmoduleState, "duplicate submodule URL mapping")
			}
			m.url = value
		default:
			return nil, refuse(GateSubmoduleState, "unexpected submodule mapping field")
		}
	}
	byPath := map[string]string{}
	for _, m := range byKey {
		if _, err := scalar.ParseRelativePath(m.name); err != nil {
			return nil, refuse(GateSubmoduleState, "invalid submodule path")
		}
		if _, ok := byPath[m.name]; ok {
			return nil, refuse(GateSubmoduleState, "ambiguous submodule path")
		}
		byPath[m.name] = m.url
	}
	names := make([]string, 0, len(links))
	for name := range links {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if s.excluded(root, name, prefix, os.ModeDir) != "" {
			return nil, refuse(GateCapturePolicy, "required submodule is excluded by capture policy")
		}
		url, ok := byPath[name]
		if !ok {
			return nil, refuse(GateSubmoduleState, "gitlink has no mapping")
		}
		url, err = resolveSubmoduleURL(ctx, runner, root, url, head, remotes)
		if err != nil {
			return nil, err
		}
		parsed, err := scalar.ParseSanitizedGitURL(url)
		if err != nil {
			return nil, refuse(GateSubmoduleState, "submodule URL is not portable and sanitized")
		}
		identity, err := deriveIdentity([]Remote{{Name: "origin", FetchURL: parsed.String()}})
		if err != nil {
			return nil, err
		}
		sub := Submodule{Path: name, RepositoryIdentity: identity, SanitizedURL: parsed.String(), GitlinkOID: links[name].OID}
		native := filepath.Join(root, filepath.FromSlash(name))
		// Reject symlinked parents before probing Git, which would otherwise follow
		// them into another checkout. The accepted index path grammar is lexical.
		cursor := root
		missing := false
		for _, component := range strings.Split(name, "/") {
			cursor = filepath.Join(cursor, component)
			info, err := os.Lstat(cursor)
			if errors.Is(err, os.ErrNotExist) {
				missing = true
				break
			}
			if err != nil {
				return nil, refuse(GateContentRead, "submodule path read failed")
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return nil, refuse(GateSubmoduleState, "unsafe submodule checkout path")
			}
		}
		if !missing {
			marker, err := os.Lstat(filepath.Join(native, ".git"))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, refuse(GateContentRead, "submodule initialization read failed")
			}
			if err == nil {
				if marker.Mode()&os.ModeSymlink != 0 || (!marker.IsDir() && !marker.Mode().IsRegular()) {
					return nil, refuse(GateSubmoduleState, "unsafe submodule Git marker")
				}
				child, err := capture(ctx, runner, native, s, path.Join(prefix, name), depth+1)
				if err != nil {
					return nil, err
				}
				if child.Head.Mode == "unborn" || child.Worktree.RepoRoot != native {
					return nil, refuse(GateSubmoduleState, "initialized submodule has no own committed checkout")
				}
				sub.Initialized, sub.State = true, child
			} else {
				entries, readErr := os.ReadDir(native)
				if readErr != nil {
					return nil, refuse(GateContentRead, "uninitialized submodule read failed")
				}
				if len(entries) != 0 {
					return nil, refuse(GateSubmoduleState, "uninitialized submodule contains unrepresented files")
				}
			}
		}
		result = append(result, sub)
	}
	return result, nil
}

// Git resolves ./ and ../ mappings as directories relative to the default
// remote repository, not its parent. This reads tracking configuration without
// running submodule init, which would mutate the user's repository config.
func resolveSubmoduleURL(ctx context.Context, runner Runner, root, mapping string, head Head, remotes []Remote) (string, error) {
	if !strings.HasPrefix(mapping, "./") && !strings.HasPrefix(mapping, "../") {
		return mapping, nil
	}
	remoteName := "origin"
	if head.Mode == "branch" {
		result, err := readOptionalConfig(ctx, runner, root, "config", "--get", "branch."+strings.TrimPrefix(*head.Ref, "refs/heads/")+".remote")
		if err != nil {
			return "", err
		}
		if result.ExitCode == 0 {
			remoteName = strings.TrimSuffix(string(result.Stdout), "\n")
		}
	}
	base := ""
	for _, remote := range remotes {
		if remote.Name == remoteName {
			base = remote.FetchURL
			break
		}
	}
	if base == "" {
		return "", refuse(GateSubmoduleState, "relative mapping has no portable default remote")
	}
	prefix, repoPath := "", ""
	if scheme := strings.Index(base, "://"); scheme >= 0 {
		slash := strings.Index(base[scheme+3:], "/")
		if slash < 0 {
			return "", refuse(GateSubmoduleState, "default remote has no repository path")
		}
		cut := scheme + 3 + slash
		prefix, repoPath = base[:cut+1], base[cut+1:]
	} else {
		host, remotePath, ok := strings.Cut(base, ":")
		if !ok {
			return "", refuse(GateSubmoduleState, "default remote is machine-local")
		}
		prefix, repoPath = host+":", remotePath
	}
	resolved := path.Join(repoPath, mapping)
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", refuse(GateSubmoduleState, "relative mapping escapes remote namespace")
	}
	return prefix + resolved, nil
}
