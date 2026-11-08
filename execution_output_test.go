package jet

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type partialValueWriter struct {
	bytes.Buffer
	cause        error
	failed       bool
	afterFailure int
}

func (w *partialValueWriter) Write(b []byte) (int, error) {
	if w.failed {
		w.afterFailure++
	}
	n := len(b)
	if n > 2 {
		n = 2
	}
	w.Buffer.Write(b[:n])
	w.failed = w.cause != nil || n != len(b)
	return n, w.cause
}

func TestExecutionOutputFailurePaths(t *testing.T) {
	cause := errors.New("partial output failed")
	for _, source := range []string{
		"abcd", `{{ "abc<&" }}`, `{{ raw("abcd") }}`, `{{ safeHtml("abcd<") }}`,
		`{{ safeJs("abcd<") }}`, `{{ json("abcd") }}`, `{{ writeJson("abcd") }}`,
		`{{ try }}abcd{{ catch }}caught{{ end }}`, `{{ ignored("abcd") }}later`, `{{ renderer }}later`,
	} {
		t.Run(source, func(t *testing.T) {
			for _, err := range []error{cause, nil} {
				want := err
				if want == nil {
					want = io.ErrShortWrite
				}
				vars := VarMap{}.SetWriter("ignored", func(w io.Writer, b []byte) {
					defer func() { recover() }()
					w.Write(b)
					w.Write(b)
				}).Set("renderer", RendererFunc(func(r *Runtime) {
					defer func() { recover() }()
					r.Writer.Write([]byte("abcd"))
					r.Writer.Write([]byte("later"))
				}))
				w := &partialValueWriter{cause: err}
				got := runtimeTemplate(t, source).ExecuteContext(context.Background(), w, vars, nil, ExecutionOptions{MaxValueBytes: 10000, MaxOutputBytes: 10000, MaxSteps: 100000, MaxDepth: 100})
				if !errors.Is(got, want) || w.afterFailure != 0 {
					t.Fatalf("got %v, writes after failure %d; want %v", got, w.afterFailure, want)
				}
			}
		})
	}
}
