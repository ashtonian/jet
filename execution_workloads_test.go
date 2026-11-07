package jet_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/CloudyKit/jet/v6"
)

type executionWorkload struct {
	Name      string
	Source    string
	Variables map[string]interface{}
	Text      string
	HTML      string
	Templates map[string]string
}

func executionWorkloads(tb testing.TB) []executionWorkload {
	tb.Helper()
	data, err := os.ReadFile("testData/execution_workloads.json")
	if err != nil {
		tb.Fatal(err)
	}
	var fixtures []executionWorkload
	if err := json.Unmarshal(data, &fixtures); err != nil {
		tb.Fatal(err)
	}
	return fixtures
}

func (f executionWorkload) set(html bool) *jet.Set {
	loader := jet.NewInMemLoader()
	for name, source := range f.Templates {
		loader.Set(name, source)
	}
	if html {
		return jet.NewSet(loader)
	}
	return jet.NewSet(loader, jet.WithSafeWriter(nil))
}

func (f executionWorkload) vars() jet.VarMap {
	vars := make(jet.VarMap, len(f.Variables))
	for name, value := range f.Variables {
		vars.Set(name, value)
	}
	return vars
}

func TestExecutionWorkloads(t *testing.T) {
	for _, f := range executionWorkloads(t) {
		t.Run(f.Name, func(t *testing.T) {
			for _, html := range []bool{false, true} {
				tpl, err := f.set(html).Parse("workload", f.Source)
				if err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				if err := tpl.Execute(&out, f.vars(), nil); err != nil {
					t.Fatal(err)
				}
				want := f.Text
				if html {
					want = f.HTML
				}
				if out.String() != want {
					t.Fatalf("html=%t: got %q, want %q", html, out.String(), want)
				}
			}
		})
	}
}

func BenchmarkExecutionWorkloads(b *testing.B) {
	for _, f := range executionWorkloads(b) {
		for _, mode := range []string{"text", "html"} {
			b.Run(f.Name+"/"+mode, func(b *testing.B) {
				html := mode == "html"
				vars := f.vars()
				b.Run("compile", func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						if _, err := f.set(html).Parse("workload", f.Source); err != nil {
							b.Fatal(err)
						}
					}
				})
				b.Run("cached_execute", func(b *testing.B) {
					tpl, err := f.set(html).Parse("workload", f.Source)
					if err != nil {
						b.Fatal(err)
					}
					var out bytes.Buffer
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						out.Reset()
						if err := tpl.Execute(&out, vars, nil); err != nil {
							b.Fatal(err)
						}
					}
				})
				b.Run("parse_execute", func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						tpl, err := f.set(html).Parse("workload", f.Source)
						if err != nil {
							b.Fatal(err)
						}
						var out bytes.Buffer
						if err := tpl.Execute(&out, vars, nil); err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}
