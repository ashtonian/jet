//go:build go1.20
// +build go1.20

package jet

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestExecutionCompilationCancellationCause(t *testing.T) {
	set, _ := compilationDiamond(false)
	tpl, err := set.GetTemplate("a.jet")
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("compilation cancelled by caller")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	checkpoint := &compilationCheckpointContext{Context: ctx, cancel: func() { cancel(cause) }, after: 12}
	err = tpl.ExecuteContext(checkpoint, io.Discard, nil, nil, ExecutionOptions{})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
}
