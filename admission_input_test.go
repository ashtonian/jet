package jet

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func TestInputAdmissionOwnsAddressableScalarCells(t *testing.T) {
	number, text := 7, "original"
	vars := VarMap{"number": reflect.ValueOf(&number).Elem(), "text": reflect.ValueOf(&text).Elem()}
	snapshot, _, err := admitInputs(context.Background(), vars, nil, inputTestOptions())
	if err != nil {
		t.Fatal(err)
	}
	number = 9
	text = "changed"
	if snapshot["number"].Int() != 7 || snapshot["text"].String() != "original" {
		t.Fatal("snapshot retained caller cells")
	}
}

func TestInputAdmissionLongActivePathAndSharing(t *testing.T) {
	child := map[string]interface{}{"value": "leaf"}
	var data interface{} = child
	for i := 0; i < 20; i++ {
		data = map[string]interface{}{"child": data}
	}
	_, snapshot, err := admitInputs(context.Background(), nil, data, InputOptions{MaxBytes: 65536, MaxItems: 1000, MaxSteps: 10000, MaxDepth: 64})
	if err != nil || snapshot == nil {
		t.Fatal(err)
	}
	// Repeated children outside the active path remain independently copied.
	root := map[string]interface{}{"left": data, "right": data}
	_, snapshot, err = admitInputs(context.Background(), nil, root, InputOptions{MaxBytes: 65536, MaxItems: 1000, MaxSteps: 10000, MaxDepth: 64})
	if err != nil || snapshot == nil {
		t.Fatal(err)
	}
	child["cycle"] = root
	_, _, err = admitInputs(context.Background(), nil, root, InputOptions{MaxBytes: 65536, MaxItems: 1000, MaxSteps: 10000, MaxDepth: 64})
	if !errors.Is(err, ErrCapabilityDenied) {
		t.Fatal(err)
	}
}

func TestRestrictedInputScopeReuseIsolation(t *testing.T) {
	first := restrictedTemplate(t, map[string]string{"main.jet": `{{ a }}{{ a = "changed" }}`}, restrictedTestOptions())
	second := restrictedTemplate(t, map[string]string{"main.jet": `{{ isset(a) }}:{{ b }}`}, restrictedTestOptions())
	original := VarMap{}.Set("a", "secret")
	for i := 0; i < 64; i++ {
		result, err := first.RenderRestrictedContext(context.Background(), original, nil)
		if err != nil || result != "secret" || original["a"].String() != "secret" {
			t.Fatalf("%q %v", result, err)
		}
		called := false
		bad := VarMap{}.Set("a", "partial").Set("bad", func() { called = true })
		if _, err := first.RenderRestrictedContext(context.Background(), bad, nil); !errors.Is(err, ErrCapabilityDenied) || called {
			t.Fatalf("%v called=%t", err, called)
		}
		next, err := second.RenderRestrictedContext(context.Background(), VarMap{}.Set("b", "public"), nil)
		if err != nil || next != "false:public" || result != "secret" {
			t.Fatalf("%q %v previous=%q", next, err, result)
		}
	}
	large := VarMap{}.Set("a", "large")
	for i := 0; i < 64; i++ {
		large.Set(fmt.Sprint(i), i)
	}
	if _, err := first.RenderRestrictedContext(context.Background(), large, nil); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.GC()
	if result, err := second.RenderRestrictedContext(context.Background(), VarMap{}.Set("b", "after GC"), nil); err != nil || result != "false:after GC" {
		t.Fatalf("%q %v", result, err)
	}
}

func TestRestrictedConcurrentVarsMapReuse(t *testing.T) {
	tpl := restrictedTemplate(t, map[string]string{"main.jet": `{{ value }}`}, restrictedTestOptions())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			value := fmt.Sprint(i)
			vars := VarMap{}.Set("value", value)
			for n := 0; n < 64; n++ {
				result, err := tpl.RenderRestrictedContext(context.Background(), vars, nil)
				if err != nil || result != value {
					t.Errorf("%q %v want %q", result, err, value)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}
