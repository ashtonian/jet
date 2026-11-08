package jet

import "testing"

func TestRangeDiscardAssignmentBoundary(t *testing.T) {
	for _, mode := range []string{"legacy", "metered", "restricted"} {
		for _, tc := range []struct {
			name, source string
			data         interface{}
			want         string
		}{
			{"existing integers", `{{ n := 0 }}{{ range _,n = ints(0,2) }}{{ n }}{{ end }}`, nil, "01"},
			{"declared integers", `{{ range _,n := ints(0,2) }}{{ n }}{{ end }}`, nil, "01"},
			{"existing slice", `{{ n := 0 }}{{ range _,n = . }}{{ n }}{{ end }}`, []int{1, 2}, "12"},
			{"existing map", `{{ value := "" }}{{ range _,value = . }}{{ value }}{{ end }}`, map[string]string{"key": "value"}, "value"},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("discard assignment escaped as %T: %v", p, p)
					}
				}()
				out, err := renderBoundary(t, mode, tc.source, nil, tc.data)
				if err != nil || out != tc.want {
					t.Fatalf("got %q, %v; want %q", out, err, tc.want)
				}
			})
		}
	}
}
