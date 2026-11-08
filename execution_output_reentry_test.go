package jet

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"text/template"
)

func TestOutputReentrantEscaperTerminal(t *testing.T) {
	for _, metered := range []bool{false, true} {
		var active *Runtime
		var out bytes.Buffer
		cause := &locatedSinkCause{}
		failed := &locatedFailWriter{cause: cause}
		entered, caught := false, false
		afterN, afterErr := -1, error(nil)
		escaper := func(w io.Writer, b []byte) {
			if active == nil || entered {
				template.HTMLEscape(w, b)
				return
			}
			entered = true
			parent := active.Writer
			func() {
				defer func() { caught = recover() != nil; active.Writer = parent }()
				active.Writer = failed
				active.YieldBlock("inner", nil)
			}()
			afterN, afterErr = w.Write([]byte("after"))
		}
		tpl, err := NewSet(NewInMemLoader(), WithSafeWriter(escaper)).Parse("terminal.jet", `{{ block inner() }}{{ value }}{{ end }}{{ saved := capture() }}{{ value }}`)
		if err != nil {
			t.Fatal(err)
		}
		vars := VarMap{}.Set("value", "<").SetFunc("capture", func(a Arguments) reflect.Value { active = a.Runtime(); return reflect.Value{} })
		if metered {
			err = tpl.ExecuteContext(context.Background(), &out, vars, nil, ExecutionOptions{MaxSteps: 1000000, MaxDepth: 64, MaxValueBytes: 8 << 20, MaxOutputBytes: 1 << 20})
		} else {
			err = tpl.Execute(&out, vars, nil)
		}
		if !entered || !caught || afterN != 0 || !errors.Is(afterErr, cause) || !errors.Is(err, cause) || out.String() != "&lt;" || failed.calls != 1 {
			t.Fatalf("metered=%t entered=%t caught=%t after=(%d,%v) output=%q calls=%d err=%v", metered, entered, caught, afterN, afterErr, out.String(), failed.calls, err)
		}
	}
}
