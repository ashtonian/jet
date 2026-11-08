package jet

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strconv"
	"testing"
)

func inputTestOptions() InputOptions {
	return InputOptions{MaxBytes: 1 << 16, MaxItems: 1000, MaxSteps: 10000, MaxDepth: 32}
}

type inputStringer struct{ called *bool }

func (v inputStringer) String() string { *v.called = true; return "called" }

type inputMarshaler struct{ called *bool }

func (v inputMarshaler) MarshalJSON() ([]byte, error) { *v.called = true; return []byte("null"), nil }

type inputRenderer struct{ called *bool }

func (v inputRenderer) Render(*Runtime) { *v.called = true }

type inputRanger struct{ called *bool }

func (v inputRanger) Range() (reflect.Value, reflect.Value, bool) {
	*v.called = true
	return reflect.Value{}, reflect.Value{}, true
}
func (v inputRanger) ProvidesIndex() bool { *v.called = true; return false }

func TestInputAdmissionRejectsCapabilities(t *testing.T) {
	called := false
	fn := func() string { called = true; return "called" }
	channel := make(chan int)
	for _, value := range []interface{}{fn, Func(func(Arguments) reflect.Value { called = true; return reflect.Value{} }), SafeWriter(func(io.Writer, []byte) { called = true }), inputStringer{&called}, inputMarshaler{&called}, inputRenderer{&called}, inputRanger{&called}, channel, map[string]interface{}{"fn": fn}, []interface{}{fn}, uintptr(123), &called} {
		called = false
		_, _, err := admitInputs(context.Background(), nil, value, inputTestOptions())
		if !errors.Is(err, ErrCapabilityDenied) || called {
			t.Fatalf("%T: err=%v called=%t", value, err, called)
		}
	}
	vars := VarMap{}.Set("alias", fn)
	if _, _, err := admitInputs(context.Background(), vars, nil, inputTestOptions()); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
}

func TestInputAdmissionCyclesAndSharing(t *testing.T) {
	self := map[string]interface{}{}
	self["self"] = self
	slice := make([]interface{}, 1)
	slice[0] = slice
	for _, value := range []interface{}{self, slice} {
		if _, _, err := admitInputs(context.Background(), nil, value, inputTestOptions()); !errors.Is(err, ErrCapabilityDenied) {
			t.Fatalf("%T: %v", value, err)
		}
	}
	child := map[string]interface{}{"value": "ok"}
	value := map[string]interface{}{"left": child, "right": child}
	_, snapshot, err := admitInputs(context.Background(), nil, value, inputTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	child["value"] = "changed"
	out := snapshot.(map[string]interface{})
	if out["left"].(map[string]interface{})["value"] != "ok" {
		t.Fatal("snapshot retained mutable input")
	}
	out["left"].(map[string]interface{})["value"] = "new"
	if out["right"].(map[string]interface{})["value"] != "ok" {
		t.Fatal("snapshot aliases repeated mutable children")
	}
}

func TestInputAdmissionBudgets(t *testing.T) {
	for _, kind := range []string{"bytes", "items", "steps", "depth"} {
		t.Run(kind, func(t *testing.T) {
			opts := inputTestOptions()
			var data interface{} = map[string]interface{}{"value": []interface{}{"payload"}}
			switch kind {
			case "bytes":
				opts.MaxBytes = 1
			case "items":
				opts.MaxItems = 1
			case "steps":
				opts.MaxSteps = 1
			case "depth":
				opts.MaxDepth = 1
			}
			vars, snapshot, err := admitInputs(context.Background(), nil, data, opts)
			if !errors.Is(err, ErrInputLimit) || vars != nil || snapshot != nil {
				t.Fatalf("got %v", err)
			}
		})
	}
	opts := inputTestOptions()
	opts.MaxDepth = 65
	if _, _, err := admitInputs(context.Background(), nil, nil, opts); !errors.Is(err, ErrInputLimit) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := admitInputs(ctx, nil, nil, inputTestOptions()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestInputAdmissionByteSnapshot(t *testing.T) {
	source := []byte("original")
	_, snapshot, err := admitInputs(context.Background(), nil, source, inputTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	source[0] = 'X'
	if string(snapshot.([]byte)) != "original" {
		t.Fatal("retained mutable bytes")
	}
}

type inputNamedData map[string]interface{}
type inputPointerString string

func (*inputPointerString) String() string { panic("input method invoked") }

type inputPointerMap map[string]interface{}

func (*inputPointerMap) String() string { panic("input method invoked") }

type inputPointerSlice []string

func (*inputPointerSlice) String() string { panic("input method invoked") }

func TestInputAdmissionPointerMethods(t *testing.T) {
	for _, data := range []interface{}{
		inputPointerString("value"), inputPointerMap{}, inputPointerSlice{},
		map[inputPointerString]string{"key": "value"},
		map[string]interface{}{"nested": inputPointerString("value")},
	} {
		if _, _, err := admitInputs(context.Background(), nil, data, inputTestOptions()); !errors.Is(err, ErrCapabilityDenied) {
			t.Fatalf("%T: %v", data, err)
		}
	}
}

func TestInputAdmissionMapBoundaries(t *testing.T) {
	entryBytes := int64(4 * (strconv.IntSize / 8))
	for _, tc := range []struct {
		name  string
		data  map[string]interface{}
		bytes int64
		steps uint64
		depth int
	}{
		{"string", map[string]interface{}{"x": "y"}, entryBytes + 2, 9, 3},
		{"nil", map[string]interface{}{"x": nil}, entryBytes + 1, 7, 2},
	} {
		for _, data := range []interface{}{tc.data, inputNamedData(tc.data)} {
			t.Run(tc.name+"/"+reflect.TypeOf(data).String(), func(t *testing.T) {
				opts := inputTestOptions()
				opts.MaxBytes, opts.MaxItems, opts.MaxSteps, opts.MaxDepth = tc.bytes, 1, tc.steps, tc.depth
				if _, _, err := admitInputs(context.Background(), nil, data, opts); err != nil {
					t.Fatal(err)
				}
				for _, kind := range []string{"bytes", "steps", "depth"} {
					limited := opts
					switch kind {
					case "bytes":
						limited.MaxBytes--
					case "steps":
						limited.MaxSteps--
					case "depth":
						limited.MaxDepth--
					}
					if vars, snapshot, err := admitInputs(context.Background(), nil, data, limited); !errors.Is(err, ErrInputLimit) || vars != nil || snapshot != nil {
						t.Fatalf("%s: vars=%v snapshot=%v err=%v", kind, vars, snapshot, err)
					}
				}
			})
		}
	}
}

type inputCancellationContext struct {
	context.Context
	calls, cancelAt int
}

func (c *inputCancellationContext) Err() error {
	c.calls++
	if c.calls >= c.cancelAt {
		return context.Canceled
	}
	return nil
}

func TestInputAdmissionCancellationBoundaries(t *testing.T) {
	for _, data := range []interface{}{
		map[string]interface{}{"x": "y"}, inputNamedData{"x": "y"},
	} {
		// Scope admission, map/key/interface/string charges and the return boundary.
		for point := 1; point <= 10; point++ {
			ctx := &inputCancellationContext{Context: context.Background(), cancelAt: point}
			vars, snapshot, err := admitInputs(ctx, nil, data, inputTestOptions())
			if !errors.Is(err, context.Canceled) || vars != nil || snapshot != nil || ctx.calls != point {
				t.Fatalf("%T at %d: vars=%v snapshot=%v calls=%d err=%v", data, point, vars, snapshot, ctx.calls, err)
			}
		}
	}
}
