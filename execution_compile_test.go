package jet

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestExecutionCompilationInitialCachePolicy(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("root.jet", strings.Repeat("x", 100))
	tpl, err := NewSet(loader).GetTemplate("root.jet")
	if err != nil {
		t.Fatal(err)
	}
	err = tpl.ExecuteContext(context.Background(), io.Discard, nil, nil, ExecutionOptions{ParseOptions: ParseOptions{MaxSourceBytes: 99}})
	if !errors.Is(err, ErrParseSourceLimit) {
		t.Fatal(err)
	}
}

func TestExecutionCompilationSharedGraph(t *testing.T) {
	for _, source := range []string{`{{include "a.jet"}}{{include "b.jet"}}`, `{{exec("a.jet")}}{{includeIfExists("b.jet")}}`} {
		loader := NewInMemLoader()
		loader.Set("a.jet", "A")
		loader.Set("b.jet", "B")
		tpl, err := NewSet(loader).Parse("root.jet", source)
		if err != nil {
			t.Fatal(err)
		}
		for _, limit := range []int64{int64(len(source)) + 1, int64(len(source)) + 2} {
			err = tpl.ExecuteContext(context.Background(), io.Discard, nil, nil, ExecutionOptions{ParseOptions: ParseOptions{MaxTotalSourceBytes: limit}})
			if limit == int64(len(source))+1 {
				if !errors.Is(err, ErrParseTotalSourceLimit) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestExecutionCompilationLimitsAreTerminal(t *testing.T) {
	for _, source := range []string{`{{try}}{{include "child.jet"}}{{catch}}caught{{end}}`, `{{try}}{{exec("child.jet")}}{{catch}}caught{{end}}`, `{{try}}{{includeIfExists("child.jet")}}{{catch}}caught{{end}}`, `{{isset(a[includeIfExists("child.jet")])}}`} {
		loader := NewInMemLoader()
		loader.Set("child.jet", strings.Repeat("x", 100))
		tpl, err := NewSet(loader).Parse("root.jet", source)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err = tpl.ExecuteContext(context.Background(), &out, VarMap{}.Set("a", map[string]string{}), nil, ExecutionOptions{ParseOptions: ParseOptions{MaxSourceBytes: 80}})
		if !errors.Is(err, ErrParseSourceLimit) || out.Len() != 0 {
			t.Fatalf("%s: %v output=%q", source, err, out.String())
		}
	}
}

func TestExecutionCompilationTemplateReuse(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("child.jet", "ok")
	tpl, err := NewSet(loader).Parse("root.jet", `{{include "child.jet"}}{{include "child.jet"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = tpl.ExecuteContext(context.Background(), &out, nil, nil, ExecutionOptions{ParseOptions: ParseOptions{MaxTemplates: 2}})
	if err != nil || out.String() != "okok" {
		t.Fatalf("%v: %q", err, out.String())
	}
	err = tpl.ExecuteContext(context.Background(), io.Discard, nil, nil, ExecutionOptions{ParseOptions: ParseOptions{MaxTemplates: 1}})
	if !errors.Is(err, ErrParseTemplateLimit) {
		t.Fatal(err)
	}
}

func TestExecutionCompilationCancelReaderCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closed := false
	loader := compilationLoader{open: func() (io.ReadCloser, error) {
		return compilationReadCloser{Reader: compilationCancelReader{cancel}, close: func() error { closed = true; return nil }}, nil
	}}
	tpl, err := NewSet(loader).Parse("root.jet", `{{try}}{{include "child.jet"}}{{catch}}caught{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = tpl.ExecuteContext(ctx, &out, nil, nil, ExecutionOptions{})
	if !errors.Is(err, context.Canceled) || !closed || out.Len() != 0 {
		t.Fatalf("%v: closed=%t output=%q", err, closed, out.String())
	}
}

func TestExecutionCompilationCloseCannotMaskLimit(t *testing.T) {
	failure := errors.New("close panic")
	loader := compilationLoader{open: func() (io.ReadCloser, error) {
		return compilationReadCloser{Reader: strings.NewReader(strings.Repeat("x", 100)), close: func() error { panic(failure) }}, nil
	}}
	tpl, err := NewSet(loader).Parse("root.jet", `{{try}}{{include "child.jet"}}{{catch}}caught{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = tpl.ExecuteContext(context.Background(), &out, nil, nil, ExecutionOptions{ParseOptions: ParseOptions{MaxSourceBytes: 80}})
	if !errors.Is(err, ErrParseSourceLimit) || out.Len() != 0 {
		t.Fatalf("%v: %q", err, out.String())
	}
}
