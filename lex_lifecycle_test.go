package jet

import (
	"strings"
	"testing"
)

func TestLexerStopsAtParseError(t *testing.T) {
	source := `{{ if }}` + strings.Repeat("unvisited", 1024)
	set := NewSet(NewInMemLoader())
	scanner := lex("error.jet", source, true)
	template := &Template{Name: "error.jet", ParseName: "error.jet", set: set}
	var err error
	func() {
		defer template.recover(&err)
		template.startParse(scanner)
		template.parseTemplate(false, nil)
	}()
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if scanner.pos >= Pos(len(source)) {
		t.Fatal("scanner consumed source beyond the syntax error")
	}
	if template.lex != nil {
		t.Fatal("parser retained scanner after error")
	}
}

func TestLexerPendingItems(t *testing.T) {
	for _, source := range []string{"plain text", "{{ ) }}", "{* comment *}tail", "{{ `raw` }}tail", "{{"} {
		t.Run(source, func(t *testing.T) {
			scanner := lex("tokens.jet", source, true)
			for {
				token := scanner.nextItem()
				if len(scanner.items) > 2 {
					t.Fatal("scanner retained more than one state's output")
				}
				if token.typ == itemEOF || token.typ == itemError {
					break
				}
			}
		})
	}
}
