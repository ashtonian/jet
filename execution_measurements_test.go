package jet_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
)

type executionMode struct {
	name   string
	live   bool
	newSet func(context.Context, executionWorkload, bool) (*jet.Set, error)
	lookup func(context.Context, *jet.Set) (*jet.Template, error)
	render func(context.Context, *jet.Template, jet.VarMap, interface{}, *bytes.Buffer) (string, error)
}

var executionModes = []executionMode{
	{
		name: "legacy_stream", newSet: legacyMeasurementSet, lookup: legacyMeasurementLookup,
		render: func(_ context.Context, t *jet.Template, v jet.VarMap, data interface{}, out *bytes.Buffer) (string, error) {
			return "", t.Execute(out, v, data)
		},
	},
	{
		name: "legacy_owned", newSet: legacyMeasurementSet, lookup: legacyMeasurementLookup,
		render: func(_ context.Context, t *jet.Template, v jet.VarMap, data interface{}, _ *bytes.Buffer) (string, error) {
			var out bytes.Buffer
			err := t.Execute(&out, v, data)
			return out.String(), err
		},
	},
}

func legacyMeasurementSet(_ context.Context, f executionWorkload, html bool) (*jet.Set, error) {
	return f.set(html), nil
}

func legacyMeasurementLookup(_ context.Context, set *jet.Set) (*jet.Template, error) {
	return set.GetTemplate("workload")
}

func (m executionMode) context() (context.Context, context.CancelFunc) {
	if m.live {
		return context.WithCancel(context.Background())
	}
	return context.Background(), func() {}
}

type measurementString string
type measurementMap map[measurementString]string
type measurementSlice []string

// Additional coverage never contributes to the fixed 15-fixture acceptance median.
func additionalExecutionWorkloads() []executionWorkload {
	items := make([]map[string]interface{}, 256)
	for i := range items {
		items[i] = map[string]interface{}{"id": i, "value": strings.Repeat("value", 8)}
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}
	meteredDiagnostic := `Jet Runtime Error ("/workload":1): identifier "missing" not available in current or parent scope, global, or default variables`
	restrictedDiagnostic := `Jet Runtime Error ("/workload":1): identifier "<string>" is not available`
	return []executionWorkload{
		{Name: "data_json_256", Source: "{{ json(.) }}", Data: items, Text: string(encoded), HTML: html.EscapeString(string(encoded))},
		{Name: "data_unused_256", Source: "ready", Data: items, Text: "ready", HTML: "ready"},
		{Name: "named_map", Source: "{{ .x }}", Data: measurementMap{"x": "named"}, Text: "named", HTML: "named"},
		{Name: "named_slice", Source: "{{ .[0] }}:{{ len(.) }}", Data: measurementSlice{"named"}, Text: "named:1", HTML: "named:1"},
		{Name: "array", Source: "{{ .[0] }}:{{ len(.) }}", Data: [2]string{"a", "b"}, Text: "a:2", HTML: "a:2"},
		{Name: "typed_nil", Source: "{{ len(.) }}", Data: map[string]string(nil), Text: "0", HTML: "0"},
		{Name: "builtin_alias", Source: "{{ fold := lower }}{{ fold(value) }}", Variables: map[string]interface{}{"value": "ALIAS"}, Text: "alias", HTML: "alias"},
		{Name: "catch_diagnostic", Source: "{{ try }}{{ missing }}{{ catch err }}{{ err }}{{ end }}", Text: `Jet Runtime Error ("/workload":1): identifier "missing" not available in current (map[]) or parent scope, global, or default variables`, HTML: `Jet Runtime Error (&#34;/workload&#34;:1): identifier &#34;missing&#34; not available in current (map[]) or parent scope, global, or default variables`, ModeOutput: map[string][2]string{
			"metered_background_stream": {meteredDiagnostic, html.EscapeString(meteredDiagnostic)},
			"metered_live_stream":       {meteredDiagnostic, html.EscapeString(meteredDiagnostic)},
			"restricted_owned":          {restrictedDiagnostic, html.EscapeString(restrictedDiagnostic)},
		}},
	}
}

func measurementWorkloads(tb testing.TB) []executionWorkload {
	return append(executionWorkloads(tb), additionalExecutionWorkloads()...)
}

func prepareMeasurement(tb testing.TB, m executionMode, ctx context.Context, f executionWorkload, html bool) (*jet.Set, *jet.Template) {
	tb.Helper()
	set, err := m.newSet(ctx, f, html)
	if err != nil {
		tb.Fatal(err)
	}
	tpl, err := m.lookup(ctx, set)
	if err != nil {
		tb.Fatal(err)
	}
	return set, tpl
}

func TestExecutionModeConformance(t *testing.T) {
	for _, m := range executionModes {
		for _, f := range measurementWorkloads(t) {
			for _, html := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/html_%t", m.name, f.Name, html), func(t *testing.T) {
					ctx, cancel := m.context()
					defer cancel()
					_, tpl := prepareMeasurement(t, m, ctx, f, html)
					var out bytes.Buffer
					got, err := m.render(ctx, tpl, f.vars(), f.Data, &out)
					if err != nil {
						t.Fatal(err)
					}
					if strings.HasSuffix(m.name, "_stream") {
						got = out.String()
					}
					want := f.Text
					if html {
						want = f.HTML
					}
					if output, ok := f.ModeOutput[m.name]; ok {
						want = output[0]
						if html {
							want = output[1]
						}
					}
					if got != want {
						t.Fatalf("got %q, want %q", got, want)
					}
				})
			}
		}
	}
}

func BenchmarkExecutionModes(b *testing.B) {
	for _, m := range executionModes {
		for _, f := range measurementWorkloads(b) {
			for _, format := range []string{"text", "html"} {
				b.Run(m.name+"/"+f.Name+"/"+format, func(b *testing.B) {
					ctx, cancel := m.context()
					defer cancel()
					_, tpl := prepareMeasurement(b, m, ctx, f, format == "html")
					vars := f.vars()
					var out bytes.Buffer
					for i := 0; i < 10; i++ {
						out.Reset()
						if _, err := m.render(ctx, tpl, vars, f.Data, &out); err != nil {
							b.Fatal(err)
						}
					}
					b.ReportAllocs()
					b.ResetTimer()
					var result string
					for i := 0; i < b.N; i++ {
						out.Reset()
						var err error
						result, err = m.render(ctx, tpl, vars, f.Data, &out)
						if err != nil {
							b.Fatal(err)
						}
					}
					runtime.KeepAlive(result)
				})
			}
		}
	}
}

func BenchmarkExecutionPhases(b *testing.B) {
	for _, m := range executionModes {
		for _, f := range executionWorkloads(b) {
			if f.Name != "header_interpolation" && f.Name != "include" && f.Name != "inheritance" {
				continue
			}
			for _, phase := range []string{"constructor", "cold_compile", "warm_lookup", "parse_render"} {
				b.Run(m.name+"/"+f.Name+"/"+phase, func(b *testing.B) {
					ctx, cancel := m.context()
					defer cancel()
					set, _ := prepareMeasurement(b, m, ctx, f, true)
					vars := f.vars()
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; {
						if phase == "cold_compile" {
							b.StopTimer()
							count := b.N - i
							if count > 128 {
								count = 128
							}
							sets := make([]*jet.Set, count)
							for j := range sets {
								var err error
								sets[j], err = m.newSet(ctx, f, true)
								if err != nil {
									b.Fatal(err)
								}
							}
							b.StartTimer()
							for _, cold := range sets {
								if _, err := m.lookup(ctx, cold); err != nil {
									b.Fatal(err)
								}
							}
							i += count
							continue
						}
						current := set
						if phase != "warm_lookup" {
							var err error
							current, err = m.newSet(ctx, f, true)
							if err != nil {
								b.Fatal(err)
							}
						}
						if phase != "constructor" {
							tpl, err := m.lookup(ctx, current)
							if err != nil {
								b.Fatal(err)
							}
							if phase == "parse_render" {
								var out bytes.Buffer
								if _, err := m.render(ctx, tpl, vars, f.Data, &out); err != nil {
									b.Fatal(err)
								}
							}
						}
						i++
					}
				})
			}
		}
	}
}

// Clock sampling stays separate from throughput, with bounded retained samples.
func BenchmarkExecutionModeLatency(b *testing.B) {
	for _, m := range executionModes {
		for _, f := range measurementWorkloads(b) {
			for _, format := range []string{"text", "html"} {
				b.Run(m.name+"/"+f.Name+"/"+format, func(b *testing.B) {
					ctx, cancel := m.context()
					defer cancel()
					_, tpl := prepareMeasurement(b, m, ctx, f, format == "html")
					vars := f.vars()
					stride := (b.N + 99999) / 100000
					samples := make([]int64, 0, (b.N+stride-1)/stride)
					var out bytes.Buffer
					var result string
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						out.Reset()
						start := time.Time{}
						if i%stride == 0 {
							start = time.Now()
						}
						var err error
						result, err = m.render(ctx, tpl, vars, f.Data, &out)
						if !start.IsZero() {
							samples = append(samples, time.Since(start).Nanoseconds())
						}
						if err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					runtime.KeepAlive(result)
					sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
					b.ReportMetric(float64(samples[(len(samples)-1)/2]), "p50_ns")
					b.ReportMetric(float64(samples[(len(samples)-1)*99/100]), "p99_ns")
				})
			}
		}
	}
}

func BenchmarkConcurrentExecutionModes(b *testing.B) {
	workers := []int{1, 2, 8}
	maximum := runtime.GOMAXPROCS(0)
	if maximum != 1 && maximum != 2 && maximum != 8 {
		workers = append(workers, maximum)
	}
	for _, m := range executionModes {
		for _, f := range measurementWorkloads(b) {
			if f.Name != "header_interpolation" && f.Name != "body_8192" && f.Name != "nested_silent" && f.Name != "escape_expansion" && f.Name != "data_json_256" {
				continue
			}
			for _, format := range []string{"text", "html"} {
				for _, count := range workers {
					for _, shared := range []bool{true, false} {
						b.Run(fmt.Sprintf("%s/%s/%s/workers_%d/shared_%t", m.name, f.Name, format, count, shared), func(b *testing.B) {
							templates := make([]*jet.Template, count)
							variables := make([]jet.VarMap, count)
							contexts := make([]context.Context, count)
							for i := range templates {
								ctx, cancel := m.context()
								defer cancel()
								contexts[i] = ctx
								variables[i] = f.vars()
								if shared && i > 0 {
									templates[i] = templates[0]
								} else {
									_, templates[i] = prepareMeasurement(b, m, ctx, f, format == "html")
								}
							}
							start := make(chan struct{})
							failures := make(chan error, count)
							var ready, done sync.WaitGroup
							ready.Add(count)
							done.Add(count)
							for worker := 0; worker < count; worker++ {
								go func(worker int) {
									defer done.Done()
									var out bytes.Buffer
									var result string
									ready.Done()
									<-start
									for i := worker; i < b.N; i += count {
										out.Reset()
										var err error
										result, err = m.render(contexts[worker], templates[worker], variables[worker], f.Data, &out)
										if err != nil {
											failures <- err
											return
										}
									}
									runtime.KeepAlive(result)
								}(worker)
							}
							ready.Wait()
							b.ReportAllocs()
							b.ResetTimer()
							close(start)
							done.Wait()
							b.StopTimer()
							close(failures)
							for err := range failures {
								b.Fatal(err)
							}
						})
					}
				}
			}
		}
	}
}

func TestExecutionRetentionMeasurements(t *testing.T) {
	if os.Getenv("JET_RETENTION_MEASUREMENTS") != "1" {
		t.Skip("set JET_RETENTION_MEASUREMENTS=1 for cold/large-then-small diagnostics")
	}
	for _, m := range executionModes {
		t.Run(m.name, func(t *testing.T) {
			ctx, cancel := m.context()
			defer cancel()
			f := executionWorkload{Source: "{{ value }}"}
			_, tpl := prepareMeasurement(t, m, ctx, f, false)
			large := jet.VarMap{}.Set("value", strings.Repeat("x", 256<<10))
			small := jet.VarMap{}.Set("value", "x")
			var out bytes.Buffer
			var result string
			render := func(vars jet.VarMap) {
				out.Reset()
				var err error
				result, err = m.render(ctx, tpl, vars, nil, &out)
				if err != nil {
					t.Fatal(err)
				}
			}
			observe := func(phase string) {
				var memory runtime.MemStats
				runtime.ReadMemStats(&memory)
				start := time.Now()
				render(small)
				t.Logf(`{"phase":%q,"render_ns":%d,"heap_alloc":%d,"heap_inuse":%d,"heap_sys":%d,"stream_capacity":%d}`, phase, time.Since(start).Nanoseconds(), memory.HeapAlloc, memory.HeapInuse, memory.HeapSys, out.Cap())
				if directory := os.Getenv("JET_RETENTION_PROFILE_DIR"); directory != "" {
					file, err := os.Create(filepath.Join(directory, phase+".pprof"))
					if err != nil {
						t.Fatal(err)
					}
					profileError := pprof.WriteHeapProfile(file)
					closeError := file.Close()
					if profileError != nil {
						t.Fatal(profileError)
					}
					if closeError != nil {
						t.Fatal(closeError)
					}
				}
			}
			runtime.GC()
			runtime.GC()
			observe("post_two_gc")
			for i := 0; i < 1000; i++ {
				render(small)
			}
			observe("warm_small")
			for i := 0; i < 16; i++ {
				render(large)
			}
			observe("after_large")
			for i := 0; i < 1000; i++ {
				render(small)
			}
			observe("large_then_small")
			runtime.GC()
			observe("large_then_small_after_gc")
			runtime.KeepAlive(result)
			runtime.KeepAlive(out)
			runtime.KeepAlive(large)
			runtime.KeepAlive(small)
			runtime.KeepAlive(tpl)
		})
	}
}

func BenchmarkExecutionHarnessControl(b *testing.B) {
	workers := []int{1, 2, 8}
	maximum := runtime.GOMAXPROCS(0)
	if maximum != 1 && maximum != 2 && maximum != 8 {
		workers = append(workers, maximum)
	}
	for _, count := range workers {
		b.Run(fmt.Sprintf("workers_%d", count), func(b *testing.B) {
			start := make(chan struct{})
			var ready, done sync.WaitGroup
			ready.Add(count)
			done.Add(count)
			for worker := 0; worker < count; worker++ {
				go func(worker int) {
					defer done.Done()
					value := uint64(worker + 1)
					ready.Done()
					<-start
					for i := worker; i < b.N; i += count {
						value = value*1099511628211 ^ uint64(i)
					}
					runtime.KeepAlive(value)
				}(worker)
			}
			ready.Wait()
			b.ResetTimer()
			close(start)
			done.Wait()
			b.StopTimer()
		})
	}
}
