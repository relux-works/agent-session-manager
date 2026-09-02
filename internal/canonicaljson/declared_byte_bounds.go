package canonicaljson

import (
	"bytes"
	"encoding/json"
)

// CanonicalByteLength reports how many bytes value occupies once encoded in its
// RFC 8785 canonical form. It is the one measurement every declared byte bound
// in this implementation is taken through, here and in the configuration
// package, so that two call sites cannot disagree about what a byte is.
//
// Every declared byte bound in the specification bounds canonical bytes.
// Measuring len(json.Marshal(value)) instead measures Go's HTML-escaped
// encoding, which is not the canonical form: encoding/json rewrites each of the
// single bytes <, > and & into a six-character \u00XX escape, so a document
// built from those characters measures up to six times its canonical size.
// U+2028 and U+2029 inflate from three canonical bytes to six the same way.
// Under an escaped measurement the bound actually in force is a property
// of encoding/json rather than the pinned limit, and a conforming document is
// refused for being too large.
//
// Disabling HTML escaping alone is not the fix, because encoding/json escapes
// U+2028 and U+2029 regardless of that setting. The canonical transform is what
// makes the measurement independent of the encoder; SetEscapeHTML(false) only
// keeps the intermediate encoding from inflating before it is re-parsed.
func CanonicalByteLength(value any) (int, error) {
	canonical, err := canonicalEncoding(value)
	if err != nil {
		return 0, err
	}
	return len(canonical), nil
}

// canonicalEncoding encodes value and returns its RFC 8785 canonical bytes.
func canonicalEncoding(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return Canonicalize(bytes.TrimSuffix(buffer.Bytes(), []byte("\n")))
}

// canonicalByteBound refuses value when its canonical encoding is larger than
// maximum bytes, and names the subject in the refusal. Both declared byte
// bounds in this package go through it, so neither can drift into measuring the
// escaped encoding on its own.
func canonicalByteBound(name string, value any, maximum int) error {
	measured, err := CanonicalByteLength(value)
	if err != nil {
		return invalidIdentity("measure canonical %s: %v", name, err)
	}
	if measured > maximum {
		return invalidIdentity("canonical %s is %d bytes, maximum is %d", name, measured, maximum)
	}
	return nil
}
