package jet

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type boundaryString string
type boundaryArray [2]int

func renderBoundary(t *testing.T, mode, source string, vars VarMap, data interface{}) (string, error) {
	t.Helper()
	if mode == "restricted" {
		tpl := restrictedTemplate(t, map[string]string{"main.jet": source}, restrictedTestOptions())
		return tpl.RenderRestrictedContext(context.Background(), vars, data)
	}
	tpl := runtimeTemplate(t, source)
	var out bytes.Buffer
	var err error
	if mode == "legacy" {
		err = tpl.Execute(&out, vars, data)
	} else {
		err = tpl.ExecuteContext(context.Background(), &out, vars, data, restrictedTestOptions().Execution)
	}
	return out.String(), err
}

func TestTemplateReflectionBoundary(t *testing.T) {
	for _, mode := range []string{"legacy", "metered", "restricted"} {
		for _, tc := range []struct {
			name, source string
			data         interface{}
			vars         VarMap
			want         string
			wantError    bool
		}{
			{"named key", `{{ .x }}`, map[boundaryString]string{"x": "ok"}, nil, "ok", false},
			{"named index", `{{ .[key] }}`, map[string]string{"x": "ok"}, VarMap{}.Set("key", boundaryString("x")), "ok", false},
			{"assignment type", `{{ .x = 1 }}`, map[string]string{"x": "ok"}, nil, "", true},
			{"nil map assignment", `{{ .x = 1 }}`, map[string]interface{}(nil), nil, "", true},
			{"named key assignment", `{{ .x = value }}{{ .x }}`, map[boundaryString]string{"x": "old"}, VarMap{}.Set("value", "new"), "new", false},
			{"invalid assignment", `{{ .x = nil }}`, map[string]interface{}{"x": "old"}, nil, "", true},
			{"array slice", `{{ .[:] }}`, [2]int{1, 2}, nil, "[1 2]", false},
			{"named array slice", `{{ .[1:] }}`, boundaryArray{1, 2}, nil, "[2]", false},
			{"nil slice", `{{ .[:] }}`, []int(nil), nil, "[]", false},
			{"nil slice base", `{{ .[:] }}`, nil, nil, "", true},
			{"invalid slice base", `{{ .[:] }}`, 3, nil, "", true},
			{"nil index", `{{ .[nil] }}`, map[string]string{"x": "ok"}, nil, "", true},
			{"nil map lookup", `{{ .["x"] }}`, map[string]string(nil), nil, "", false},
			{"odd map", `{{ map("x") }}`, nil, nil, "", true},
			{"nil map value", `{{ map("x", nil) }}`, nil, nil, "", true},
			{"nil slice value", `{{ slice(nil) }}`, nil, nil, "", true},
			{"nil array value", `{{ array(nil) }}`, nil, nil, "", true},
			{"typed nil map value", `{{ len(map("x", value)) }}`, nil, VarMap{}.Set("value", []int(nil)), "1", false},
			{"typed nil slice value", `{{ len(slice(value)) }}`, nil, VarMap{}.Set("value", []int(nil)), "1", false},
			{"typed nil array value", `{{ len(array(value)) }}`, nil, VarMap{}.Set("value", []int(nil)), "1", false},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("template input escaped as %T: %v", p, p)
					}
				}()
				out, err := renderBoundary(t, mode, tc.source, tc.vars, tc.data)
				if tc.wantError {
					if err == nil || errors.Is(err, ErrCapabilityDenied) || errors.Is(err, ErrValueLimit) || errors.Is(err, ErrStepLimit) {
						t.Fatalf("want ordinary error, got %q, %v", out, err)
					}
					if mode == "restricted" && out != "" {
						t.Fatalf("failed render returned %q", out)
					}
				} else if err != nil || out != tc.want {
					t.Fatalf("got %q, %v; want %q", out, err, tc.want)
				}
			})
		}
	}
}

func TestRestrictedCatchDiagnostic(t *testing.T) {
	for _, source := range []string{
		`{{ try }}{{ missing }}{{ catch err }}{{ err }}{{ end }}`,
		`{{ try }}{{ missing }}{{ catch err }}{{ m := map("message", err) }}{{ m.message }}{{ end }}`,
		`{{ try }}{{ map("x") }}{{ catch err }}{{ err }}{{ end }}`,
	} {
		t.Run(source, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("catch escaped as %T: %v", p, p)
				}
			}()
			out, err := renderBoundary(t, "restricted", source, nil, nil)
			if err != nil || out == "" || len(out) > 512 {
				t.Fatalf("got %q, %v", out, err)
			}
		})
	}
}
