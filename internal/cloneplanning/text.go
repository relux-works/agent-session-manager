package cloneplanning

import (
	"unicode/utf8"
)

// This file carries the single text-validity gate every exported
// entry routes its consumed text through: SPEC.v0.7.0.md:256
// ("text MUST be valid UTF-8"). encoding/json silently rewrites
// invalid UTF-8 as U+FFFD and succeeds, so any entry that encodes
// must refuse before the encoding step; entries that decide only
// refuse with the same gate before their vocabulary gates. Text
// the entry ignores (non-fact payload members in ClassifyItem)
// never reaches a decision or an output, so no entry ever returns
// a value whose text differs from its input.

// validText reports whether the field is valid UTF-8. Every
// exported entry with a text-carrying parameter applies it to the
// text it consumes; the census test pins that entry set and the
// refusal table pins the refusal at every entry.
func validText(field string) bool {
	return utf8.ValidString(field)
}
