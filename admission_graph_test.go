package jet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestCompilationSharedDiamondWork(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("left0.jet", "left")
	loader.Set("right0.jet", "right")
	const levels = 12
	for level := 1; level <= levels; level++ {
		source := fmt.Sprintf(`{{ import "left%d.jet" }}{{ import "right%d.jet" }}`, level-1, level-1)
		loader.Set(fmt.Sprintf("left%d.jet", level), source)
		loader.Set(fmt.Sprintf("right%d.jet", level), source)
	}
	set := NewSet(loader)
	root := fmt.Sprintf("left%d.jet", levels)
	if _, err := set.GetTemplate(root); err != nil {
		t.Fatal(err)
	}
	// The stored graph is small even though it has exponentially many paths.
	if _, err := set.GetTemplateContext(context.Background(), root, ParseOptions{MaxSteps: 2000, MaxTemplates: 2*levels + 1, MaxNodes: 10000, MaxDepth: 64}); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []ParseOptions{{MaxTemplates: 2 * levels}, {MaxNodes: 1}, {MaxDepth: levels}} {
		if _, err := set.GetTemplateContext(context.Background(), root, opts); err == nil {
			t.Fatalf("accepted tighter graph budget: %+v", opts)
		}
	}
}

func TestCompilationDepthAcrossPreviouslyAdmittedPath(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("leaf.jet", "leaf")
	loader.Set("middle.jet", `{{ import "leaf.jet" }}`)
	loader.Set("root.jet", `{{ import "leaf.jet" }}{{ import "middle.jet" }}`)
	set := NewSet(loader)
	if _, err := set.GetTemplate("root.jet"); err != nil {
		t.Fatal(err)
	}
	if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxDepth: 2}); !errors.Is(err, ErrParseDepthLimit) {
		t.Fatal(err)
	}
	if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxDepth: 64}); err != nil {
		t.Fatal(err)
	}
}

func TestCompilationCachedGraphInsideActiveParse(t *testing.T) {
	loader := NewInMemLoader()
	loader.Set("a.jet", "a")
	loader.Set("b.jet", `{{ import "a.jet" }}`)
	set := NewSet(loader)
	if _, err := set.GetTemplate("b.jet"); err != nil {
		t.Fatal(err)
	}
	// A compiled b is acyclic by itself, but its a dependency conflicts with
	// the canonical name currently being parsed. Unlimited calls still reject it.
	if _, err := set.Parse("a.jet", `{{ import "b.jet" }}`); !errors.Is(err, ErrTemplateCycle) {
		t.Fatal(err)
	}
	if _, err := set.ParseContext(context.Background(), "a.jet", `{{ import "b.jet" }}`, ParseOptions{}); !errors.Is(err, ErrTemplateCycle) {
		t.Fatal(err)
	}
}

func TestCompilationGraphSpillAndRepeatedVersions(t *testing.T) {
	loader := NewInMemLoader()
	var source strings.Builder
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("child%d.jet", i)
		loader.Set(name, "child")
		fmt.Fprintf(&source, `{{ import %q }}`, name)
	}
	loader.Set("root.jet", source.String())
	for _, dev := range []bool{false, true} {
		set := NewSet(loader, DevelopmentMode(dev))
		if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxTemplates: 21, MaxSteps: 100000, MaxNodes: 10000}); err != nil {
			t.Fatal(err)
		}
		if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxTemplates: 20, MaxSteps: 100000}); !errors.Is(err, ErrParseTemplateLimit) {
			t.Fatal(err)
		}
	}
}

type admissionEvictingCache struct {
	cache
	omit bool
}

func (c *admissionEvictingCache) Get(name string) *Template {
	if c.omit && name == "/a.jet" {
		return nil
	}
	return c.cache.Get(name)
}
func TestCompilationCachedSubgraphAgainstActiveVersion(t *testing.T) {
	for _, contextual := range []bool{false, true} {
		loader := NewInMemLoader()
		loader.Set("a.jet", "old")
		loader.Set("b.jet", `{{ import "a.jet" }}`)
		c := &admissionEvictingCache{}
		set := NewSet(loader, WithCache(c))
		if _, err := set.GetTemplate("b.jet"); err != nil {
			t.Fatal(err)
		}
		c.omit = true
		loader.Set("a.jet", `{{ import "b.jet" }}`)
		loader.Set("root.jet", `{{ import "b.jet" }}{{ import "a.jet" }}`)
		var err error
		if contextual {
			_, err = set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxSteps: 10000})
		} else {
			_, err = set.GetTemplate("root.jet")
		}
		if !errors.Is(err, ErrTemplateCycle) {
			t.Fatalf("context=%t: %v", contextual, err)
		}
	}
}

func TestCompilationClearsRuntimeScratch(t *testing.T) {
	loader := NewInMemLoader()
	var source strings.Builder
	for i := 0; i < 24; i++ {
		name := fmt.Sprintf("part%d.jet", i)
		loader.Set(name, "part")
		fmt.Fprintf(&source, `{{ import %q }}`, name)
	}
	source.WriteString(`{{ capture() }}`)
	loader.Set("root.jet", source.String())
	tpl, err := NewSet(loader).GetTemplate("root.jet")
	if err != nil {
		t.Fatal(err)
	}
	var retained *Runtime
	vars := VarMap{}.SetFunc("capture", func(a Arguments) reflect.Value { retained = a.Runtime(); return reflect.Value{} })
	if err := tpl.ExecuteContext(context.Background(), ioutil.Discard, vars, nil, ExecutionOptions{MaxSteps: 100000, ParseOptions: ParseOptions{MaxTemplates: 25, MaxNodes: 10000, MaxSteps: 100000}}); err != nil {
		t.Fatal(err)
	}
	if retained == nil || !reflect.ValueOf(retained.execution).IsZero() {
		t.Fatal("runtime retained operation state")
	}
}
func TestCompilationColdScratchAfterGC(t *testing.T) {
	loader := NewInMemLoader()
	var source strings.Builder
	for i := 0; i < 64; i++ {
		name := fmt.Sprintf("part%d.jet", i)
		loader.Set(name, "part")
		fmt.Fprintf(&source, `{{ import %q }}`, name)
	}
	loader.Set("root.jet", source.String())
	set := NewSet(loader)
	if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxTemplates: 65}); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.GC()
	if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxTemplates: 64}); !errors.Is(err, ErrParseTemplateLimit) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := set.GetTemplateContext(ctx, "root.jet", ParseOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := set.GetTemplateContext(context.Background(), "root.jet", ParseOptions{MaxTemplates: 65}); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionCompilationSourceVersions(t *testing.T) {
	source := strings.Repeat(`{{ include "v.jet" }}`, 6)
	versions := []string{strings.Repeat("a", 8), strings.Repeat("b", 512), "tiny", strings.Repeat("d", 32), strings.Repeat("e", 256), "ok"}
	for _, tight := range []bool{false, true} {
		opened := 0
		loader := compilationLoader{open: func() (io.ReadCloser, error) {
			if opened >= len(versions) {
				return nil, fmt.Errorf("unexpected extra load")
			}
			payload := versions[opened]
			opened++
			return io.NopCloser(strings.NewReader(payload)), nil
		}}
		tpl, err := NewSet(loader, InDevelopmentMode(), WithSafeWriter(nil)).Parse("main.jet", source)
		if err != nil {
			t.Fatal(err)
		}
		total := int64(len(source) + 512)
		if tight {
			total--
		}
		var result bytes.Buffer
		err = tpl.ExecuteContext(context.Background(), &result, nil, nil, ExecutionOptions{MaxSteps: 100000, MaxOutputBytes: 10000, ParseOptions: ParseOptions{MaxSourceBytes: 1024, MaxTotalSourceBytes: total, MaxNodes: 1000, MaxSteps: 100000, MaxTemplates: 2}})
		if tight {
			if !errors.Is(err, ErrParseTotalSourceLimit) {
				t.Fatal(err)
			}
		} else if err != nil || result.String() != strings.Join(versions, "") || opened != 6 {
			t.Fatalf("err=%v opens=%d bytes=%d", err, opened, result.Len())
		}
	}
}
