package jet_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CloudyKit/jet/v6"
)

type measurementCancellationContext struct {
	context.Context
	calls     int64
	threshold int64
	ready     chan struct{}
	cancel    context.CancelFunc
	phase     string
}

func (c *measurementCancellationContext) Err() error {
	count := atomic.AddInt64(&c.calls, 1)
	if c.phase != "" && c.threshold == 0 {
		var pcs [32]uintptr
		n := runtime.Callers(2, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		previous := ""
		for {
			frame, more := frames.Next()
			if strings.Contains(frame.Function, c.phase) && (c.phase != "builtinCase" || strings.Contains(previous, ".checkpoint")) {
				c.threshold = count
				break
			}
			previous = frame.Function
			if !more {
				break
			}
		}
	}
	if count == c.threshold {
		if c.cancel != nil {
			c.cancel()
		} else {
			close(c.ready)
		}
	}
	return c.Context.Err()
}

type cancellationOperation struct {
	name  string
	phase string
	run   func(context.Context) error
}

func cancellationOperations(t *testing.T) []cancellationOperation {
	loop := executionWorkload{Source: "{{ range ints(0, 100000) }}{{ end }}"}
	scan := executionWorkload{Source: "{{ range ints(0, 10) }}{{ lower(value) }}{{ end }}", Variables: map[string]interface{}{"value": strings.Repeat("AbCd", 16384)}}
	_, loopTemplate := prepareMeasurement(t, executionModes[0], context.Background(), loop, false)
	_, scanTemplate := prepareMeasurement(t, executionModes[0], context.Background(), scan, false)
	vars := scan.vars()
	opts := measurementExecutionOptions()
	input := executionWorkload{Source: "ready", Data: make([]string, 8192)}
	mode := executionModes[len(executionModes)-1]
	_, inputTemplate := prepareMeasurement(t, mode, context.Background(), input, false)
	source := strings.Repeat("literal ", 12000) + "{{ value }}"
	return []cancellationOperation{
		{"interpreter_loop", ".advanceRanger", func(ctx context.Context) error {
			return loopTemplate.ExecuteContext(ctx, ioutil.Discard, nil, nil, opts)
		}},
		{"prepaid_builtin_operation", "builtinCase", func(ctx context.Context) error {
			return scanTemplate.ExecuteContext(ctx, ioutil.Discard, vars, nil, opts)
		}},
		{"compile_admission", "(*lexer).next", func(ctx context.Context) error {
			set := executionWorkload{}.set(false)
			_, err := set.ParseContext(ctx, "large", source, opts.ParseOptions)
			return err
		}},
		{"input_admission", "(*inputAdmission).clone", func(ctx context.Context) error {
			_, err := inputTemplate.RenderRestrictedContext(ctx, nil, input.Data)
			return err
		}},
	}
}

func TestExecutionCancellationControl(t *testing.T) {
	for _, operation := range cancellationOperations(t) {
		t.Run(operation.name, func(t *testing.T) {
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx := &measurementCancellationContext{Context: base, phase: operation.phase, cancel: cancel}
			if err := operation.run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation not preserved: %v", err)
			}
			if ctx.threshold == 0 {
				t.Fatal("cancellation phase was not reached")
			}
		})
	}
}

// Request/return timestamps are observations, never wall-clock pass criteria.
// A completed-before-request sample remains visible rather than being discarded.
func TestExecutionCancellationMeasurements(t *testing.T) {
	value := os.Getenv("JET_CANCELLATION_SAMPLES")
	if value == "" {
		t.Skip("set JET_CANCELLATION_SAMPLES for bounded diagnostic sampling")
	}
	count, err := strconv.Atoi(value)
	if err != nil || count <= 0 || count > 1000 {
		t.Fatal("JET_CANCELLATION_SAMPLES must be in [1,1000]")
	}
	for _, operation := range cancellationOperations(t) {
		t.Run(operation.name, func(t *testing.T) {
			base, cancel := context.WithCancel(context.Background())
			calibration := &measurementCancellationContext{Context: base, phase: operation.phase, cancel: cancel}
			calibrationError := operation.run(calibration)
			cancel()
			if !errors.Is(calibrationError, context.Canceled) || calibration.threshold == 0 {
				t.Fatalf("cannot locate cancellation phase: %v", calibrationError)
			}
			for i := 0; i < count; i++ {
				base, cancel := context.WithCancel(context.Background())
				ctx := &measurementCancellationContext{Context: base, threshold: calibration.threshold, ready: make(chan struct{})}
				start := time.Now()
				complete := make(chan struct{})
				requested := make(chan time.Time, 1)
				go func() {
					select {
					case <-ctx.ready:
					case <-complete:
					}
					at := time.Now()
					cancel()
					requested <- at
				}()
				err := operation.run(ctx)
				returned := time.Now()
				checks := atomic.LoadInt64(&ctx.calls)
				close(complete)
				request := <-requested
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if after := atomic.LoadInt64(&ctx.calls); after != checks {
					t.Fatal("context work continued after synchronous return")
				}
				t.Logf(`{"sample":%d,"request_ns":%d,"return_ns":%d,"overshoot_ns":%d,"checks":%d,"phase_check":%d,"canceled":%t,"post_return_checks":0}`, i, request.Sub(start).Nanoseconds(), returned.Sub(start).Nanoseconds(), returned.Sub(request).Nanoseconds(), checks, calibration.threshold, errors.Is(err, context.Canceled))
			}
		})
	}
}

func TestExecutionCallbackCancellationOwnership(t *testing.T) {
	for _, cooperative := range []bool{false, true} {
		t.Run(fmt.Sprintf("cooperative_%t", cooperative), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			entered, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var calls int64
			vars := jet.VarMap{}.SetFunc("callback", func(a jet.Arguments) reflect.Value {
				atomic.AddInt64(&calls, 1)
				close(entered)
				if cooperative {
					<-ctx.Done()
				} else {
					<-release
				}
				close(exited)
				return reflect.Value{}
			})
			f := executionWorkload{Source: "{{ callback() }}"}
			_, tpl := prepareMeasurement(t, executionModes[0], context.Background(), f, false)
			returned := make(chan error, 1)
			go func() { returned <- tpl.ExecuteContext(ctx, ioutil.Discard, vars, nil, measurementExecutionOptions()) }()
			<-entered
			request := time.Now()
			cancel()
			if !cooperative {
				select {
				case err := <-returned:
					t.Errorf("returned before callback release: %v", err)
				default:
				}
			}
			close(release)
			if err := <-returned; !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			<-exited
			t.Logf("callback cancellation return_ns=%d", time.Since(request).Nanoseconds())
			mode := executionModes[len(executionModes)-1]
			_, restricted := prepareMeasurement(t, mode, context.Background(), f, false)
			if _, err := restricted.RenderRestrictedContext(context.Background(), vars, nil); !errors.Is(err, jet.ErrCapabilityDenied) {
				t.Fatal(err)
			}
			if atomic.LoadInt64(&calls) != 1 {
				t.Fatal("restricted execution invoked supplied callback")
			}
		})
	}
}
