package jet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCompilationSourceBoundaries(t *testing.T) {
	for _, named := range []bool{false, true} {
		for _, warm := range []bool{false, true} {
			for _, n := range []int{7, 8, 9} {
				t.Run(fmt.Sprintf("named=%t/warm=%t/bytes=%d", named, warm, n), func(t *testing.T) {
					source := strings.Repeat("x", n)
					loader := NewInMemLoader()
					loader.Set("s.jet", source)
					set := NewSet(loader)
					if warm {
						if _, err := set.GetTemplate("s.jet"); err != nil {
							t.Fatal(err)
						}
					}
					var tpl *Template
					var err error
					opts := ParseOptions{MaxSourceBytes: 8}
					if named {
						tpl, err = set.GetTemplateContext(context.Background(), "s.jet", opts)
					} else {
						tpl, err = set.ParseContext(context.Background(), "s.jet", source, opts)
					}
					if n > 8 {
						if !errors.Is(err, ErrParseSourceLimit) || tpl != nil {
							t.Fatalf("got template=%v err=%v", tpl, err)
						}
					} else if err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func compilationDiamond(development bool) (*Set, int64) {
	loader := NewInMemLoader()
	sources := map[string]string{
		"a.jet":      `{{import "b.jet"}}{{import "c.jet"}}{{yield shared()}}`,
		"b.jet":      `{{import "shared.jet"}}`,
		"c.jet":      `{{import "./shared.jet"}}`,
		"shared.jet": `{{block shared()}}ok{{end}}`,
	}
	total := int64(0)
	for name, source := range sources {
		loader.Set(name, source)
		total += int64(len(source))
	}
	return NewSet(loader, DevelopmentMode(development)), total
}

func TestCompilationGraphAdmission(t *testing.T) {
	for _, development := range []bool{false, true} {
		for _, warm := range []bool{false, true} {
			for _, kind := range []string{"total", "templates", "valid"} {
				t.Run(fmt.Sprintf("development=%t/warm=%t/%s", development, warm, kind), func(t *testing.T) {
					set, total := compilationDiamond(development)
					if warm {
						if _, err := set.GetTemplate("a.jet"); err != nil {
							t.Fatal(err)
						}
					}
					opts := ParseOptions{MaxTotalSourceBytes: total, MaxTemplates: 4}
					var want error
					if kind == "total" {
						opts.MaxTotalSourceBytes--
						want = ErrParseTotalSourceLimit
					}
					if kind == "templates" {
						opts.MaxTemplates--
						want = ErrParseTemplateLimit
					}
					tpl, err := set.GetTemplateContext(context.Background(), "./a", opts)
					if want != nil {
						if !errors.Is(err, want) || tpl != nil {
							t.Fatalf("got %v, want %v", err, want)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					var out bytes.Buffer
					if err = tpl.Execute(&out, nil, nil); err != nil || out.String() != "ok" {
						t.Fatalf("output %q: %v", out.String(), err)
					}
				})
			}
		}
	}
}

func TestCompilationNodeAndDepthCacheParity(t *testing.T) {
	for _, warm := range []bool{false, true} {
		for _, tc := range []struct {
			name, source string
			opts         ParseOptions
			want         error
		}{
			{"nodes", `{{1}}{{2}}{{3}}`, ParseOptions{MaxNodes: 3}, ErrParseNodeLimit},
			{"parentheses", "{{" + strings.Repeat("(", 100) + "1" + strings.Repeat(")", 100) + "}}", ParseOptions{MaxDepth: 16}, ErrParseDepthLimit},
			{"unary", "{{" + strings.Repeat("not ", 100) + "true}}", ParseOptions{MaxDepth: 16}, ErrParseDepthLimit},
			{"binary chain", "{{1" + strings.Repeat("+1", 100) + "}}", ParseOptions{MaxDepth: 16}, ErrParseDepthLimit},
			{"lists", strings.Repeat("{{if true}}", 100) + "x" + strings.Repeat("{{end}}", 100), ParseOptions{MaxDepth: 16}, ErrParseDepthLimit},
			{"else if", `{{if false}}` + strings.Repeat(`{{else if false}}`, 100) + `{{end}}`, ParseOptions{MaxDepth: 16}, ErrParseDepthLimit},
			{"calls", "{{" + strings.Repeat("len(", 100) + `"x"` + strings.Repeat(")", 100) + "}}", ParseOptions{MaxDepth: 16}, ErrParseDepthLimit},
		} {
			t.Run(fmt.Sprintf("warm=%t/%s", warm, tc.name), func(t *testing.T) {
				loader := NewInMemLoader()
				loader.Set("s.jet", tc.source)
				set := NewSet(loader)
				if warm {
					if _, err := set.GetTemplate("s.jet"); err != nil {
						t.Fatal(err)
					}
				}
				tpl, err := set.GetTemplateContext(context.Background(), "s.jet", tc.opts)
				if tpl != nil || !errors.Is(err, tc.want) {
					t.Fatalf("got %v, want %v", err, tc.want)
				}
			})
		}
	}
}

func TestCompilationOptionValidation(t *testing.T) {
	set := NewSet(NewInMemLoader())
	for _, opts := range []ParseOptions{{MaxSourceBytes: -1}, {MaxTotalSourceBytes: -1}, {MaxDepth: -1}} {
		if _, err := set.ParseContext(context.Background(), "s", "", opts); !errors.Is(err, ErrInvalidParseOptions) {
			t.Fatal(err)
		}
	}
	if _, err := set.ParseContext(nil, "s", "", ParseOptions{}); !errors.Is(err, ErrInvalidParseOptions) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := set.ParseContext(ctx, "s", "", ParseOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	opts := ParseOptions{MaxSourceBytes: math.MaxInt64, MaxTotalSourceBytes: math.MaxInt64, MaxNodes: math.MaxUint64, MaxSteps: math.MaxUint64, MaxDepth: int(^uint(0) >> 1), MaxTemplates: math.MaxUint64}
	if _, err := set.ParseContext(context.Background(), "s", "ok", opts); err != nil {
		t.Fatal(err)
	}
}

type compilationCheckpointContext struct {
	context.Context
	cancel       context.CancelFunc
	calls, after int
}

func (c *compilationCheckpointContext) Err() error {
	c.calls++
	if c.calls == c.after {
		c.cancel()
	}
	return c.Context.Err()
}

func TestCompilationScanCancellation(t *testing.T) {
	for _, source := range []string{strings.Repeat("x", 20000), "{*" + strings.Repeat("x", 20000) + "*}", "{{`" + strings.Repeat("x", 20000) + "`}}", "{{ " + strings.Repeat(" ", 20000) + "1}}"} {
		ctx, cancel := context.WithCancel(context.Background())
		checkpoint := &compilationCheckpointContext{Context: ctx, cancel: cancel, after: 12}
		tpl, err := NewSet(NewInMemLoader()).ParseContext(checkpoint, "s", source, ParseOptions{})
		cancel()
		if tpl != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	}
}

type compilationReadCloser struct {
	io.Reader
	close func() error
}

func (r compilationReadCloser) Close() error { return r.close() }

type compilationLoader struct{ open func() (io.ReadCloser, error) }

func (l compilationLoader) Exists(string) bool                 { return true }
func (l compilationLoader) Open(string) (io.ReadCloser, error) { return l.open() }

func TestCompilationReaderCleanup(t *testing.T) {
	failure := errors.New("reader failure")
	for _, kind := range []string{"source limit", "total limit", "syntax", "read", "close", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			closed := false
			opts := ParseOptions{}
			var want error
			var reader io.Reader = strings.NewReader("{{if}}")
			switch kind {
			case "source limit":
				opts.MaxSourceBytes = 2
				want = ErrParseSourceLimit
			case "total limit":
				opts.MaxTotalSourceBytes = 2
				want = ErrParseTotalSourceLimit
			case "read":
				reader = compilationErrorReader{failure}
				want = failure
			case "close":
				reader = strings.NewReader("ok")
				want = failure
			case "cancel":
				reader = compilationCancelReader{cancel}
				want = context.Canceled
			}
			loader := compilationLoader{open: func() (io.ReadCloser, error) {
				return compilationReadCloser{Reader: reader, close: func() error {
					closed = true
					if kind == "close" {
						return failure
					}
					return nil
				}}, nil
			}}
			tpl, err := NewSet(loader).GetTemplateContext(ctx, "s", opts)
			if err == nil || tpl != nil || !closed {
				t.Fatalf("template=%v err=%v closed=%t", tpl, err, closed)
			}
			if want != nil && !errors.Is(err, want) {
				t.Fatalf("got %v, want %v", err, want)
			}
		})
	}
}

type compilationErrorReader struct{ err error }

func (r compilationErrorReader) Read([]byte) (int, error) { return 0, r.err }

type compilationCancelReader struct{ cancel context.CancelFunc }

func (r compilationCancelReader) Read(p []byte) (int, error) { r.cancel(); p[0] = 'x'; return 1, nil }

func TestCompilationBlockedLoaderOwnership(t *testing.T) {
	entered, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	loader := compilationLoader{open: func() (io.ReadCloser, error) {
		close(entered)
		<-release
		return compilationReadCloser{Reader: strings.NewReader("ok"), close: func() error { close(closed); return nil }}, nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := NewSet(loader).GetTemplateContext(ctx, "s", ParseOptions{}); result <- err }()
	<-entered
	cancel()
	close(release)
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("returned before closing owned reader")
	}
}

func TestCompilationStepLimitDevelopmentDiamond(t *testing.T) {
	loader := NewInMemLoader()
	for i := 0; i < 12; i++ {
		loader.Set(fmt.Sprintf("%d.jet", i), fmt.Sprintf(`{{import "%d.jet"}}{{import "%d.jet"}}`, i+1, i+1))
	}
	loader.Set("12.jet", "ok")
	_, err := NewSet(loader, InDevelopmentMode()).GetTemplateContext(context.Background(), "0.jet", ParseOptions{MaxSteps: 2000, MaxTemplates: 13})
	if !errors.Is(err, ErrParseStepLimit) {
		t.Fatal(err)
	}
}

func TestCompilationCacheIsolation(t *testing.T) {
	shared := &cache{}
	first := NewInMemLoader()
	first.Set("s.jet", "first")
	second := NewInMemLoader()
	second.Set("s.jet", "second")
	a, b := NewSet(first, WithCache(shared)), NewSet(second, WithCache(shared))
	if _, err := a.GetTemplate("s.jet"); err != nil {
		t.Fatal(err)
	}
	tpl, err := b.GetTemplateContext(context.Background(), "s.jet", ParseOptions{MaxSourceBytes: 6})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = tpl.Execute(&out, nil, nil); err != nil || out.String() != "second" {
		t.Fatalf("%q: %v", out.String(), err)
	}
}

func TestCompilationConcurrentPolicies(t *testing.T) {
	set, _ := compilationDiamond(false)
	if _, err := set.GetTemplate("a.jet"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(strict bool) {
			defer wg.Done()
			opts := ParseOptions{}
			if strict {
				opts.MaxTemplates = 1
			}
			tpl, err := set.GetTemplateContext(context.Background(), "a.jet", opts)
			if strict {
				if !errors.Is(err, ErrParseTemplateLimit) {
					t.Errorf("strict: %v", err)
				}
			} else if err != nil || tpl == nil {
				t.Errorf("loose: %v", err)
			}
		}(i%2 == 0)
	}
	wg.Wait()
}

func TestCompilationDepthSubprocess(t *testing.T) {
	for _, kind := range []string{"expression", "list", "dependency"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCompilationDepthChild$")
			cmd.Env = append(os.Environ(), "JET_COMPILE_DEPTH_CHILD="+kind, "GOTRACEBACK=single")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bounded child: %v ctx=%v output=%.1000s", err, ctx.Err(), out)
			}
		})
	}
}
func TestCompilationDepthChild(t *testing.T) {
	kind := os.Getenv("JET_COMPILE_DEPTH_CHILD")
	if kind == "" {
		t.Skip("subprocess only")
	}
	debug.SetMaxStack(256 << 10)
	loader := NewInMemLoader()
	set := NewSet(loader)
	opts := ParseOptions{MaxDepth: 16}
	var err error
	switch kind {
	case "expression":
		_, err = set.ParseContext(context.Background(), "deep.jet", "{{"+strings.Repeat("(", 2000)+"1"+strings.Repeat(")", 2000)+"}}", opts)
	case "list":
		_, err = set.ParseContext(context.Background(), "deep.jet", strings.Repeat("{{if true}}", 2000)+"ok"+strings.Repeat("{{end}}", 2000), opts)
	case "dependency":
		for i := 0; i < 2000; i++ {
			loader.Set(fmt.Sprintf("%d.jet", i), fmt.Sprintf(`{{import "%d.jet"}}`, i+1))
		}
		loader.Set("2000.jet", "ok")
		_, err = set.GetTemplateContext(context.Background(), "0.jet", opts)
	default:
		t.Fatal(kind)
	}
	if !errors.Is(err, ErrParseDepthLimit) {
		t.Fatalf("got %v", err)
	}
}

func TestCompilationLegacyIncludeIfExistsSyntaxError(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("bad.jet", "{{if}}")
	set := NewSet(loader)
	tpl, err := set.Parse("root.jet", `{{includeIfExists("bad.jet")}}`)
	if err != nil {
		t.Fatal(err)
	}
	if err = tpl.Execute(io.Discard, nil, nil); err == nil {
		t.Fatal("includeIfExists suppressed a syntax error")
	}
}

func TestCompilationExactNodeAndDepthBoundaries(t *testing.T) {
	for _, warm := range []bool{false, true} {
		for _, limit := range []int{1, 2, 3} {
			loader := NewInMemLoader()
			loader.Set("literal.jet", "text")
			set := NewSet(loader)
			if warm {
				if _, err := set.GetTemplate("literal.jet"); err != nil {
					t.Fatal(err)
				}
			}
			_, err := set.GetTemplateContext(context.Background(), "literal.jet", ParseOptions{MaxNodes: uint64(limit), MaxDepth: 1})
			if limit < 2 {
				if !errors.Is(err, ErrParseNodeLimit) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		}
		for _, limit := range []int{2, 3, 4} {
			loader := NewInMemLoader()
			loader.Set("action.jet", "{{1}}")
			set := NewSet(loader)
			if warm {
				if _, err := set.GetTemplate("action.jet"); err != nil {
					t.Fatal(err)
				}
			}
			_, err := set.GetTemplateContext(context.Background(), "action.jet", ParseOptions{MaxDepth: limit})
			if limit < 3 {
				if !errors.Is(err, ErrParseDepthLimit) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCompilationDiagnosticsBounded(t *testing.T) {
	_, err := NewSet(NewInMemLoader()).ParseContext(context.Background(), "s", "{{"+strings.Repeat("9", 20000)+"}}", ParseOptions{})
	if err == nil || len(err.Error()) > 512 {
		t.Fatalf("diagnostic length=%d", len(fmt.Sprint(err)))
	}
}
