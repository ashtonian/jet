package jet_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/CloudyKit/jet/v6"
)

type scalarBorrowWriter struct {
	bytes.Buffer
	runtime  *jet.Runtime
	borrowed []byte
	checked  bool
	changed  string
}

func (w *scalarBorrowWriter) Write(b []byte) (int, error) {
	if w.borrowed != nil {
		w.checked = true
		if string(w.borrowed) != "123" {
			w.changed = string(w.borrowed)
		}
	} else if w.runtime != nil {
		w.borrowed = b
		w.runtime.YieldBlock("inner", nil)
		w.borrowed = nil
	}
	return w.Buffer.Write(b)
}

func TestExecutionWriterBorrowDuringReentry(t *testing.T) {
	for _, live := range []bool{false, true} {
		name := "background"
		ctx := context.Background()
		cancel := func() {}
		if live {
			name = "live"
			ctx, cancel = context.WithCancel(ctx)
		}
		t.Run(name, func(t *testing.T) {
			defer cancel()
			var out scalarBorrowWriter
			vars := jet.VarMap{}.SetFunc("capture", func(a jet.Arguments) reflect.Value {
				out.runtime = a.Runtime()
				return reflect.Value{}
			})
			vars.Set("outerValue", int(123)).Set("nestedValue", int(456))
			set := jet.NewSet(jet.NewInMemLoader(), jet.WithSafeWriter(nil))
			tpl, err := set.Parse("borrowed.jet", "{{ block inner() }}{{ nestedValue }}{{ end }}{{ saved := capture() }}{{ outerValue }}")
			if err != nil {
				t.Fatal(err)
			}
			err = tpl.ExecuteContext(ctx, &out, vars, nil, jet.ExecutionOptions{
				MaxSteps: 1000000, MaxDepth: 64, MaxValueBytes: 8 << 20, MaxOutputBytes: 1 << 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !out.checked || out.changed != "" {
				t.Fatalf("nested writer checked=%t, borrowed chunk changed to %q", out.checked, out.changed)
			}
		})
	}
}
