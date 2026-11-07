package jet

import (
	"encoding/json"
	"html"
	"net/url"
	"testing"
	"unicode/utf8"
)

func TestEncodedValueSizes(t *testing.T) {
	for _, input := range []string{
		"", "abc-_.~012 XYZ", "<&>\"'", "\x00\x01\b\f\n\r\t\\/",
		"äȿⱾ世😀", "\u2028\u2029", "\xff\xfe\xc0\x80", "\xef\xbf\xbd",
		"a\xe2\x80b", "\xe2\x80", "\xf4\x90\x80\x80",
	} {
		t.Run(input, func(t *testing.T) {
			encodedJSON, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			for _, tt := range []struct {
				name string
				size func(string) (int64, bool)
				want int
			}{
				{"json", jsonStringValueBound, len(encodedJSON)},
				{"html", htmlValueSize, len(html.EscapeString(input))},
				{"query", queryValueSize, len(url.QueryEscape(input))},
			} {
				t.Run(tt.name, func(t *testing.T) {
					got, ok := tt.size(input)
					if tt.name == "json" && !utf8.ValidString(input) {
						if !ok || got < int64(tt.want) || got > 2+6*int64(len(input)) {
							t.Fatalf("invalid UTF-8 bound (%d, %t) does not cover %d bytes", got, ok, tt.want)
						}
						return
					}
					if !ok || got != int64(tt.want) {
						t.Fatalf("got (%d, %t), want (%d, true)", got, ok, tt.want)
					}
				})
			}
		})
	}
}
