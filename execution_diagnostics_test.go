package jet

import (
	"context"
	"io"
	"testing"
)

func TestExecutionErrorFormattingBounded(t *testing.T) {
	const graph = `{{ x := "a" }}{{ range ints(0, 16) }}{{ x = slice(x, x) }}{{ end }}`
	for _, source := range []string{
		graph + `{{ missing }}`,
		graph + `{{ missing = x }}`,
		graph + `{{ map(x, "v") }}`,
		graph + `{{ ints(x, 3) }}`,
		`{{ range repeat("x", 16384) }}{{ end }}`,
	} {
		t.Run(source, func(t *testing.T) {
			err := runtimeTemplate(t, source).ExecuteContext(context.Background(), io.Discard, nil, nil, ExecutionOptions{MaxValueBytes: 32768, MaxOutputBytes: 1024, MaxSteps: 1000000, MaxDepth: 100})
			if err == nil {
				t.Fatal("invalid operation succeeded")
			}
			t.Logf("diagnostic_bytes=%d", len(err.Error()))
			if len(err.Error()) > 1024 {
				t.Fatalf("error recursively formatted runtime data: %d bytes", len(err.Error()))
			}
		})
	}
}

type diagnosticStringer struct{ called *bool }

func (v diagnosticStringer) String() string {
	*v.called = true
	return "unexpected formatting hook"
}

func TestExecutionDiagnosticsAvoidNativeHooks(t *testing.T) {
	called := false
	vars := VarMap{}.Set("v", diagnosticStringer{called: &called})
	for _, source := range []string{`{{ missing = v }}`, `{{ map(v, "x") }}`, `{{ ints(v, 3) }}`, `{{ range v }}{{ end }}`} {
		err := runtimeTemplate(t, source).ExecuteContext(context.Background(), io.Discard, vars, nil, ExecutionOptions{MaxSteps: 10000, MaxValueBytes: 10000, MaxDepth: 100})
		if err == nil || called {
			t.Fatalf("diagnostic invoked a native hook: called=%t", called)
		}
	}
}

func TestExecutionMissingIdentifierWithSharedInput(t *testing.T) {
	var value interface{} = "a"
	for i := 0; i < 16; i++ {
		value = []interface{}{value, value}
	}
	err := runtimeTemplate(t, `{{ missing }}`).ExecuteContext(context.Background(), io.Discard, VarMap{}.Set("v", value), nil, ExecutionOptions{MaxSteps: 10000, MaxValueBytes: 10000, MaxDepth: 100})
	if err == nil || len(err.Error()) > 1024 {
		t.Fatal("missing identifier diagnostic expanded shared input")
	}
}
