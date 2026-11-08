package jet

import (
	"bytes"
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

type boundaryInner struct{ Value string }
type boundaryOuter struct{ *boundaryInner }
type boundaryStruct struct {
	Value  string
	hidden string
}

func TestTrustedReflectionControls(t *testing.T) {
	for _, mode := range []string{"legacy", "metered"} {
		for _, tc := range []struct {
			name, source string
			data         interface{}
			vars         VarMap
			want         string
			wantError    bool
		}{
			{"struct assignment", `{{ .Value = value }}{{ .Value }}`, &boundaryStruct{Value: "old"}, VarMap{}.Set("value", "new"), "new", false},
			{"unaddressable struct", `{{ .Value = value }}`, boundaryStruct{}, VarMap{}.Set("value", "new"), "", true},
			{"inaccessible struct", `{{ .hidden = value }}`, &boundaryStruct{}, VarMap{}.Set("value", "new"), "", true},
			{"struct type mismatch", `{{ .Value = 1 }}`, &boundaryStruct{}, nil, "", true},
			{"nil struct", `{{ .Value = value }}`, (*boundaryStruct)(nil), VarMap{}.Set("value", "new"), "", true},
			{"nil embedded lookup", `{{ .Value }}`, boundaryOuter{}, nil, "", true},
			{"nil embedded assignment", `{{ .Value = value }}`, &boundaryOuter{}, VarMap{}.Set("value", "new"), "", true},
			{"invalid assignment target", `{{ .x = 1 }}`, 0, nil, "", true},
			{"unhashable key", `{{ .[key] }}`, map[interface{}]string{}, VarMap{}.Set("key", []int{1}), "", true},
			{"unhashable nested key", `{{ .[key] }}`, map[interface{}]string{}, VarMap{}.Set("key", [1]interface{}{[]int{1}}), "", true},
			{"comparable key", `{{ .[key] }}`, map[interface{}]string{[1]interface{}{1}: "ok"}, VarMap{}.Set("key", [1]interface{}{1}), "ok", false},
			{"large unsigned index", `{{ .[index] }}`, []int{1}, VarMap{}.Set("index", uint64(math.MaxUint64)), "", true},
			{"large slice index", `{{ .[index:] }}`, []int{1}, VarMap{}.Set("index", uint64(math.MaxUint64)), "", true},
			{"nan index", `{{ .[index] }}`, []int{1}, VarMap{}.Set("index", math.NaN()), "", true},
			{"float index", `{{ .[index] }}`, []int{1}, VarMap{}.Set("index", -0.5), "1", false},
			{"float slice", `{{ .[start:stop] }}`, []int{1, 2}, VarMap{}.Set("start", 0.9).Set("stop", 1.9), "[1]", false},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("escaped %T: %v", p, p)
					}
				}()
				out, err := renderBoundary(t, mode, tc.source, tc.vars, tc.data)
				if (err != nil) != tc.wantError || (!tc.wantError && out != tc.want) {
					t.Fatalf("got %q, %v", out, err)
				}
			})
		}
	}
}

func TestArraySliceReservationAndIsolation(t *testing.T) {
	source := `{{ .[:] }}`
	array := [2]int{1, 2}
	for _, tc := range []struct {
		limit int64
		want  error
	}{{int64(reflect.TypeOf(array).Size()) - 1, ErrValueLimit}, {int64(reflect.TypeOf(array).Size()), nil}} {
		var out bytes.Buffer
		err := runtimeTemplate(t, source).ExecuteContext(context.Background(), &out, nil, array, ExecutionOptions{MaxValueBytes: tc.limit, MaxSteps: 1000, MaxDepth: 64})
		if !errors.Is(err, tc.want) {
			t.Fatalf("got %v, want %v", err, tc.want)
		}
	}
	data := map[boundaryString]string{"x": "original"}
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `{{ .x = value }}{{ .x }}`}, restrictedTestOptions())
	out, err := tpl.RenderRestrictedContext(context.Background(), VarMap{}.Set("value", "changed"), data)
	if err != nil || out != "changed" || data["x"] != "original" {
		t.Fatalf("snapshot mutated: %q, %v, %v", out, err, data)
	}
}

type catchHookError struct{ called *bool }

func (e catchHookError) Error() string { *e.called = true; return "native error" }

func TestCatchDiagnosticDataOnly(t *testing.T) {
	called := false
	st := &Runtime{escapeeWriter: new(escapeeWriter)}
	got := st.catchDiagnostic(catchHookError{&called})
	if called || got != "template execution failed" {
		t.Fatalf("invoked native hook: %t, %q", called, got)
	}
	message := strings.Repeat("界", 300)
	got = st.catchDiagnostic(&templateError{message: message})
	if len(got) > 512 || !utf8.ValidString(got) || !strings.HasSuffix(got, "...") {
		t.Fatalf("invalid bounded diagnostic: %q", got)
	}

	opts := restrictedTestOptions()
	opts.Execution.MaxValueBytes = 40
	for _, bind := range []bool{false, true} {
		source := `{{ try }}{{ missing }}{{ catch }}{{ end }}`
		if bind {
			source = `{{ try }}{{ missing }}{{ catch err }}{{ end }}`
		}
		tpl := restrictedTemplate(t, map[string]string{"main.jet": source}, opts)
		out, err := tpl.RenderRestrictedContext(context.Background(), nil, nil)
		if bind {
			if !errors.Is(err, ErrValueLimit) || out != "" {
				t.Fatalf("diagnostic ignored reservation: %q, %v", out, err)
			}
		} else if err != nil || out != "" {
			t.Fatalf("unbound catch: %q, %v", out, err)
		}
	}
}

func TestBuiltinMapLegacyKeyOnce(t *testing.T) {
	for _, mode := range []string{"legacy", "metered"} {
		calls := 0
		vars := VarMap{}.SetFunc("key", func(a Arguments) reflect.Value { calls++; return reflect.ValueOf([]byte("x")) })
		out, err := renderBoundary(t, mode, `{{ m:=map(key(), "ok") }}{{ m.x }}`, vars, nil)
		if err != nil || out != "ok" || calls != 1 {
			t.Fatalf("%s: got %q, %v, calls=%d", mode, out, err, calls)
		}
	}
}

func TestBuiltinInaccessibleArguments(t *testing.T) {
	value := reflect.ValueOf(struct{ hidden string }{"value"}).Field(0)
	for _, mode := range []string{"legacy", "metered"} {
		for _, source := range []string{`{{ map(hidden, "value") }}`, `{{ map("key", hidden) }}`, `{{ slice(hidden) }}`, `{{ array(hidden) }}`} {
			t.Run(mode+source, func(t *testing.T) {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("inaccessible argument escaped as %T: %v", p, p)
					}
				}()
				_, err := renderBoundary(t, mode, source, VarMap{"hidden": value}, nil)
				if err == nil || errors.Is(err, ErrCapabilityDenied) {
					t.Fatalf("want ordinary argument error: %v", err)
				}
			})
		}
	}
}

func TestTemplateErrorCauseCompatibility(t *testing.T) {
	cause := errors.New("original cause")
	for _, mode := range []string{"legacy", "metered"} {
		vars := VarMap{}.SetFunc("fail", func(a Arguments) reflect.Value { a.Panicf("helper: %w", cause); return reflect.Value{} })
		_, err := renderBoundary(t, mode, `{{ fail() }}`, vars, nil)
		if !errors.Is(err, cause) || err.Error() != "helper: original cause" {
			t.Fatalf("%s: lost cause: %v", mode, err)
		}
		out, err := renderBoundary(t, mode, `{{ try }}{{ fail() }}{{ catch err }}{{ err.Error() }}{{ end }}`, vars, nil)
		if err != nil || out != "helper: original cause" {
			t.Fatalf("trusted catch changed: %q, %v", out, err)
		}
	}
}

func TestRestrictedCatchHTMLAndJSON(t *testing.T) {
	for _, html := range []bool{false, true} {
		opts := restrictedTestOptions()
		opts.Text = !html
		for _, source := range []string{
			`{{ try }}{{ missing }}{{ catch err }}{{ err }}{{ end }}`,
			`{{ try }}{{ missing }}{{ catch err }}{{ json(map("message", err)) }}{{ end }}`,
		} {
			tpl := restrictedTemplate(t, map[string]string{"main.jet": source}, opts)
			out, err := tpl.RenderRestrictedContext(context.Background(), nil, nil)
			if err != nil || !strings.Contains(out, "not available") {
				t.Fatalf("html=%t: %q, %v", html, out, err)
			}
		}
	}
}

func TestDumpNilGlobals(t *testing.T) {
	for _, source := range []string{`{{ dump() }}`, `{{ dump("nothing") }}`} {
		tpl := runtimeTemplate(t, source)
		tpl.set.AddGlobal("nothing", nil)
		var out bytes.Buffer
		if err := tpl.Execute(&out, nil, nil); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "nothing") {
			t.Fatalf("missing nil binding: %q", out.String())
		}
	}
}
