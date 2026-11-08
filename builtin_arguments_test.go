package jet

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"
)

func TestBuiltinInvalidArguments(t *testing.T) {
	for _, source := range []string{
		`{{ lower(map("k", 1)) }}`,
		`{{ repeat("x", slice(1)) }}`,
		`{{ split(slice(1), ",") }}`,
		`{{ slice(nil) }}`,
		`{{ array(nil) }}`,
		`{{ len(p) }}`,
	} {
		t.Run(source, func(t *testing.T) {
			var err error
			var recovered interface{}
			func() {
				defer func() { recovered = recover() }()
				err = runtimeTemplate(t, source).ExecuteContext(context.Background(), io.Discard, VarMap{}.Set("p", (*int)(nil)), nil, ExecutionOptions{MaxValueBytes: 10000, MaxSteps: 100000, MaxDepth: 100})
			}()
			if recovered != nil || err == nil {
				t.Fatalf("invalid argument escaped as %T; error=%v", recovered, err)
			}
		})
	}
}

func TestBuiltinMapConvertsValidatedKeyOnce(t *testing.T) {
	called := 0
	vars := VarMap{}.SetFunc("key", func(a Arguments) reflect.Value {
		called++
		return reflect.ValueOf([]byte("k"))
	})
	var out bytes.Buffer
	var err error
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		err = runtimeTemplate(t, `{{ m := map(key(), "v") }}{{ m.k }}`).ExecuteContext(context.Background(), &out, vars, nil, ExecutionOptions{MaxValueBytes: 10000, MaxSteps: 100000, MaxDepth: 100})
	}()
	if recovered != nil || err != nil || called != 1 || out.String() != "v" {
		t.Fatalf("got panic %T, error %v, key calls %d, output %q", recovered, err, called, out.String())
	}
}
