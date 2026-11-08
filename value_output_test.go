package jet

import (
	"bytes"
	"context"
	"testing"
)

func TestValueOutputCompatibility(t *testing.T) {
	tests := []struct {
		name, source, text, html string
	}{
		{"lower", `{{ lower("ÄBC") }}`, "äbc", "äbc"},
		{"upper", `{{ upper("ȿabc") }}`, "ⱾABC", "ⱾABC"},
		{"hasPrefix", `{{ hasPrefix("abc", "ab") }}`, "true", "true"},
		{"hasSuffix", `{{ hasSuffix("abc", "bc") }}`, "true", "true"},
		{"repeat", `{{ repeat("ab", 3) }}`, "ababab", "ababab"},
		{"repeatZero", `{{ repeat("ab", 0) }}`, "", ""},
		{"replace", `{{ replace("aba", "a", "xy", -1) }}`, "xybxy", "xybxy"},
		{"replaceEmpty", `{{ replace("äb", "", "x", -1) }}`, "xäxbx", "xäxbx"},
		{"replaceZero", `{{ replace("aba", "a", "xy", 0) }}`, "aba", "aba"},
		{"split", `{{ s := split("a::b::", "::") }}{{ len(s) }}:{{ s[1] }}`, "3:b", "3:b"},
		{"splitEmpty", `{{ s := split("äb", "") }}{{ len(s) }}:{{ s[0] }}`, "2:ä", "2:ä"},
		{"trimSpace", `{{ trimSpace("\t ä \n") }}`, "ä", "ä"},
		{"html", `{{ html("<&") }}`, "&lt;&amp;", "&amp;lt;&amp;amp;"},
		{"url", `{{ url("a b&ä") }}`, "a+b%26%C3%A4", "a+b%26%C3%A4"},
		{"safeHtml", `{{ safeHtml("<&") }}`, "&lt;&amp;", "&lt;&amp;"},
		{"safeJs", `{{ safeJs("<\n") }}`, `\u003C\u000A`, `\u003C\u000A`},
		{"raw", `{{ raw("<&") }}`, "<&", "<&"},
		{"unsafe", `{{ unsafe("<&") }}`, "<&", "<&"},
		{"json", `{{ json(map("a", "<")) }}`, `{"a":"\u003c"}`, `{&#34;a&#34;:&#34;\u003c&#34;}`},
		{"writeJson", `{{ writeJson(map("a", "<")) }}`, "{\"a\":\"\\u003c\"}\n", "{\"a\":\"\\u003c\"}\n"},
		{"map", `{{ m := map("a", 7) }}{{ m.a }}`, "7", "7"},
		{"slice", `{{ s := slice("a", 7) }}{{ s[0] }}:{{ s[1] }}`, "a:7", "a:7"},
		{"array", `{{ s := array("a", 7) }}{{ len(s) }}`, "2", "2"},
		{"alias", `{{ s := slice("a") }}{{ m := map("a", s, "b", s) }}{{ len(m.a) + len(m.b) }}`, "2", "2"},
		{"isset", `{{ isset(missing) }}:{{ isset(1, "a") }}`, "false:true", "false:true"},
		{"len", `{{ len("ä") }}`, "2", "2"},
		{"ints", `{{ range i, v := ints(-2, 1) }}{{ i }}:{{ v }};{{ end }}`, "0:-2;1:-1;2:0;", "0:-2;1:-1;2:0;"},
		{"dump", `{{ dump("missing") }}`, "", ""},
		{"includeIfExists", `{{ includeIfExists("payload") }}{{ includeIfExists("missing") }}ok`, "<&ok", "<&ok"},
		{"exec", `{{ exec("payload") }}ok`, "ok", "ok"},
		{"byteConcat", `{{ "a" + json("b") }}`, `a"b"`, `a&#34;b&#34;`},
		{"scalarConcat", `{{ "n=" + 7 }}`, "n=7", "n=7"},
		{"escapedOutput", `{{ "<&" }}`, "<&", "&lt;&amp;"},
		{"nestedTry", `{{ try }}a{{ try }}{{ "<&" }}{{ end }}b{{ end }}`, "a<&b", "a&lt;&amp;b"},
		{"rollback", `a{{ try }}hidden{{ missing }}{{ catch }}caught{{ end }}b`, "acaughtb", "acaughtb"},
		{"nestedRollback", `{{ try }}a{{ try }}hidden{{ missing }}{{ catch }}b{{ end }}c{{ end }}`, "abc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, mode := range []string{"text", "html"} {
				t.Run(mode, func(t *testing.T) {
					set := valueOutputSet(mode == "html")
					tpl, err := set.Parse("compatibility", tt.source)
					if err != nil {
						t.Fatal(err)
					}
					var out bytes.Buffer
					if err := tpl.Execute(&out, nil, nil); err != nil {
						t.Fatal(err)
					}
					want := tt.text
					if mode == "html" {
						want = tt.html
					}
					if out.String() != want {
						t.Fatalf("got %q, want %q", out.String(), want)
					}
					out.Reset()
					if err := tpl.ExecuteContext(context.Background(), &out, nil, nil, ExecutionOptions{MaxSteps: 1000000, MaxDepth: 100, MaxValueBytes: 1 << 20, MaxOutputBytes: 1 << 20}); err != nil {
						t.Fatal(err)
					}
					if out.String() != want {
						t.Fatalf("ExecuteContext got %q, want %q", out.String(), want)
					}
				})
			}
		})
	}
}

func valueOutputSet(html bool) *Set {
	loader := NewInMemLoader()
	loader.Set("payload", "<&")
	if html {
		return NewSet(loader)
	}
	return NewSet(loader, WithSafeWriter(nil))
}
