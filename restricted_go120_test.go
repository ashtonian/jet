//go:build go1.20
// +build go1.20

package jet

import (
	"context"
	"errors"
	"testing"
)

func TestRestrictedInputCancellationCause(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": "ok"}, restrictedTestOptions())
	cause := errors.New("caller cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	result, err := tpl.RenderRestrictedContext(ctx, nil, nil)
	if result != "" || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatalf("%q %v", result, err)
	}
}
