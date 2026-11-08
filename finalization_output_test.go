package jet

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"text/template"
)

type locatedSinkCause struct{}

func (*locatedSinkCause) Error() string { return "sink failed" }

type locatedFailWriter struct {
	cause        error
	calls        int
	failed       bool
	afterFailure int
}

func (w *locatedFailWriter) Write(b []byte) (int, error) {
	w.calls++
	if w.failed {
		w.afterFailure++
	}
	n := len(b) / 2
	w.failed = w.cause != nil || n != len(b)
	return n, w.cause
}

func TestOutputFailureLocation(t *testing.T) {
	for _, source := range []string{
		"body", `{{ "<&" }}`, `{{ raw("<&") }}`, `{{ safeHtml("<&") }}`, `{{ safeJs("<&") }}`,
		`{{ json("<&") }}`, `{{ writeJson("<&") }}`, `{{ ignored("body") }}later`, `{{ renderer }}later`,
		`{{ try }}{{ try }}body{{ end }}{{ catch }}caught{{ end }}`,
	} {
		for _, metered := range []bool{false, true} {
			t.Run(source+map[bool]string{true: "/metered", false: "/legacy"}[metered], func(t *testing.T) {
				cause := &locatedSinkCause{}
				w := &locatedFailWriter{cause: cause}
				vars := VarMap{}.SetWriter("ignored", func(w io.Writer, b []byte) { w.Write(b); w.Write(b) }).Set("renderer", RendererFunc(func(r *Runtime) { r.Writer.Write([]byte("body")); r.Writer.Write([]byte("later")) }))
				tpl, err := NewSet(NewInMemLoader()).Parse("receipt.jet", source)
				if err != nil {
					t.Fatal(err)
				}
				if metered {
					err = tpl.ExecuteContext(context.Background(), w, vars, nil, ExecutionOptions{MaxSteps: 100000, MaxDepth: 64, MaxValueBytes: 10000, MaxOutputBytes: 10000})
				} else {
					err = tpl.Execute(w, vars, nil)
				}
				var typed *locatedSinkCause
				if !errors.Is(err, cause) || !errors.As(err, &typed) || (!strings.Contains(err.Error(), `receipt.jet":1`) && !strings.Contains(err.Error(), `receipt.jet":byte 0`)) || w.calls != 1 {
					t.Fatalf("cause/location/first failure lost: %v; calls=%d", err, w.calls)
				}
			})
		}
	}
}

func TestOutputFailureNestedLocation(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("child.jet", `{{ "<&" }}`)
	set := NewSet(loader)
	tpl, err := set.Parse("receipt.jet", `{{ include "child.jet" }}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, metered := range []bool{false, true} {
		cause := &locatedSinkCause{}
		w := &locatedFailWriter{cause: cause}
		if metered {
			err = tpl.ExecuteContext(context.Background(), w, nil, nil, ExecutionOptions{})
		} else {
			err = tpl.Execute(w, nil, nil)
		}
		if !errors.Is(err, cause) || !strings.Contains(err.Error(), `child.jet":1`) {
			t.Fatalf("nested write location: %v", err)
		}
	}
}

func TestOutputFailurePrecedesLaterLimit(t *testing.T) {
	cause := &locatedSinkCause{}
	w := &locatedFailWriter{cause: cause}
	vars := VarMap{}.SetFunc("write", func(a Arguments) reflect.Value {
		a.Runtime().Writer.Write([]byte("body"))
		func() { defer func() { recover() }(); a.Runtime().ConsumeSteps(100000) }()
		return reflect.Value{}
	})
	tpl, err := NewSet(NewInMemLoader()).Parse("receipt.jet", `{{ write() }}`)
	if err != nil {
		t.Fatal(err)
	}
	err = tpl.ExecuteContext(context.Background(), w, vars, nil, ExecutionOptions{MaxSteps: 1000})
	if !errors.Is(err, cause) || errors.Is(err, ErrStepLimit) || !strings.Contains(err.Error(), "receipt.jet") {
		t.Fatalf("first failure replaced: %v", err)
	}
}

func TestOutputReentrantEscaper(t *testing.T) {
	for _, metered := range []bool{false, true} {
		var runtime *Runtime
		first := true
		set := NewSet(NewInMemLoader(), WithSafeWriter(func(w io.Writer, b []byte) {
			if runtime != nil && first {
				first = false
				runtime.YieldBlock("inner", nil)
			}
			template.HTMLEscape(w, b)
		}))
		tpl, err := set.Parse("receipt.jet", `{{ block inner() }}{{ try }}{{ "&" }}{{ end }}{{ end }}{{ capture() }}{{ "<" }}{{ ">" }}`)
		if err != nil {
			t.Fatal(err)
		}
		vars := VarMap{}.SetFunc("capture", func(a Arguments) reflect.Value { runtime = a.Runtime(); return reflect.Value{} })
		var out bytes.Buffer
		if metered {
			err = tpl.ExecuteContext(context.Background(), &out, vars, nil, ExecutionOptions{MaxSteps: 10000, MaxDepth: 64, MaxValueBytes: 10000, MaxOutputBytes: 10000})
		} else {
			err = tpl.Execute(&out, vars, nil)
		}
		if err != nil || out.String() != "&amp;&amp;&lt;&gt;" {
			t.Fatalf("reentrant escaping: %q, %v", out.String(), err)
		}
	}
}

type outputWriterFunc func([]byte) (int, error)

func (f outputWriterFunc) Write(b []byte) (int, error) { return f(b) }

func TestOutputReentrantCommitCountsGeneration(t *testing.T) {
	var active *Runtime
	entered := false
	var out bytes.Buffer
	sink := outputWriterFunc(func(b []byte) (int, error) {
		if active != nil && !entered {
			entered = true
			active.YieldBlock("inner", nil)
		}
		return out.Write(b)
	})
	tpl, err := NewSet(NewInMemLoader(), WithSafeWriter(nil)).Parse("receipt.jet", `{{ block inner() }}b{{ end }}{{ capture() }}{{ try }}a{{ end }}`)
	if err != nil {
		t.Fatal(err)
	}
	vars := VarMap{}.SetFunc("capture", func(a Arguments) reflect.Value { active = a.Runtime(); return reflect.Value{} })
	err = tpl.ExecuteContext(context.Background(), sink, vars, nil, ExecutionOptions{MaxSteps: 10000, MaxDepth: 64, MaxValueBytes: 10000, MaxOutputBytes: 2})
	if !entered || !errors.Is(err, ErrOutputLimit) || out.String() != "b" {
		t.Fatalf("reentrant commit generation: %q, %v, entered=%t", out.String(), err, entered)
	}
}

func TestOutputShortWriteLocation(t *testing.T) {
	for _, metered := range []bool{false, true} {
		tpl, err := NewSet(NewInMemLoader()).Parse("receipt.jet", `{{ safeHtml("<body>") }}`)
		if err != nil {
			t.Fatal(err)
		}
		w := &locatedFailWriter{}
		if metered {
			err = tpl.ExecuteContext(context.Background(), w, nil, nil, ExecutionOptions{})
		} else {
			err = tpl.Execute(w, nil, nil)
		}
		if !errors.Is(err, io.ErrShortWrite) || !strings.Contains(err.Error(), "receipt.jet") || !w.failed || w.afterFailure != 0 {
			t.Fatalf("short write cause/location: %v, calls=%d", err, w.calls)
		}
	}
}

func TestOutputBoundedLocation(t *testing.T) {
	name := strings.Repeat("segment/", 1024) + "receipt.jet"
	tpl, err := NewSet(NewInMemLoader()).Parse(name, "body")
	if err != nil {
		t.Fatal(err)
	}
	cause := &locatedSinkCause{}
	err = tpl.Execute(&locatedFailWriter{cause: cause}, nil, nil)
	if !errors.Is(err, cause) || len(err.Error()) > 1200 || !strings.Contains(err.Error(), ":byte 0") {
		t.Fatalf("unbounded or absent location: %v", err)
	}
}

func TestOutputOwnedLargeThenSmall(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `{{ v }}`}, restrictedTestOptions())
	large := strings.Repeat("x", 65536)
	saved, err := tpl.RenderRestrictedContext(context.Background(), VarMap{}.Set("v", large), nil)
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	for _, small := range []string{"", "ok", "<", ""} {
		result, err := tpl.RenderRestrictedContext(context.Background(), VarMap{}.Set("v", small), nil)
		if err != nil || result != small || saved != large {
			t.Fatalf("owned result changed across cold/small reuse: len=%d err=%v", len(result), err)
		}
	}
}

func TestOutputRedirectedWriterFailure(t *testing.T) {
	for _, source := range []string{`{{ redirect() }}{{ "body" }}`, `{{ redirect() }}{{ try }}body{{ end }}`} {
		for _, metered := range []bool{false, true} {
			cause := &locatedSinkCause{}
			sink := &locatedFailWriter{cause: cause}
			vars := VarMap{}.SetFunc("redirect", func(a Arguments) reflect.Value { a.Runtime().Writer = sink; return reflect.Value{} })
			tpl, err := NewSet(NewInMemLoader()).Parse("receipt.jet", source)
			if err != nil {
				t.Fatal(err)
			}
			if metered {
				err = tpl.ExecuteContext(context.Background(), io.Discard, vars, nil, ExecutionOptions{})
			} else {
				err = tpl.Execute(io.Discard, vars, nil)
			}
			if !errors.Is(err, cause) || !strings.Contains(err.Error(), "receipt.jet") {
				t.Fatalf("redirected output %s lost failure: %v", source, err)
			}
		}
	}
}
