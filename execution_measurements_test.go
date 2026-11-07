package jet_test

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
)

// Latency sampling is separate from throughput measurements: reading the
// clock changes small-template costs. Retain at most 100,000 samples.
func BenchmarkExecutionWorkloadLatency(b *testing.B) {
	for _, f := range executionWorkloads(b) {
		for _, mode := range []string{"text", "html"} {
			b.Run(f.Name+"/"+mode, func(b *testing.B) {
				tpl, err := f.set(mode == "html").Parse("workload", f.Source)
				if err != nil {
					b.Fatal(err)
				}
				vars := f.vars()
				var out bytes.Buffer
				for i := 0; i < 100; i++ {
					out.Reset()
					if err := tpl.Execute(&out, vars, nil); err != nil {
						b.Fatal(err)
					}
				}
				stride := b.N / 100000
				if b.N%100000 != 0 {
					stride++
				}
				if stride < 1 {
					stride = 1
				}
				samples := make([]int64, 0, b.N/stride+1)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					out.Reset()
					if i%stride == 0 {
						start := time.Now()
						err = tpl.Execute(&out, vars, nil)
						samples = append(samples, time.Since(start).Nanoseconds())
					} else {
						err = tpl.Execute(&out, vars, nil)
					}
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
				b.ReportMetric(float64(samples[(len(samples)-1)/2]), "p50_ns")
				b.ReportMetric(float64(samples[(len(samples)-1)*99/100]), "p99_ns")
			})
		}
	}
}

func BenchmarkConcurrentExecutionWorkloads(b *testing.B) {
	workers := []int{1, 2, 8}
	maximum := runtime.GOMAXPROCS(0)
	if maximum != 1 && maximum != 2 && maximum != 8 {
		workers = append(workers, maximum)
	}
	for _, f := range executionWorkloads(b) {
		if f.Name != "header_interpolation" && f.Name != "body_8192" && f.Name != "nested_silent" && f.Name != "escape_expansion" {
			continue
		}
		for _, mode := range []string{"text", "html"} {
			for _, count := range workers {
				for _, shared := range []bool{true, false} {
					kind := "shared_template"
					if !shared {
						kind = "separate_sets"
					}
					b.Run(fmt.Sprintf("%s/%s/workers_%d/%s", f.Name, mode, count, kind), func(b *testing.B) {
						templates := make([]*jet.Template, count)
						for i := range templates {
							if shared && i > 0 {
								templates[i] = templates[0]
								continue
							}
							tpl, err := f.set(mode == "html").Parse("workload", f.Source)
							if err != nil {
								b.Fatal(err)
							}
							templates[i] = tpl
						}
						vars := f.vars()
						start := make(chan struct{})
						failures := make(chan error, count)
						var done sync.WaitGroup
						done.Add(count)
						b.ReportAllocs()
						b.ResetTimer()
						for worker := 0; worker < count; worker++ {
							go func(worker int) {
								defer done.Done()
								var out bytes.Buffer
								<-start
								for i := worker; i < b.N; i += count {
									out.Reset()
									if err := templates[worker].Execute(&out, vars, nil); err != nil {
										failures <- err
										return
									}
								}
							}(worker)
						}
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
