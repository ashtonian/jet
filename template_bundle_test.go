package jet

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestTemplateBundleSnapshotAndComposition(t *testing.T) {
	sources := map[string]string{"views/a.jet": `{{extends "base.jet"}}{{block body()}}ok{{end}}`, "views/base.jet": `[{{yield body()}}]`}
	bundle, err := admitTemplateBundle(context.Background(), sources, ParseOptions{MaxSourceBytes: 100, MaxTotalSourceBytes: 200, MaxTemplates: 2})
	if err != nil {
		t.Fatal(err)
	}
	sources["views/a.jet"] = "changed"
	delete(sources, "views/base.jet")
	tpl, err := NewSet(bundle).GetTemplate("views/a.jet")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = tpl.Execute(&out, nil, nil); err != nil || out.String() != "[ok]" {
		t.Fatalf("%v: %q", err, out.String())
	}
	if _, err = bundle.Open("/foreign.jet"); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
}

func TestTemplateBundleAdmission(t *testing.T) {
	for _, tc := range []struct {
		sources map[string]string
		opts    ParseOptions
		want    error
	}{
		{map[string]string{"a": "one"}, ParseOptions{MaxSourceBytes: 2, MaxTotalSourceBytes: 10, MaxTemplates: 2}, ErrParseSourceLimit},
		{map[string]string{"a": "one", "b": "two"}, ParseOptions{MaxSourceBytes: 3, MaxTotalSourceBytes: 5, MaxTemplates: 2}, ErrParseTotalSourceLimit},
		{map[string]string{"a": "one", "b": "two"}, ParseOptions{MaxSourceBytes: 3, MaxTotalSourceBytes: 6, MaxTemplates: 1}, ErrParseTemplateLimit},
		{map[string]string{"a": "one", "./a": "two"}, ParseOptions{MaxSourceBytes: 3, MaxTotalSourceBytes: 6, MaxTemplates: 2}, ErrCapabilityDenied},
	} {
		if bundle, err := admitTemplateBundle(context.Background(), tc.sources, tc.opts); bundle != nil || !errors.Is(err, tc.want) {
			t.Fatalf("got %v, want %v", err, tc.want)
		}
	}
}
