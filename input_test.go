package jet

import (
	"context"
	"errors"
	"io"
	"reflect"
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
