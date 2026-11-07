package jet

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// templateBundle is sealed: caller-provided Loader implementations cannot acquire
// this identity. Sources and namespace are immutable after admission.
type templateBundle struct{ sources map[string]string }

func admitTemplateBundle(ctx context.Context, sources map[string]string, opts ParseOptions) (bundle *templateBundle, err error) {
	if opts.MaxSourceBytes <= 0 || opts.MaxTotalSourceBytes <= 0 || opts.MaxTemplates == 0 {
		return nil, fmt.Errorf("%w: positive bundle source/total/template limits required", ErrInvalidParseOptions)
	}
	a, err := newParseAdmission(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			bundle = nil
		}
	}()
	defer a.recover(&err)
	if uint64(len(sources)) > opts.MaxTemplates {
		a.limit("templates", opts.MaxTemplates)
	}
	admitted := make(map[string]string, len(sources))
	for name, source := range sources {
		// Bound normalization and key storage as well as the template payloads.
		if len(name) > 1024 {
			return nil, fmt.Errorf("%w: template names must fit in 1024 bytes", ErrCapabilityDenied)
		}
		canonical, err := normalizeParseName(name)
		if err != nil {
			return nil, err
		}
		if _, ok := admitted[canonical]; ok {
			return nil, fmt.Errorf("%w: duplicate canonical template name %s", ErrCapabilityDenied, parseDiagnostic(canonical))
		}
		a.source(canonical, int64(len(source)))
		admitted[canonical] = source
	}
	a.check()
	return &templateBundle{sources: admitted}, nil
}

func (b *templateBundle) Exists(name string) bool { _, ok := b.sources[name]; return ok }
func (b *templateBundle) Open(name string) (io.ReadCloser, error) {
	source, ok := b.sources[name]
	if !ok {
		return nil, fmt.Errorf("%w: template name %s is outside the admitted bundle", ErrCapabilityDenied, parseDiagnostic(name))
	}
	return io.NopCloser(strings.NewReader(source)), nil
}
