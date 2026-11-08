package jet_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/CloudyKit/jet/v6"
)

func measurementExecutionOptions() jet.ExecutionOptions {
	return jet.ExecutionOptions{
		MaxSteps: 1000000, MaxDepth: 64, MaxValueBytes: 8 << 20, MaxOutputBytes: 1 << 20,
		ParseOptions: jet.ParseOptions{MaxSourceBytes: 128 << 10, MaxTotalSourceBytes: 1 << 20,
			MaxNodes: 20000, MaxSteps: 1000000, MaxDepth: 64, MaxTemplates: 64},
	}
}

func TestExecutionMeasurementBounds(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		for _, tc := range []struct {
			name, value string
			html        bool
			wantLimit   bool
		}{
			{"exact_text", strings.Repeat("x", 1024), false, false},
			{"over_text", strings.Repeat("x", 1025), false, true},
			{"near_escaped", strings.Repeat(`"`, 204), true, false},
			{"over_escaped", strings.Repeat(`"`, 205), true, true},
		} {
			t.Run(fmt.Sprintf("restricted_%t/%s", restricted, tc.name), func(t *testing.T) {
				opts := measurementExecutionOptions()
				opts.MaxOutputBytes = 1024
				f := executionWorkload{Source: "{{ value }}"}
				set := f.set(tc.html)
				if restricted {
					var err error
					set, err = jet.NewRestrictedSet(context.Background(), map[string]string{"workload": f.Source}, jet.RestrictedOptions{
						Text: !tc.html, Execution: opts,
						Input: jet.InputOptions{MaxBytes: 1 << 20, MaxItems: 20000, MaxSteps: 1000000, MaxDepth: 64},
					})
					if err != nil {
						t.Fatal(err)
					}
				}
				tpl, err := set.GetTemplate("workload")
				if err != nil {
					t.Fatal(err)
				}
				vars := jet.VarMap{}.Set("value", tc.value)
				var out bytes.Buffer
				got := ""
				if restricted {
					got, err = tpl.RenderRestrictedContext(context.Background(), vars, nil)
				} else {
					err = tpl.ExecuteContext(context.Background(), &out, vars, nil, opts)
					got = out.String()
				}
				if tc.wantLimit {
					if !errors.Is(err, jet.ErrOutputLimit) || (restricted && got != "") {
						t.Fatalf("result length %d, error %v", len(got), err)
					}
					return
				}
				want := tc.value
				if tc.html {
					want = strings.Repeat("&#34;", len(tc.value))
				}
				if err != nil || got != want {
					t.Fatalf("result length %d, error %v", len(got), err)
				}
			})
		}
	}
}

func init() {
	for _, live := range []bool{false, true} {
		name := "metered_background_stream"
		if live {
			name = "metered_live_stream"
		}
		executionModes = append(executionModes, executionMode{
			name: name, live: live, newSet: legacyMeasurementSet,
			lookup: func(ctx context.Context, set *jet.Set) (*jet.Template, error) {
				return set.GetTemplateContext(ctx, "workload", measurementExecutionOptions().ParseOptions)
			},
			render: func(ctx context.Context, t *jet.Template, vars jet.VarMap, data interface{}, out *bytes.Buffer) (string, error) {
				return "", t.ExecuteContext(ctx, out, vars, data, measurementExecutionOptions())
			},
		})
	}
	executionModes = append(executionModes, executionMode{
		name: "restricted_owned", live: true,
		newSet: func(ctx context.Context, f executionWorkload, html bool) (*jet.Set, error) {
			sources := map[string]string{"workload": f.Source}
			for name, source := range f.Templates {
				sources[name] = source
			}
			opts := jet.RestrictedOptions{Text: !html, Execution: measurementExecutionOptions(),
				Input: jet.InputOptions{MaxBytes: 1 << 20, MaxItems: 20000, MaxSteps: 1000000, MaxDepth: 64}}
			return jet.NewRestrictedSet(ctx, sources, opts)
		},
		lookup: func(ctx context.Context, set *jet.Set) (*jet.Template, error) {
			return set.GetTemplateContext(ctx, "workload", measurementExecutionOptions().ParseOptions)
		},
		render: func(ctx context.Context, t *jet.Template, vars jet.VarMap, data interface{}, _ *bytes.Buffer) (string, error) {
			return t.RenderRestrictedContext(ctx, vars, data)
		},
	})
}
