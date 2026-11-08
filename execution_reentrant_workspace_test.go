package jet_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/CloudyKit/jet/v6"
)

type workspaceBorrowWriter struct {
	bytes.Buffer
	runtime   *jet.Runtime
	active    [][]byte
	snapshots [][]byte
	changed   bool
	checked   int
}

func (w *workspaceBorrowWriter) check() {
	for i, b := range w.active {
		w.checked++
		if !bytes.Equal(b, w.snapshots[i]) {
			w.changed = true
		}
	}
}
func (w *workspaceBorrowWriter) Write(b []byte) (int, error) {
	w.check()
	if w.runtime != nil && len(w.active) < 3 {
		w.active = append(w.active, b)
		w.snapshots = append(w.snapshots, append([]byte(nil), b...))
		w.runtime.YieldBlock("inner", nil)
		w.check()
		w.active = w.active[:len(w.active)-1]
		w.snapshots = w.snapshots[:len(w.snapshots)-1]
	}
	return w.Buffer.Write(b)
}

type workspaceNamedInt int64

func TestExecutionPrinterWorkspaceBorrow(t *testing.T) {
	for _, tt := range []struct {
		name         string
		outer, inner interface{}
	}{
		{"int", int(123), int(456)},
		{"uint", uint64(18446744073709551615), uint64(2)},
		{"bool", true, false},
		{"float", 1.25, 4.5},
		{"string", "outer", "inner"},
		{"named", workspaceNamedInt(123), workspaceNamedInt(456)},
		{"collection", []interface{}{123, true, uint64(99)}, []interface{}{456, false, uint64(1)}},
	} {
		for _, live := range []bool{false, true} {
			name := tt.name + "/background"
			ctx := context.Background()
			cancel := func() {}
			if live {
				name = tt.name + "/live"
				ctx, cancel = context.WithCancel(ctx)
			}
			t.Run(name, func(t *testing.T) {
				defer cancel()
				var out workspaceBorrowWriter
				vars := jet.VarMap{}.Set("outerValue", tt.outer).Set("nestedValue", tt.inner)
				vars.SetFunc("capture", func(a jet.Arguments) reflect.Value { out.runtime = a.Runtime(); return reflect.Value{} })
				tpl, err := jet.NewSet(jet.NewInMemLoader(), jet.WithSafeWriter(nil)).Parse("workspace.jet", `{{ block inner() }}{{ nestedValue }}{{ end }}{{ capture() }}{{ outerValue }}`)
				if err != nil {
					t.Fatal(err)
				}
				err = tpl.ExecuteContext(ctx, &out, vars, nil, jet.ExecutionOptions{MaxSteps: 1000000, MaxDepth: 64, MaxValueBytes: 8 << 20, MaxOutputBytes: 1 << 20})
				if err != nil || out.changed || out.checked == 0 {
					t.Fatalf("borrow stability: changed=%t checks=%d err=%v", out.changed, out.checked, err)
				}
			})
		}
	}
}
