package scalar

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var scpLikeGitURLPattern = regexp.MustCompile(`^(?:[A-Za-z0-9._-]+@)?[A-Za-z0-9.-]+:[^/].+$`)

type GitOID struct {
	objectFormat string
	value        string
}

func ParseGitOID(value string) (GitOID, error) {
	switch {
	case strings.HasPrefix(value, "sha1:") && len(value) == len("sha1:")+40:
		if !allLowerHex(value[len("sha1:"):]) {
			break
		}
		return GitOID{objectFormat: "sha1", value: value}, nil
	case strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+64:
		if !allLowerHex(value[len("sha256:"):]) {
			break
		}
		return GitOID{objectFormat: "sha256", value: value}, nil
	}
	return GitOID{}, invalid("git-oid", "must use sha1 plus 40 or sha256 plus 64 lowercase hexadecimal digits")
}

func ParseGitOIDForObjectFormat(value, objectFormat string) (GitOID, error) {
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return GitOID{}, invalid("git object format", "must be sha1 or sha256")
	}
	oid, err := ParseGitOID(value)
	if err != nil {
		return GitOID{}, err
	}
	if oid.objectFormat != objectFormat {
		return GitOID{}, invalid("git-oid", "prefix must match the containing repository object format")
	}
	return oid, nil
}

func allLowerHex(value string) bool {
	for index := range value {
		if !isLowerHex(value[index]) {
			return false
		}
	}
	return true
}

func (value GitOID) String() string       { return value.value }
func (value GitOID) ObjectFormat() string { return value.objectFormat }

func (value GitOID) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "git-oid", func(candidate string) error {
		_, err := ParseGitOIDForObjectFormat(candidate, value.objectFormat)
		return err
	})
}

func (value *GitOID) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "git-oid")
	if err != nil {
		return err
	}
	parsed, err := ParseGitOID(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

type GitRef struct{ value string }

func ParseGitRef(value string) (GitRef, error) {
	if value == "HEAD" || !strings.HasPrefix(value, "refs/") || len(value) > 1024 || !utf8.ValidString(value) {
		return GitRef{}, invalid("git-ref", "must be a 1 to 1024 byte fully qualified refs/ name")
	}
	if strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") ||
		strings.Contains(value, "..") || strings.Contains(value, "@{") || strings.HasSuffix(value, ".") || value == "@" {
		return GitRef{}, invalid("git-ref", "does not satisfy git check-ref-format grammar")
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".lock") {
			return GitRef{}, invalid("git-ref", "does not satisfy git check-ref-format grammar")
		}
	}
	for _, character := range value {
		if character <= 0x20 || character == 0x7f || strings.ContainsRune("~^:?*[\\", character) {
			return GitRef{}, invalid("git-ref", "does not satisfy git check-ref-format grammar")
		}
	}
	return GitRef{value: value}, nil
}

func (value GitRef) String() string { return value.value }

func (value GitRef) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "git-ref", func(candidate string) error {
		_, err := ParseGitRef(candidate)
		return err
	})
}

func (value *GitRef) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "git-ref")
	if err != nil {
		return err
	}
	parsed, err := ParseGitRef(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

type SanitizedGitURL struct{ value string }

func ParseSanitizedGitURL(value string) (SanitizedGitURL, error) {
	if value == "" || !utf8.ValidString(value) || strings.ContainsAny(value, "\r\n\t?#") {
		return SanitizedGitURL{}, invalid("sanitized-git-URL", "must be a non-empty valid UTF-8 URL")
	}
	if len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' {
		return SanitizedGitURL{}, invalid("sanitized-git-URL", "must not contain a machine-local drive path")
	}
	if scpLikeGitURLPattern.MatchString(value) && !strings.Contains(value, "://") {
		return SanitizedGitURL{value: value}, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return SanitizedGitURL{}, invalid("sanitized-git-URL", "must be an absolute remote URL without query or fragment")
	}
	switch parsed.Scheme {
	case "https", "git":
		if parsed.User != nil {
			return SanitizedGitURL{}, invalid("sanitized-git-URL", "must not contain userinfo credentials")
		}
	case "ssh":
		if parsed.User != nil {
			if _, password := parsed.User.Password(); password || parsed.User.Username() == "" {
				return SanitizedGitURL{}, invalid("sanitized-git-URL", "must not contain a password or token")
			}
		}
	default:
		return SanitizedGitURL{}, invalid("sanitized-git-URL", "scheme must be https, ssh, git, or provider-neutral git syntax")
	}
	return SanitizedGitURL{value: value}, nil
}

func (value SanitizedGitURL) String() string { return value.value }

func (value SanitizedGitURL) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "sanitized-git-URL", func(candidate string) error {
		_, err := ParseSanitizedGitURL(candidate)
		return err
	})
}

func (value *SanitizedGitURL) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "sanitized-git-URL")
	if err != nil {
		return err
	}
	parsed, err := ParseSanitizedGitURL(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

var _ json.Marshaler = GitOID{}
var _ json.Unmarshaler = (*GitOID)(nil)
var _ json.Marshaler = GitRef{}
var _ json.Unmarshaler = (*GitRef)(nil)
var _ json.Marshaler = SanitizedGitURL{}
var _ json.Unmarshaler = (*SanitizedGitURL)(nil)
