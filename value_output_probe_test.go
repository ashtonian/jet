package jet

import (
	"bytes"
	"errors"
	"runtime"
	"testing"
)

type valueOutputFixture struct {
	name, source, output string
}

// These finite inputs also serve as the baseline for budgeted execution.
var valueOutputFixtures = []valueOutputFixture{
	{"doubling", `{{ s := "ab" }}{{ range ints(0, 16) }}{{ s = s + s }}{{ end }}{{ len(s) }}`, "131072"},
	{"repeat", `{{ s := repeat("ab", 65536) }}{{ len(s) }}`, "131072"},
	{"replace", `{{ s := replace("aaaaaaaaaaaaaaaa", "a", repeat("b", 8192), -1) }}{{ len(s) }}`, "131072"},
	{"split", `{{ s := split(repeat("a", 4096), "") }}{{ len(s) }}`, "4096"},
	{"nestedTry", `{{ try }}{{ try }}{{ repeat("x", 16384) }}{{ end }}{{ end }}`, ""},
	{"rollback", `{{ range ints(0, 4) }}{{ try }}{{ repeat("x", 16384) }}{{ missing }}{{ catch }}{{ end }}{{ end }}ok`, "ok"},
	{"discard", `{{ exec("generated") }}ok`, "ok"},
}

func TestValueOutputAmplification(t *testing.T) {
	for _, fixture := range valueOutputFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			loader := NewInMemLoader()
			loader.Set("generated", `{{ repeat("x", 16384) }}`)
			set := NewSet(loader, WithSafeWriter(nil))
			tpl, err := set.Parse("amplification", fixture.source)
			if err != nil {
				t.Fatal(err)
			}
			out := &valueOutputProbeWriter{limit: 1024}
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err = tpl.Execute(out, nil, nil)
			runtime.ReadMemStats(&after)
			t.Logf("output=%q accepted=%d attempted=%d writes=%d error=%v sink_limit=%t allocated_bytes=%d allocations=%d", out.String(), out.Len(), out.attempted, out.writes, err, errors.Is(err, errValueOutputProbeSink), after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs)
			if fixture.name == "nestedTry" {
				// A small final sink cannot prevent temporary transaction buffers.
				if out.Len() != 0 || out.attempted == 0 {
					t.Fatal("transaction did not reach the refusing sink")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != fixture.output {
				t.Fatalf("got %q, want %q", out.String(), fixture.output)
			}
		})
	}
}

var errValueOutputProbeSink = errors.New("value/output probe sink limit")

type valueOutputProbeWriter struct {
	bytes.Buffer
	limit, attempted, writes int
}

func (w *valueOutputProbeWriter) Write(p []byte) (int, error) {
	w.writes++
	w.attempted += len(p)
	if len(p) > w.limit-w.Len() {
		return 0, errValueOutputProbeSink
	}
	return w.Buffer.Write(p)
}
