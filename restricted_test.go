package jet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

func restrictedTestOptions() RestrictedOptions {
	return RestrictedOptions{
		Execution: ExecutionOptions{MaxSteps: 1000000, MaxDepth: 64, MaxValueBytes: 8 << 20, MaxOutputBytes: 1 << 20,
			ParseOptions: ParseOptions{MaxSourceBytes: 65536, MaxTotalSourceBytes: 262144, MaxNodes: 20000, MaxSteps: 1000000, MaxDepth: 64, MaxTemplates: 32}},
		Input: InputOptions{MaxBytes: 262144, MaxItems: 10000, MaxSteps: 1000000, MaxDepth: 64}, Text: true,
	}
}
func restrictedTemplate(t *testing.T, sources map[string]string, opts RestrictedOptions) *Template {
	t.Helper()
	set, err := NewRestrictedSet(context.Background(), sources, opts)
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := set.GetTemplateContext(context.Background(), "main.jet", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return tpl
}

func TestRestrictedSupportedWorkflow(t *testing.T) {
	opts := restrictedTestOptions()
	binding, ok := LookupBuiltin("upper")
	if !ok {
		t.Fatal("missing upper")
	}
	opts.BuiltinAliases = map[string]BuiltinBinding{"capitalize": binding}
	opts.OptionalTemplates = []string{"optional.jet"}
	sources := map[string]string{
		"main.jet":    `{{ f := capitalize }}{{ funcs := array(f) }}{{ f = funcs[0] }}{{ f("ok") }}|{{ m := map("f", upper) }}{{ f = m.f }}{{ f("yes") }}|{{ f = exec("binding.jet") }}{{ f("go") }}|{{ include "part.jet" . }}|{{ includeIfExists("optional.jet") }}{{ range i := ints(1,4) }}{{ i }}{{ end }}|{{ writeJson(map("b", array(1,2))) }}`,
		"binding.jet": `{{ return upper }}`,
		"part.jet":    `{{ .message }}:{{ "<" | safeHtml }}`,
	}
	tpl := restrictedTemplate(t, sources, opts)
	result, err := tpl.RenderRestrictedContext(context.Background(), nil, map[string]string{"message": "data"})
	if err != nil || result != "OK|YES|GO|data:&lt;|012|{\"b\":[1,2]}\n" {
		t.Fatalf("result=%q err=%v", result, err)
	}
}

type restrictedFormatter struct{ called *bool }

func (v restrictedFormatter) Format(s fmt.State, verb rune) { *v.called = true }

type restrictedError struct{ called *bool }

func (v restrictedError) Error() string { *v.called = true; return "error" }

type restrictedTextMarshaler struct{ called *bool }

func (v restrictedTextMarshaler) MarshalText() ([]byte, error) {
	*v.called = true
	return []byte("text"), nil
}

type restrictedMethod string

func (*restrictedMethod) Method() string { panic("method invoked") }

func TestRestrictedDenialWorkflow(t *testing.T) {
	called := false
	fn := func() string { called = true; return "called" }
	cases := []struct {
		name, source string
		value        interface{}
	}{
		{"function", `{{ value() }}`, fn},
		{"shadow", `{{ upper() }}`, fn},
		{"reentry", `{{ value() }}`, Func(func(a Arguments) reflect.Value { called = true; _ = a; return reflect.Value{} })},
		{"nested-map", `{{ value.fn() }}`, map[string]interface{}{"fn": fn}},
		{"nested-slice", `{{ f := value[0] }}{{ f() }}`, []interface{}{fn}},
		{"stringer", `{{ value }}`, inputStringer{&called}},
		{"formatter", `{{ "x" + value }}`, restrictedFormatter{&called}},
		{"error", `{{ value }}`, restrictedError{&called}},
		{"json", `{{ json(value) }}`, inputMarshaler{&called}},
		{"text-json", `{{ writeJson(value) }}`, restrictedTextMarshaler{&called}},
		{"renderer", `{{ value }}`, inputRenderer{&called}},
		{"ranger", `{{ range value }}x{{ end }}`, inputRanger{&called}},
		{"writer", `{{ "x" | value }}`, SafeWriter(func(io.Writer, []byte) { called = true })},
		{"pointer-method", `{{ value.Method() }}`, restrictedMethod("data")},
		{"channel", `{{ range value }}x{{ end }}`, make(chan string)},
		{"unused", `ok`, fn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			tpl := restrictedTemplate(t, map[string]string{"main.jet": tc.source}, restrictedTestOptions())
			vars := VarMap{}.Set("value", tc.value).Set("upper", tc.value)
			result, err := tpl.RenderRestrictedContext(context.Background(), vars, nil)
			if !errors.Is(err, ErrCapabilityDenied) || result != "" || called {
				t.Fatalf("result=%q err=%v called=%t", result, err, called)
			}
		})
	}
	// Global admission uses the same boundary even when a template never reads it.
	opts := restrictedTestOptions()
	opts.Globals = VarMap{}.Set("value", fn)
	if _, err := NewRestrictedSet(context.Background(), map[string]string{"main.jet": "ok"}, opts); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
}

func TestRestrictedTerminalAndCeilings(t *testing.T) {
	for _, source := range []string{`{{ try }}{{ dump() }}{{ catch }}caught{{ end }}`, `{{ isset(dump()[0]) }}`, `{{ f := dump }}{{ f() }}`, `{{ m := map("fn",dump) }}{{ f := m.fn }}{{ f() }}`, `{{ f := exec("bad.jet") }}{{ f() }}`} {
		tpl := restrictedTemplate(t, map[string]string{"main.jet": source, "bad.jet": `{{ return dump }}`}, restrictedTestOptions())
		result, err := tpl.RenderRestrictedContext(context.Background(), nil, nil)
		if !errors.Is(err, ErrCapabilityDenied) || result != "" {
			t.Fatalf("%s: %q %v", source, result, err)
		}
	}
	for _, kind := range []string{"steps", "depth", "value", "output", "parse"} {
		t.Run(kind, func(t *testing.T) {
			opts := restrictedTestOptions()
			source := `{{ range i := ints(0,1000) }}{{ i }}{{ end }}`
			want := ErrStepLimit
			switch kind {
			case "steps":
				opts.Execution.MaxSteps = 10
			case "depth":
				opts.Execution.MaxDepth = 2
				want = ErrDepthLimit
			case "value":
				opts.Execution.MaxValueBytes = 8
				want = ErrValueLimit
			case "output":
				opts.Execution.MaxOutputBytes = 2
				want = ErrOutputLimit
			case "parse":
				source = `{{ exec("part.jet") }}`
				opts.Execution.ParseOptions.MaxTotalSourceBytes = int64(len(source) + 4)
				want = ErrParseTotalSourceLimit
			}
			sources := map[string]string{"main.jet": source}
			var tpl *Template
			if kind == "parse" {
				set, err := NewRestrictedSet(context.Background(), map[string]string{"part.jet": "12345"}, opts)
				if err != nil {
					t.Fatal(err)
				}
				tpl, err = set.Parse("main.jet", source)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				tpl = restrictedTemplate(t, sources, opts)
			}
			result, err := tpl.RenderRestrictedContext(context.Background(), nil, nil)
			if !errors.Is(err, want) || result != "" {
				t.Fatalf("%q %v want %v", result, err, want)
			}
			// Explicit zero/looser per-call settings also cannot loosen the policy.
			sink := &restrictedSink{owner: tpl}
			err = tpl.ExecuteContext(context.Background(), sink, nil, nil, restrictedTestOptions().Execution)
			if !errors.Is(err, want) {
				t.Fatalf("looser options: %v", err)
			}
		})
	}
}

func TestRestrictedSnapshotsAndNamespace(t *testing.T) {
	globals := map[string]string{"message": "original"}
	opts := restrictedTestOptions()
	opts.Globals = VarMap{}.Set("g", globals)
	opts.OptionalTemplates = []string{"missing.jet"}
	sources := map[string]string{"main.jet": `{{ g.message }}|{{ include "part.jet" }}|{{ includeIfExists("missing.jet") }}`, "part.jet": "part"}
	tpl := restrictedTemplate(t, sources, opts)
	globals["message"] = "changed"
	sources["part.jet"] = "changed"
	escaped, _ := tpl.set.LookupGlobal("g")
	escaped.(reflect.Value).Interface().(map[string]string)["message"] = "escaped"
	for i := 0; i < 2; i++ {
		result, err := tpl.RenderRestrictedContext(context.Background(), nil, nil)
		if err != nil || result != "original|part|" {
			t.Fatalf("%q %v", result, err)
		}
	}
	for _, source := range []string{`{{ include "../secret.jet" }}`, `{{ try }}{{ exec("secret.jet") }}{{ catch }}caught{{ end }}`, `{{ isset(includeIfExists("secret.jet")[0]) }}`} {
		inline, err := tpl.set.Parse("inline.jet", source)
		if err != nil {
			t.Fatal(err)
		}
		result, err := inline.RenderRestrictedContext(context.Background(), nil, nil)
		if !errors.Is(err, ErrCapabilityDenied) || result != "" {
			t.Fatalf("%q %v", result, err)
		}
	}
	for _, mutate := range []func(){func() { tpl.set.AddGlobal("x", true) }, func() { WithSafeWriter(nil)(tpl.set) }, func() { WithCache(&cache{})(tpl.set) }, func() { WithDelims("<%", "%>")(tpl.set) }, func() { DevelopmentMode(true)(tpl.set) }, func() { WithTemplateNameExtensions([]string{".x"})(tpl.set) }} {
		func() {
			defer func() {
				if err, ok := recover().(error); !ok || !errors.Is(err, ErrCapabilityDenied) {
					t.Fatalf("mutation: %v", err)
				}
			}()
			mutate()
		}()
	}
}

func TestRestrictedInputLimitsAndSemantics(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `{{ if .array }}array{{ end }}|{{ len(.nil) }}|{{ .bytes }}|{{ json(.nil) | raw }}|{{ .named }}`}, restrictedTestOptions())
	type text string
	result, err := tpl.RenderRestrictedContext(context.Background(), nil, map[string]interface{}{"array": [1]int{0}, "nil": []string(nil), "bytes": []byte("bytes"), "named": text("named")})
	if err != nil || result != "|0|bytes|null|named" {
		t.Fatalf("%q %v", result, err)
	}
	cycle := map[string]interface{}{}
	cycle["self"] = cycle
	for _, value := range []interface{}{cycle, []interface{}{cycle}, new(interface{})} {
		if result, err := tpl.RenderRestrictedContext(context.Background(), nil, value); !errors.Is(err, ErrCapabilityDenied) || result != "" {
			t.Fatalf("%q %v", result, err)
		}
	}
	opts := restrictedTestOptions()
	opts.Input.MaxBytes = 1
	tpl = restrictedTemplate(t, map[string]string{"main.jet": "ok"}, opts)
	if _, err := tpl.RenderRestrictedContext(context.Background(), nil, "large"); !errors.Is(err, ErrInputLimit) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := tpl.RenderRestrictedContext(ctx, nil, nil); !errors.Is(err, context.Canceled) || result != "" {
		t.Fatalf("%q %v", result, err)
	}
}

func TestRestrictedMixedConcurrency(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `{{ upper(.text) }}`}, restrictedTestOptions())
	trusted := NewSet(NewInMemLoader())
	trusted.AddGlobal("upper", func(s string) string { return "trusted:" + s })
	other, err := trusted.Parse("main.jet", `{{ upper(.text) }}`)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 10; n++ {
				result, err := tpl.RenderRestrictedContext(context.Background(), nil, map[string]string{"text": "ok"})
				if err != nil || result != "OK" {
					t.Errorf("%q %v", result, err)
					return
				}
				var b bytes.Buffer
				err = other.ExecuteContext(context.Background(), &b, nil, map[string]string{"text": "ok"}, ExecutionOptions{})
				if err != nil || b.String() != "trusted:ok" {
					t.Errorf("%q %v", b.String(), err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestRestrictedRejectsExternalWriter(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": "hello"}, restrictedTestOptions())
	called := false
	writer := restrictedCallbackWriter(func([]byte) (int, error) { called = true; return 0, nil })
	if err := tpl.ExecuteContext(context.Background(), writer, nil, nil, ExecutionOptions{}); !errors.Is(err, ErrCapabilityDenied) || called {
		t.Fatalf("%v %t", err, called)
	}
	if err := tpl.Execute(writer, nil, nil); !errors.Is(err, ErrCapabilityDenied) || called {
		t.Fatalf("%v %t", err, called)
	}
}

type restrictedCallbackWriter func([]byte) (int, error)

func (w restrictedCallbackWriter) Write(b []byte) (int, error) { return w(b) }

func TestRestrictedDepthChild(t *testing.T) {
	if os.Getenv("JET_RESTRICTED_DEPTH_CHILD") == "1" {
		debug.SetMaxStack(256 << 10)
		opts := restrictedTestOptions()
		sources := map[string]string{"main.jet": `{{ block b() }}{{ yield b() }}{{ end }}`}
		tpl := restrictedTemplate(t, sources, opts)
		if _, err := tpl.RenderRestrictedContext(context.Background(), nil, nil); !errors.Is(err, ErrDepthLimit) {
			t.Fatal(err)
		}
		deep := "x"
		for i := 0; i < 100; i++ {
			deep = "{{ if true }}" + deep + "{{ end }}"
		}
		if _, err := tpl.set.Parse("deep.jet", deep); !errors.Is(err, ErrParseDepthLimit) {
			t.Fatal(err)
		}
		var data interface{} = "x"
		for i := 0; i < 100; i++ {
			data = []interface{}{data}
		}
		if _, err := tpl.RenderRestrictedContext(context.Background(), nil, data); !errors.Is(err, ErrInputLimit) {
			t.Fatal(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRestrictedDepthChild$", "-test.timeout=8s")
	cmd.Env = append(os.Environ(), "JET_RESTRICTED_DEPTH_CHILD=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v\n%s", err, output)
	}
}

func TestRestrictedConfiguration(t *testing.T) {
	if _, ok := LookupBuiltin("dump"); ok {
		t.Fatal("diagnostic binding exposed")
	}
	opts := restrictedTestOptions()
	opts.BuiltinAliases = map[string]BuiltinBinding{"x": {}}
	if _, err := NewRestrictedSet(context.Background(), nil, opts); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
	opts = restrictedTestOptions()
	opts.Execution.MaxSteps = 0
	if _, err := NewRestrictedSet(context.Background(), nil, opts); !errors.Is(err, ErrInvalidOptions) {
		t.Fatal(err)
	}
	opts = restrictedTestOptions()
	opts.Execution.MaxDepth = 65
	if _, err := NewRestrictedSet(context.Background(), nil, opts); !errors.Is(err, ErrInvalidOptions) {
		t.Fatal(err)
	}
	opts = restrictedTestOptions()
	opts.OptionalTemplates = []string{strings.Repeat("x", 1024)}
	if _, err := NewRestrictedSet(context.Background(), nil, opts); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
}

func TestRestrictedForeignCacheAndConfiguration(t *testing.T) {
	opts := restrictedTestOptions()
	opts.Text = false
	opts.LeftDelim = "<<"
	opts.RightDelim = ">>"
	opts.LeftComment = "/*"
	opts.RightComment = "*/"
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `<< include "part.jet" >>`, "part.jet": `/* comment */<< . >>`}, opts)
	foreign, err := NewSet(NewInMemLoader()).Parse("part.jet", "foreign")
	if err != nil {
		t.Fatal(err)
	}
	tpl.set.cache.Put("/part.jet", foreign)
	tpl.set.cache.Put("/secret.jet", foreign)
	result, err := tpl.RenderRestrictedContext(context.Background(), nil, "<safe>")
	if err != nil || result != "&lt;safe&gt;" {
		t.Fatalf("%q %v", result, err)
	}
	if _, err := tpl.set.GetTemplate("secret.jet"); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
	// Legacy compilation calls also retain the immutable ceilings.
	if _, err := tpl.set.Parse("large.jet", strings.Repeat("x", 65537)); !errors.Is(err, ErrParseSourceLimit) {
		t.Fatal(err)
	}
}

func TestRestrictedUnknownKindsAndNativeDefenses(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": "ok"}, restrictedTestOptions())
	for _, test := range []struct {
		name string
		call func(*Runtime)
	}{
		{"native-call", func(st *Runtime) { st.beforeCall(reflect.ValueOf(func() {})) }},
		{"forged-binding", func(st *Runtime) { st.beforeCall(reflect.ValueOf(&builtinDescriptor{contract: builtinString})) }},
		{"native-renderer", func(st *Runtime) { called := false; st.beforeCall(reflect.ValueOf(inputRenderer{&called})) }},
		{"format-fallback", func(st *Runtime) { st.beforeNative(reflect.ValueOf(0)) }},
		{"pointer-index", func(st *Runtime) { value := 1; st.resolveIndex(reflect.ValueOf(&value), reflect.Value{}, "Method") }},
		{"function-conversion", func(st *Runtime) { st.convertValue(reflect.ValueOf(upperDummy), reflect.TypeOf(upperDummy)) }},
		{"new-AST-kind", func(st *Runtime) {
			st.executeList(&ListNode{Nodes: []Node{&TextNode{NodeBase: NodeBase{NodeType: NodeType(999)}}}})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			st := &Runtime{escapeeWriter: &escapeeWriter{set: tpl.set}, execution: executionState{ctx: context.Background(), metered: true, options: restrictedTestOptions().Execution, remaining: 1000000}}
			defer func() {
				abort, ok := recover().(executionAbort)
				if !ok || !errors.Is(abort.error, ErrCapabilityDenied) {
					t.Fatalf("got %#v", abort)
				}
			}()
			test.call(st)
		})
	}
}
func upperDummy(string) string { return "unused" }

func TestExecutionOwnsBlockingWriter(t *testing.T) {
	entered, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
	writer := restrictedCallbackWriter(func(b []byte) (int, error) { close(entered); <-release; close(exited); return len(b), nil })
	tpl, err := NewSet(NewInMemLoader()).Parse("writer.jet", "hello")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		err := tpl.ExecuteContext(ctx, writer, nil, nil, ExecutionOptions{})
		select {
		case <-exited:
		default:
			err = errors.New("returned before writer exit")
		}
		result <- err
	}()
	<-entered
	cancel()
	select {
	case err := <-result:
		close(release)
		t.Fatalf("returned while writer owns execution: %v", err)
	default:
	}
	close(release)
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRestrictedGeneratedCyclesAndMutableSnapshots(t *testing.T) {
	for _, source := range []string{`{{ m := map() }}{{ m.self = m }}{{ m }}`, `{{ m := map() }}{{ m.self = m }}{{ json(m) }}`} {
		tpl := restrictedTemplate(t, map[string]string{"main.jet": source}, restrictedTestOptions())
		result, err := tpl.RenderRestrictedContext(context.Background(), nil, nil)
		if !errors.Is(err, ErrDepthLimit) || result != "" {
			t.Fatalf("%s: %q %v", source, result, err)
		}
	}
	opts := restrictedTestOptions()
	opts.Globals = VarMap{}.Set("g", map[string]string{"value": "global"})
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `{{ g.value }}{{ g.value = "mutated" }}:{{ .value }}{{ .value = "changed" }}`}, opts)
	data := map[string]string{"value": "caller"}
	for i := 0; i < 2; i++ {
		result, err := tpl.RenderRestrictedContext(context.Background(), nil, data)
		if err != nil || result != "global:caller" || data["value"] != "caller" {
			t.Fatalf("%q %v %v", result, err, data)
		}
	}
}
