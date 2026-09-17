// Selector grammar and configured source validation for the v0.6.0
// session selector contract (SPEC Section 14.7.1, refining Section 2.3).
//
// A selector is one CLI argument. The split is at the FIRST literal "@"
// only: the key is a session NAME, a bare UUID, or "id:SESSION_UUID", and
// the source is empty (bare Section 2.3 tiers), "local",
// "peer:ALIAS" (the entire remaining suffix, literal), or
// "id:HOST_UUID". The parser never joins arguments, decodes percent
// escapes, normalizes Unicode, trims, or case-folds aliases: a percent
// sign, surrounding spaces, or a second "@" inside a peer alias are
// literal characters of that alias.
//
// Configuration (local host identity, peer alias/host mappings, peer
// allowlist) arrives from the owning configuration layer together with
// the learned index handles, exactly as Reader documents. The selector
// validates the effective configuration it is given — including exact
// alias and host-ID uniqueness — because a caller-supplied mapping is
// not independently attested here. Duplicate source mappings are
// invalid configuration, never first-match selection.
package sessquery

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// MaxSelectorRunes bounds one selector argument: 64 NAME characters plus
// "@" plus "peer:" plus a 64-character alias, counted in runes so a
// Unicode alias keeps its configured 64-character allowance.
const MaxSelectorRunes = 134

// MaxAliasRunes is the existing configured peer-name bound (1-64
// printable characters) this grammar reuses without adding an alias
// encoding, normalization, or registration path.
const MaxAliasRunes = 64

// Selector source kinds. SourceBare retains the exact Section 2.3
// tiers; every other kind selects in exactly one source index with no
// fallback to any other source.
type SelectorSource int

const (
	// SourceBare resolves local exact name, allowlisted peer exact
	// name, exact UUID, then not found.
	SourceBare SelectorSource = iota
	// SourceLocal selects only in the local derived source index.
	SourceLocal
	// SourcePeerAlias selects only in the configured peer source
	// whose entire remaining suffix is the literal alias.
	SourcePeerAlias
	// SourcePeerHost selects only in the source whose canonical host
	// UUID is given; the local UUID selects local, otherwise exactly
	// one allowlisted configured peer.
	SourcePeerHost
)

// Selector is one parsed selector argument: the literal raw text, the
// key before the first "@", and the source after it.
type Selector struct {
	// Raw is the original single argument, preserved verbatim for plan
	// binding: another spelling that resolves to the same session never
	// authorizes the same plan.
	Raw string
	// Key is the literal text before the first "@", or the whole
	// argument when it carries no "@".
	Key string
	// IDKey reports the "id:" key form, which bypasses names and never
	// denotes a provider-native ID, directory instance ID, lease ID,
	// or UUID prefix.
	IDKey bool
	// UUIDShaped reports a key that parses as a canonical UUIDv7. A
	// bare UUID-shaped key still tries exact names first (Section 2.3
	// name-first precedence); a qualified UUID-shaped NAME tries the
	// exact name in its source first, then the UUID.
	UUIDShaped bool
	// Source is the selector source kind.
	Source SelectorSource
	// SourceValue is the literal peer alias (SourcePeerAlias) or the
	// canonical host UUID (SourcePeerHost), and empty otherwise.
	SourceValue string
}

// ParseSelector applies the literal grammar to one selector argument.
// A percent sign in an alias is literal. "peer:" with an empty alias,
// unknown source prefixes, malformed durable IDs, an empty key, and an
// overlong argument are invalid arguments, never silent names.
func ParseSelector(argument string) (Selector, error) {
	if utf8.RuneCountInString(argument) > MaxSelectorRunes {
		return Selector{}, fmt.Errorf("%w: selector exceeds %d characters", ErrInvalidArgument, MaxSelectorRunes)
	}
	key, source, qualified := strings.Cut(argument, "@")
	if key == "" {
		return Selector{}, fmt.Errorf("%w: selector carries an empty key", ErrInvalidArgument)
	}
	parsed := Selector{Raw: argument, Key: key}
	if id, ok := strings.CutPrefix(key, "id:"); ok {
		if _, err := scalar.ParseUUIDv7(id); err != nil {
			return Selector{}, fmt.Errorf("%w: malformed id: session UUID: %v", ErrInvalidArgument, err)
		}
		parsed.IDKey = true
		parsed.UUIDShaped = true
	} else if _, err := scalar.ParseUUIDv7(key); err == nil {
		parsed.UUIDShaped = true
	}
	if !qualified {
		parsed.Source = SourceBare
		return parsed, nil
	}
	switch {
	case source == "local":
		parsed.Source = SourceLocal
	case strings.HasPrefix(source, "peer:"):
		alias := strings.TrimPrefix(source, "peer:")
		if err := checkAlias(alias); err != nil {
			return Selector{}, fmt.Errorf("%w: malformed peer alias: %v", ErrInvalidArgument, err)
		}
		parsed.Source = SourcePeerAlias
		parsed.SourceValue = alias
	case strings.HasPrefix(source, "id:"):
		host := strings.TrimPrefix(source, "id:")
		if _, err := scalar.ParseUUIDv7(host); err != nil {
			return Selector{}, fmt.Errorf("%w: malformed id: host UUID: %v", ErrInvalidArgument, err)
		}
		parsed.Source = SourcePeerHost
		parsed.SourceValue = host
	default:
		return Selector{}, fmt.Errorf("%w: unknown selector source %q", ErrInvalidArgument, source)
	}
	return parsed, nil
}

// checkAlias enforces the existing configured peer-name bound without
// normalizing: valid UTF-8, 1-64 printable non-control runes, compared
// exactly everywhere it is used.
func checkAlias(alias string) error {
	if !utf8.ValidString(alias) {
		return fmt.Errorf("alias is not valid UTF-8")
	}
	count := utf8.RuneCountInString(alias)
	if count < 1 || count > MaxAliasRunes {
		return fmt.Errorf("alias carries %d characters, want 1-%d", count, MaxAliasRunes)
	}
	for _, character := range alias {
		if unicode.IsControl(character) || !unicode.IsPrint(character) {
			return fmt.Errorf("alias carries a non-printable character")
		}
	}
	return nil
}

// validatedConfig is the effective source configuration every selector
// entry validates before reading any index: the local host, the
// allowlist as a set, and the alias mapping restricted to known peer
// hosts. Indexes and mappings stay separate inputs; this type only
// records that the mapping shapes and uniqueness hold.
type validatedConfig struct {
	localHost string
	allowed   map[string]struct{}
	aliasOf   map[string]string
}

// validatedConfiguration validates the effective configuration the
// caller supplied: local host shape, peer host shapes and alias
// shapes, exact uniqueness of host-ID and alias mappings (including
// against the local host), and allowlist ID shapes. Duplicate source
// mappings are invalid configuration, never first-match selection. An
// allowlisted ID without a learned index stays absence in union tiers;
// it is validated for shape here and resolved at read time.
func (reader *Reader) validatedConfiguration() (validatedConfig, error) {
	config := validatedConfig{allowed: map[string]struct{}{}, aliasOf: map[string]string{}}
	if reader == nil {
		return config, fmt.Errorf("%w: reader carries no configuration", ErrInvalidConfig)
	}
	if reader.LocalHostID != "" {
		if _, err := scalar.ParseUUIDv7(reader.LocalHostID); err != nil {
			return config, fmt.Errorf("%w: malformed local host ID: %v", ErrInvalidConfig, err)
		}
		config.localHost = reader.LocalHostID
	}
	// Duplicate learned-index observations for one peer host keep their
	// existing read-time duplicate-index refusal (peerCandidates owns
	// it); configuration validation polices the mapping plane only, so
	// a peer host colliding with the local host is the duplicate
	// source mapping refused here.
	seenAliases := map[string]struct{}{}
	for _, peer := range reader.Peers {
		if _, err := scalar.ParseUUIDv7(peer.HostID); err != nil {
			return config, fmt.Errorf("%w: malformed peer host ID: %v", ErrInvalidConfig, err)
		}
		if peer.HostID == config.localHost && config.localHost != "" {
			return config, fmt.Errorf("%w: duplicate source mapping for host %s", ErrInvalidConfig, peer.HostID)
		}
	}
	for host, alias := range reader.Aliases {
		if _, err := scalar.ParseUUIDv7(host); err != nil {
			return config, fmt.Errorf("%w: malformed alias host ID: %v", ErrInvalidConfig, err)
		}
		if err := checkAlias(alias); err != nil {
			return config, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
		if _, duplicate := seenAliases[alias]; duplicate {
			return config, fmt.Errorf("%w: duplicate source mapping for alias %q", ErrInvalidConfig, alias)
		}
		seenAliases[alias] = struct{}{}
		config.aliasOf[host] = alias
	}
	for _, id := range reader.AllowlistedPeerIDs {
		if _, err := scalar.ParseUUIDv7(id); err != nil {
			return config, fmt.Errorf("%w: malformed allowlisted peer ID: %v", ErrInvalidConfig, err)
		}
		config.allowed[id] = struct{}{}
	}
	return config, nil
}

// aliasFor reports the exact configured alias for a known peer host,
// or empty when the source carries no alias mapping.
func (config validatedConfig) aliasFor(host string) string {
	return config.aliasOf[host]
}

// hostForAlias resolves an exact literal alias to its single host. The
// comparison is exact: folding, trimming, or decoding never applies.
func (config validatedConfig) hostForAlias(reader *Reader, alias string) (string, bool) {
	for _, peer := range reader.Peers {
		if config.aliasOf[peer.HostID] == alias && alias != "" {
			return peer.HostID, true
		}
	}
	return "", false
}

// sortedPeerHosts returns the known peer hosts in bytewise order so
// configuration digests and tie breaks stay deterministic.
func (config validatedConfig) sortedPeerHosts(reader *Reader) []string {
	hosts := make([]string, 0, len(reader.Peers))
	for _, peer := range reader.Peers {
		hosts = append(hosts, peer.HostID)
	}
	sort.Strings(hosts)
	return hosts
}
