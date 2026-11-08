package jet

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestBuiltinValueBudgets(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		vars         VarMap
		limit        int64
	}{
		{"repeat", valueOutputFixtures[1].source, nil, 32768},
		{"replace", valueOutputFixtures[2].source, nil, 32768},
		{"split", valueOutputFixtures[3].source, nil, 32768},
		{"upper", `{{ s := upper(v) }}{{ len(s) }}`, VarMap{}.Set("v", strings.Repeat("ȿ", 8192)), 4096},
		{"lower", `{{ s := lower(v) }}{{ len(s) }}`, VarMap{}.Set("v", strings.Repeat("ABC", 8192)), 4096},
		{"html", `{{ s := html(v) }}{{ len(s) }}`, VarMap{}.Set("v", strings.Repeat("&", 8192)), 8192},
		{"url", `{{ s := url(v) }}{{ len(s) }}`, VarMap{}.Set("v", strings.Repeat("&", 8192)), 8192},
		{"json", `{{ s := json(v) }}{{ len(s) }}`, VarMap{}.Set("v", map[string]interface{}{"s": strings.Repeat("x", 8192)}), 4096},
		{"writeJson", `{{ writeJson(v) }}`, VarMap{}.Set("v", []interface{}{strings.Repeat("x", 8192)}), 4096},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := runtimeTemplate(t, tt.source).ExecuteContext(context.Background(), &out, tt.vars, nil, ExecutionOptions{MaxValueBytes: tt.limit, MaxOutputBytes: 1 << 20, MaxSteps: 1000000, MaxDepth: 100})
			if !errors.Is(err, ErrValueLimit) || out.Len() != 0 {
				t.Fatalf("got %v, %q", err, out.String())
			}
		})
	}
}

func TestBuiltinSharedChildren(t *testing.T) {
	for _, source := range []string{
		`{{ x := "a" }}{{ range ints(0, 16) }}{{ x = slice(x, x) }}{{ end }}ok`,
		`{{ x := "a" }}{{ range ints(0, 8) }}{{ x = map("a", x, "b", x) }}{{ end }}ok`,
	} {
		t.Run(source, func(t *testing.T) {
			var out bytes.Buffer
			err := runtimeTemplate(t, source).ExecuteContext(context.Background(), &out, nil, nil, ExecutionOptions{MaxValueBytes: 1024, MaxSteps: 100000, MaxDepth: 100})
			if err != nil || out.String() != "ok" {
				t.Fatalf("got %v, %q", err, out.String())
			}
		})
	}
}

func TestBuiltinDataCycles(t *testing.T) {
	v := map[string]interface{}{}
	v["self"] = v
	for _, source := range []string{`{{ json(v) }}`, `{{ writeJson(v) }}`, `{{ v }}`, `{{ "x" + v }}`} {
		t.Run(source, func(t *testing.T) {
			err := runtimeTemplate(t, source).ExecuteContext(context.Background(), io.Discard, VarMap{}.Set("v", v), nil, ExecutionOptions{MaxValueBytes: 1 << 20, MaxOutputBytes: 1 << 20, MaxSteps: 100000, MaxDepth: 20})
			if !errors.Is(err, ErrDepthLimit) {
				t.Fatal(err)
			}
		})
	}
}

func TestBuiltinAliasesAndNativeArguments(t *testing.T) {
	vars := VarMap{}.Set("apply", func(fn func(string, int) string) string { return fn("ab", 2) }).SetFunc("native", func(a Arguments) reflect.Value {
		return reflect.ValueOf(a.Get(0).Interface().(func(string, int) string)("ab", 2))
	}).SetFunc("resolve", func(a Arguments) reflect.Value {
		return reflect.ValueOf(a.Runtime().MustResolve("repeat").Interface().(func(string, int) string)("ab", 2))
	})
	for _, tt := range []struct{ source, want string }{
		{`{{ f := repeat }}{{ f("ab", 3) }}|{{ "a" | f(2) }}|{{ f == repeat }}`, "ababab|aa|false"},
		{`{{ m := map("f", repeat) }}{{ m.f("ab", 3) }}`, "ababab"},
		{`{{ apply(repeat) }}|{{ native(repeat) }}|{{ resolve() }}`, "abab|abab|abab"},
	} {
		t.Run(tt.source, func(t *testing.T) {
			var out bytes.Buffer
			err := runtimeTemplate(t, tt.source).ExecuteContext(context.Background(), &out, vars, nil, ExecutionOptions{MaxValueBytes: 10000, MaxOutputBytes: 10000, MaxSteps: 100000, MaxDepth: 100})
			if err != nil || out.String() != tt.want {
				t.Fatalf("got %v, %q; want %q", err, out.String(), tt.want)
			}
		})
	}
}

func TestBuiltinIntegerBoundaries(t *testing.T) {
	for _, endpoints := range [][2]int64{{math.MinInt64, math.MinInt64 + 2}, {math.MaxInt64 - 2, math.MaxInt64}} {
		r := newIntsRanger(endpoints[0], endpoints[1])
		for i := int64(0); i < 2; i++ {
			index, value, done := r.Range()
			if done || index.Int() != i || value.Int() != endpoints[0]+i {
				t.Fatalf("invalid boundary range: %v %v %v", index, value, done)
			}
		}
		for i := 0; i < 2; i++ {
			if _, _, done := r.Range(); !done {
				t.Fatal("range continued after end")
			}
		}
	}
	for _, source := range []string{`{{ try }}{{ repeat("x", -1) }}{{ catch }}caught{{ end }}`, `{{ try }}{{ ints(lo, hi) }}{{ catch }}caught{{ end }}`} {
		var out bytes.Buffer
		err := runtimeTemplate(t, source).ExecuteContext(context.Background(), &out, VarMap{}.Set("lo", int64(math.MinInt64)).Set("hi", int64(math.MaxInt64)), nil, ExecutionOptions{MaxValueBytes: 10000, MaxSteps: 10000, MaxDepth: 100})
		if err != nil || out.String() != "caught" {
			t.Fatalf("got %v, %q", err, out.String())
		}
	}
}

func TestCooperativeCostHooks(t *testing.T) {
	for _, tt := range []struct {
		name string
		work uint64
		size int64
		want error
	}{
		{"steps", 1000, 0, ErrStepLimit},
		{"values", 0, 1000, ErrValueLimit},
		{"negative", 0, -1, ErrValueLimit},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			vars := VarMap{}.SetFunc("owned", func(a Arguments) reflect.Value {
				a.Runtime().ConsumeSteps(tt.work)
				a.Runtime().ReserveValueBytes(tt.size)
				reached = true
				return reflect.Value{}
			})
			err := runtimeTemplate(t, `{{ try }}{{ owned() }}{{ catch }}caught{{ end }}`).ExecuteContext(context.Background(), io.Discard, vars, nil, ExecutionOptions{MaxSteps: 100, MaxValueBytes: 100})
			if !errors.Is(err, tt.want) || reached {
				t.Fatalf("got %v; callback continued=%t", err, reached)
			}
		})
	}
}
