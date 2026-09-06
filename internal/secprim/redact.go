package secprim

import (
	"strings"
	"unicode/utf8"
)

// Redaction markers. The corpus and key-pattern arms share one marker so
// arm order cannot leak which rule fired; the private-key arm names its
// class because a redacted key block must still read as a key block to an
// operator deciding whether to re-enter it.
const (
	redactedMarker        = "[redacted]"
	redactedPrivateKey    = "[redacted private key block]"
	minSecretRunes        = 8
	privateKeyBeginPrefix = "-----BEGIN "
	privateKeyBeginSuffix = "PRIVATE KEY-----"
	privateKeyEndPrefix   = "-----END "
)

// sensitiveKeyNames is the reviewed exact-match list for the key=value arm
// below, derived from the Section 16.2 exclusion table (provider
// credentials, SSH/private identity, environment secrets, machine
// authentication). Matching is exact on the lowercased key, never by
// substring: refusing every key containing "token" would refuse
// "token_count" while passing a secret written under an innocuous name,
// which is pure false-positive surface with no true-positive capability.
//
// Deliberately absent: bare "key" (cache keys, sort keys, key=value
// identifiers are ordinary diagnostics), bare "auth" (auth status and
// authority digests are not secrets), and "pwd"/"PWD" (working-directory
// diagnostics are not secrets). Compound keys outside this list (for
// example "db_password") are not matched: the corpus arm covers secret
// values the caller knows, and no static list can close the naming
// space. That residue is stated, not hidden.
var sensitiveKeyNames = map[string]bool{
	"access_token": true, "api_key": true, "apikey": true,
	"auth_json": true, "authorization": true, "auth_token": true,
	"bearer_token": true, "client_secret": true, "cookie": true,
	"cookies": true, "credential": true, "credentials": true,
	"dotenv": true, "env_secret": true, "environment_secret": true,
	"oauth_token": true, "passphrase": true, "password": true,
	"passwd": true, "private_key": true, "refresh_token": true,
	"secret": true, "secrets": true, "session_token": true,
	"ssh_private_key": true, "subscription_token": true, "token": true,
}

// Redact scrubs free text before persistence or display (Section 16.4:
// plugin stderr, provider logs, and doctor output are redacted before
// persistence). It decides exactly three shapes, in this order:
//
//  1. private-key blocks: -----BEGIN ... PRIVATE KEY----- through the
//     matching -----END ...----- (or to end of input when unterminated,
//     failing closed) become "[redacted private key block]";
//  2. caller-known secret values: every secrets entry of at least 8 runes
//     is replaced verbatim with "[redacted]". Shorter values are skipped:
//     a two-character secret would redact every occurrence of a common
//     bigram, which is pure false-positive surface. ULID/UUID-shaped
//     non-secrets the caller lists are the caller's error and are
//     redacted all the same;
//  3. sensitive key=value pairs: key=, key:, key =, or "key": " shapes
//     whose key exactly matches sensitiveKeyNames (case-insensitively)
//     have their value replaced with "[redacted]", plus URL userinfo
//     (scheme://...@) whose credentials are never displayable.
//
// Section 16.2 is explicit that v0.5.0 does not claim reliable
// content-level secret scrubbing, and this function claims no more: a
// secret under an unlisted key, a bare credential with no key shape, or a
// key block without its header passes through, and only the corpus arm
// can catch those when the caller knows the value. The function is
// idempotent: its markers contain no sensitive key shape, so rescanning
// redacted text changes nothing.
func Redact(line string, secrets []string) string {
	redacted := redactPrivateKeyBlocks(line)
	redacted = redactCorpusValues(redacted, secrets)
	redacted = redactSensitivePairs(redacted)
	redacted = redactURLUserinfo(redacted)
	return redacted
}

// redactPrivateKeyBlocks replaces PEM private-key blocks. The header must
// contain PRIVATE KEY so certificates and public keys (which are routinely
// logged for pinning) keep passing through.
func redactPrivateKeyBlocks(line string) string {
	var rendered strings.Builder
	rendered.Grow(len(line))
	rest := line
	for {
		begin := strings.Index(rest, privateKeyBeginPrefix)
		if begin < 0 {
			rendered.WriteString(rest)
			return rendered.String()
		}
		// The header close is searched past the opening dashes: the
		// prefix itself starts with dashes, which are not the close.
		headerClose := strings.Index(rest[begin+len(privateKeyBeginPrefix):], "-----")
		if headerClose < 0 {
			rendered.WriteString(rest)
			return rendered.String()
		}
		header := rest[begin : begin+len(privateKeyBeginPrefix)+headerClose+len("-----")]
		if !strings.Contains(header, privateKeyBeginSuffix) {
			rendered.WriteString(rest[:begin+len(header)])
			rest = rest[begin+len(header):]
			continue
		}
		end := strings.Index(rest[begin+len(header):], privateKeyEndPrefix)
		if end < 0 {
			rendered.WriteString(rest[:begin])
			rendered.WriteString(redactedPrivateKey)
			return rendered.String()
		}
		// Same dashPrefix care for the END header close.
		afterEnd := rest[begin+len(header)+end+len(privateKeyEndPrefix):]
		closeDashes := strings.Index(afterEnd, "-----")
		if closeDashes < 0 {
			rendered.WriteString(rest[:begin])
			rendered.WriteString(redactedPrivateKey)
			return rendered.String()
		}
		rendered.WriteString(rest[:begin])
		rendered.WriteString(redactedPrivateKey)
		rest = afterEnd[closeDashes+len("-----"):]
	}
}

// redactCorpusValues replaces caller-known secret values verbatim.
// Entries below minSecretRunes are skipped (see Redact): matching them
// would redact ordinary text.
func redactCorpusValues(line string, secrets []string) string {
	for _, secret := range secrets {
		if utf8.RuneCountInString(secret) < minSecretRunes || secret == "" {
			continue
		}
		line = strings.ReplaceAll(line, secret, redactedMarker)
	}
	return line
}

// redactSensitivePairs masks values of exact sensitive keys in key=value,
// key:value, and "key": "value" shapes, including values spaced across
// chunks ("key": "value", "key = value"). The line is walked chunk by
// chunk (whitespace-delimited); each chunk is spliced head + masked value
// + tail so surrounding quotes, braces, and commas survive and JSON stays
// parseable.
func redactSensitivePairs(line string) string {
	chunks := mergeBareKeyChunks(strings.Split(line, " "))
	for index, chunk := range chunks {
		chunks[index] = maskSensitiveChunk(chunk)
	}
	// Second pass: a "key": shape whose value starts after this chunk.
	// The first pass left the dangling chunk alone because its value was
	// empty; mask the remainder instead.
	var out []string
	index := 0
	for index < len(chunks) {
		if _, ok := danglingSensitiveKey(chunks[index]); ok && index+1 < len(chunks) {
			rest := strings.Join(chunks[index+1:], " ")
			head, tail := maskDanglingValue(rest)
			out = append(out, chunks[index], head)
			if tail != "" {
				out = append(out, tail)
			}
			break
		}
		out = append(out, chunks[index])
		index++
	}
	return strings.Join(out, " ")
}

// mergeBareKeyChunks joins a bare sensitive-key chunk with a following
// separator-leading chunk ("key" "=" "value" becomes "key =" "value"),
// preserving the separating space, so spaced assignments reach the same
// first pass as compact ones.
func mergeBareKeyChunks(chunks []string) []string {
	var merged []string
	index := 0
	for index < len(chunks) {
		if index+1 < len(chunks) && isBareSensitiveKey(chunks[index]) && startsWithSep(chunks[index+1]) {
			merged = append(merged, chunks[index]+" "+chunks[index+1])
			index += 2
			continue
		}
		merged = append(merged, chunks[index])
		index++
	}
	return merged
}

// isBareSensitiveKey reports whether the chunk is exactly a sensitive key
// past surrounding quotes and braces.
func isBareSensitiveKey(chunk string) bool {
	core := strings.Trim(chunk, `"'{}`)
	if core == "" {
		return false
	}
	for index := 0; index < len(core); index++ {
		if !isKeyByte(core[index]) {
			return false
		}
	}
	return sensitiveKeyNames[strings.ToLower(core)]
}

// startsWithSep reports whether the chunk opens with '=' or ':' past
// leading spaces.
func startsWithSep(chunk string) bool {
	trimmed := strings.TrimLeft(chunk, " \t")
	return strings.HasPrefix(trimmed, "=") || strings.HasPrefix(trimmed, ":")
}

// danglingSensitiveKey reports whether the chunk is a sensitive key with
// an empty value (trailing ':' or '='): its value follows in the next
// chunk.
func danglingSensitiveKey(chunk string) (string, bool) {
	trimmed := strings.TrimRight(chunk, ",;")
	if !strings.HasSuffix(trimmed, ":") && !strings.HasSuffix(trimmed, "=") {
		return "", false
	}
	_, key := splitChunkHead(trimmed[:len(trimmed)-1])
	if !sensitiveKeyNames[strings.ToLower(key)] {
		return "", false
	}
	return key, true
}

// maskDanglingValue masks the value at the start of the remainder
// following a dangling key: past one optional opening quote to its closing
// quote (keeping the tail, so JSON stays parseable), or to the next comma,
// semicolon, or end of input when unquoted. It returns the masked head and
// the untouched tail.
func maskDanglingValue(rest string) (head, tail string) {
	blank := rest[:len(rest)-len(strings.TrimLeft(rest, " \t"))]
	val := strings.TrimLeft(rest, " \t")
	quote := ""
	if strings.HasPrefix(val, `"`) || strings.HasPrefix(val, `'`) {
		quote = val[:1]
		val = val[1:]
	}
	if strings.HasPrefix(val, redactedMarker) {
		return rest, ""
	}
	end := len(val)
	if quote != "" {
		if closing := strings.Index(val, quote); closing >= 0 {
			end = closing
		}
	} else {
		end = strings.IndexAny(val, ",;")
		if end < 0 {
			end = len(val)
		}
	}
	if end == 0 {
		return rest, ""
	}
	tail = val[end:]
	if quote != "" && strings.HasPrefix(tail, quote) {
		return blank + quote + redactedMarker + quote + tail[len(quote):], ""
	}
	if quote != "" {
		return blank + quote + redactedMarker, tail
	}
	if tail == "" {
		return blank + redactedMarker, ""
	}
	return blank + redactedMarker, tail
}

// maskSensitiveChunk masks the value of one chunk when its key exactly
// matches sensitiveKeyNames. The key must start at the chunk start (past
// opening braces and quotes) and be followed by '=' or ':'; the value runs
// to the matching closing quote when quoted, else to the next value
// terminator (comma, semicolon, closing brace or bracket) or chunk end.
func maskSensitiveChunk(chunk string) string {
	for _, sep := range []string{"=", ":"} {
		position := strings.Index(chunk, sep)
		if position < 0 {
			continue
		}
		head, key := splitChunkHead(chunk[:position])
		if key == "" || !sensitiveKeyNames[strings.ToLower(key)] {
			continue
		}
		rest := chunk[position+len(sep):]
		blank := rest[:len(rest)-len(strings.TrimLeft(rest, " \t"))]
		val := strings.TrimLeft(rest, " \t")
		quote := ""
		if strings.HasPrefix(val, `"`) || strings.HasPrefix(val, `'`) {
			quote = val[:1]
			val = val[1:]
		}
		// An already-redacted value stays untouched: without this guard
		// the bracket and brace terminators below would eat into the
		// marker itself and idempotence would fail.
		if strings.HasPrefix(val, redactedMarker) {
			return chunk
		}
		end := len(val)
		if quote != "" {
			if closing := strings.Index(val, quote); closing >= 0 {
				end = closing
			}
		} else {
			end = strings.IndexAny(val, ",;}]")
			if end < 0 {
				end = len(val)
			}
		}
		if end == 0 {
			continue
		}
		tail := val[end:]
		if quote != "" && strings.HasPrefix(tail, quote) {
			tail = tail[len(quote):]
			return head + key + chunk[position:position+len(sep)] + blank + quote + redactedMarker + quote + tail
		}
		return head + key + chunk[position:position+len(sep)] + blank + quote + redactedMarker + tail
	}
	return chunk
}

// splitChunkHead separates the key from its leading punctuation: opening
// braces and quotes belong to the head, the key is the trailing key-shaped
// run past one optional closing quote and trailing spaces.
func splitChunkHead(before string) (head, key string) {
	keyEnd := len(before)
	for keyEnd > 0 && (before[keyEnd-1] == ' ' || before[keyEnd-1] == '\t') {
		keyEnd--
	}
	if keyEnd > 0 && (before[keyEnd-1] == '"' || before[keyEnd-1] == '\'') {
		keyEnd--
	}
	keyStart := keyEnd
	for keyStart > 0 && isKeyByte(before[keyStart-1]) {
		keyStart--
	}
	if keyStart == keyEnd {
		return before, ""
	}
	return before[:keyStart], before[keyStart:keyEnd]
}

// isKeyByte reports whether the byte can appear in a key: letters,
// digits, underscore, dash, and dot. Key bytes are ASCII by construction,
// so byte-wise scanning is exact.
func isKeyByte(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		character == '_' || character == '-' || character == '.'
}

// redactURLUserinfo masks credentials in scheme://userinfo@host shapes:
// everything between "://" and the next "@" is never displayable.
func redactURLUserinfo(line string) string {
	var rendered strings.Builder
	rendered.Grow(len(line))
	rest := line
	for {
		scheme := strings.Index(rest, "://")
		if scheme < 0 {
			rendered.WriteString(rest)
			return rendered.String()
		}
		at := strings.Index(rest[scheme+3:], "@")
		if at < 0 {
			rendered.WriteString(rest)
			return rendered.String()
		}
		at += scheme + 3
		if strings.ContainsAny(rest[scheme+3:at], " \t\"'") {
			rendered.WriteString(rest[:scheme+3])
			rest = rest[scheme+3:]
			continue
		}
		rendered.WriteString(rest[:scheme+3])
		rendered.WriteString(redactedMarker)
		rest = rest[at:]
	}
}
