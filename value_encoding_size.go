package jet

import "unicode/utf8"

// jsonStringValueBound includes quotes and encoding/json's default HTML escaping.
// Invalid UTF-8 reserves six bytes per replacement, covering both escaped and
// literal replacement characters across supported Go versions.
// Callers must charge scanning work before using the result to reserve storage.
func jsonStringValueBound(s string) (int64, bool) {
	size := int64(2)
	for i := 0; i < len(s); {
		width, encoded := 1, int64(1)
		c := s[i]
		switch {
		case c == '\\', c == '"', c == '\b', c == '\f', c == '\n', c == '\r', c == '\t':
			encoded = 2
		case c < 0x20, c == '<', c == '>', c == '&':
			encoded = 6
		case c >= utf8.RuneSelf:
			var r rune
			r, width = utf8.DecodeRuneInString(s[i:])
			encoded = int64(width)
			if (r == utf8.RuneError && width == 1) || r == '\u2028' || r == '\u2029' {
				encoded = 6
			}
		}
		var ok bool
		size, ok = addValueSizes(size, encoded)
		if !ok {
			return 0, false
		}
		i += width
	}
	return size, true
}

func htmlValueSize(s string) (int64, bool) {
	size := int64(len(s))
	for i := 0; i < len(s); i++ {
		var extra int64
		switch s[i] {
		case '&', '\'', '"':
			extra = 4
		case '<', '>':
			extra = 3
		default:
			continue
		}
		var ok bool
		size, ok = addValueSizes(size, extra)
		if !ok {
			return 0, false
		}
	}
	return size, true
}

func queryValueSize(s string) (int64, bool) {
	size := int64(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' || c == ' ' {
			continue
		}
		var ok bool
		size, ok = addValueSizes(size, 2)
		if !ok {
			return 0, false
		}
	}
	return size, true
}
