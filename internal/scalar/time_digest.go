package scalar

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

var timestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}[Tt][0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3,}([Zz]|\+00:00)$`)

type Timestamp struct{ value string }

func ParseTimestamp(value string) (Timestamp, error) {
	if !timestampPattern.MatchString(value) {
		return Timestamp{}, invalid("timestamp", "must be UTC RFC3339 with at least 3 fractional digits")
	}
	instant, err := parseTimestampInstant(value)
	if err != nil {
		return Timestamp{}, invalid("timestamp", "must identify a real calendar instant")
	}
	_, offset := instant.Zone()
	if offset != 0 {
		return Timestamp{}, invalid("timestamp", "must use UTC")
	}
	return Timestamp{value: value}, nil
}

func (value Timestamp) String() string { return value.value }

func (value Timestamp) MarshalText() ([]byte, error) {
	return marshalValidatedText(value.value, func(candidate string) error {
		_, err := ParseTimestamp(candidate)
		return err
	})
}

func (value *Timestamp) UnmarshalText(data []byte) error {
	parsed, err := ParseTimestamp(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value Timestamp) Time() (time.Time, error) {
	parsed, err := ParseTimestamp(value.value)
	if err != nil {
		return time.Time{}, err
	}
	instant, err := parseTimestampInstant(parsed.value)
	if err != nil {
		return time.Time{}, invalid("timestamp", "must identify a real calendar instant")
	}
	return instant.UTC(), nil
}

func parseTimestampInstant(value string) (time.Time, error) {
	normalized := value
	if value[10] == 't' {
		normalized = value[:10] + "T" + value[11:]
	}
	if strings.HasSuffix(normalized, "z") {
		normalized = normalized[:len(normalized)-1] + "Z"
	}

	leapSecond := normalized[17:19] == "60"
	if leapSecond {
		if normalized[11:16] != "23:59" || !isPublishedUTCLeapSecond(normalized[:10]) {
			return time.Time{}, invalid("timestamp", "second 60 is valid only at a published UTC leap second")
		}
		// time.Time has no leap-second representation. Parse the preceding
		// representable second, then advance by one second; Timestamp retains the
		// original RFC3339 text for lossless wire round trips.
		normalized = normalized[:17] + "59" + normalized[19:]
	}

	instant, err := time.Parse(time.RFC3339Nano, normalized)
	if err != nil {
		return time.Time{}, err
	}
	if leapSecond {
		instant = instant.Add(time.Second)
	}
	return instant, nil
}

// isPublishedUTCLeapSecond is the immutable positive-leap-second history in
// effect for the pinned AX v0.5.0 source. Future announced leap seconds require
// an explicit implementation and conformance-fixture update; an arbitrary
// 23:59:60 must never become valid merely because it is lexically RFC3339.
func isPublishedUTCLeapSecond(date string) bool {
	switch date {
	case "1972-06-30", "1972-12-31", "1973-12-31", "1974-12-31",
		"1975-12-31", "1976-12-31", "1977-12-31", "1978-12-31",
		"1979-12-31", "1981-06-30", "1982-06-30", "1983-06-30",
		"1985-06-30", "1987-12-31", "1989-12-31", "1990-12-31",
		"1992-06-30", "1993-06-30", "1994-06-30", "1995-12-31",
		"1997-06-30", "1998-12-31", "2005-12-31", "2008-12-31",
		"2012-06-30", "2015-06-30", "2016-12-31":
		return true
	default:
		return false
	}
}

func (value Timestamp) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "timestamp", func(candidate string) error {
		_, err := ParseTimestamp(candidate)
		return err
	})
}

func (value *Timestamp) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "timestamp")
	if err != nil {
		return err
	}
	parsed, err := ParseTimestamp(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

type Digest struct{ value string }

func ParseDigest(value string) (Digest, error) {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return Digest{}, invalid("digest", "must use the sha256 prefix and exactly 64 digits")
	}
	for index := len("sha256:"); index < len(value); index++ {
		if !isLowerHex(value[index]) {
			return Digest{}, invalid("digest", "must contain lowercase hexadecimal digits")
		}
	}
	return Digest{value: value}, nil
}

func SHA256Digest(value []byte) Digest {
	sum := sha256.Sum256(value)
	return Digest{value: "sha256:" + hex.EncodeToString(sum[:])}
}

func (value Digest) String() string { return value.value }
func (value Digest) Hex() string    { return strings.TrimPrefix(value.value, "sha256:") }

func (value Digest) MarshalText() ([]byte, error) {
	return marshalValidatedText(value.value, func(candidate string) error {
		_, err := ParseDigest(candidate)
		return err
	})
}

func (value *Digest) UnmarshalText(data []byte) error {
	parsed, err := ParseDigest(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value Digest) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "digest", func(candidate string) error {
		_, err := ParseDigest(candidate)
		return err
	})
}

func (value *Digest) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "digest")
	if err != nil {
		return err
	}
	parsed, err := ParseDigest(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

var _ json.Marshaler = Timestamp{}
var _ json.Unmarshaler = (*Timestamp)(nil)
var _ json.Marshaler = Digest{}
var _ json.Unmarshaler = (*Digest)(nil)
