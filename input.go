package jet

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

var (
	ErrInputLimit       = errors.New("jet: input admission limit exceeded")
	ErrCapabilityDenied = errors.New("jet: capability denied")
)

// InputOptions bounds data copied into one restricted execution. All limits must be positive.
type InputOptions struct {
	MaxBytes int64
	MaxItems uint64
	MaxSteps uint64
	MaxDepth int
}

type inputAdmission struct {
	ctx          context.Context
	opts         InputOptions
	bytes        int64
	items, steps uint64
	active       map[inputIdentity]bool
}
type inputIdentity struct {
	kind    reflect.Kind
	typ     reflect.Type
	pointer uintptr
	length  int
}

func admitInputs(ctx context.Context, vars VarMap, data interface{}, opts InputOptions) (VarMap, interface{}, error) {
	return admitInputScopes(ctx, []VarMap{vars}, data, opts)
}

func admitInputScopes(ctx context.Context, scopes []VarMap, data interface{}, opts InputOptions) (out VarMap, snapshot interface{}, err error) {
	if ctx == nil || opts.MaxBytes <= 0 || opts.MaxItems == 0 || opts.MaxSteps == 0 || opts.MaxDepth <= 0 || opts.MaxDepth > 64 {
		return nil, nil, fmt.Errorf("%w: positive input budgets and depth <=64 required", ErrInputLimit)
	}
	a := &inputAdmission{ctx: ctx, opts: opts, active: make(map[inputIdentity]bool)}
	defer func() {
		if p := recover(); p != nil {
			if failure, ok := p.(inputAbort); ok {
				err = failure.error
				out = nil
				snapshot = nil
			} else {
				panic(p)
			}
		}
	}()
	out = make(VarMap)
	for _, scope := range scopes {
		a.reserveItems(uint64(len(scope)))
		a.reserveBytes(uint64(len(scope)), uint64(5*(strconv.IntSize/8)))
		for name, value := range scope {
			a.reserveBytes(uint64(len(name)), 1)
			out[name] = a.clone(value, 1)
		}
	}
	value := a.clone(reflect.ValueOf(data), 1)
	if value.IsValid() {
		snapshot = value.Interface()
	}
	a.step(0)
	return out, snapshot, nil
}

type inputAbort struct{ error }

func (a *inputAdmission) deny(reason string) {
	panic(inputAbort{fmt.Errorf("%w: %s", ErrCapabilityDenied, reason)})
}
func (a *inputAdmission) limit(kind string) {
	panic(inputAbort{fmt.Errorf("%w: %s", ErrInputLimit, kind)})
}
func (a *inputAdmission) step(n uint64) {
	if err := a.ctx.Err(); err != nil {
		panic(inputAbort{err})
	}
	if n > a.opts.MaxSteps-a.steps {
		a.limit("steps")
	}
	a.steps += n
}
func (a *inputAdmission) reserveItems(n uint64) {
	a.step(1)
	if n > a.opts.MaxItems-a.items {
		a.limit("items")
	}
	a.items += n
}
func (a *inputAdmission) reserveBytes(n, width uint64) {
	a.step(1)
	remaining := uint64(a.opts.MaxBytes - a.bytes)
	if width != 0 && n > remaining/width {
		a.limit("bytes")
	}
	a.bytes += int64(n * width)
}

func (a *inputAdmission) clone(v reflect.Value, depth int) reflect.Value {
	a.step(1)
	if depth > a.opts.MaxDepth {
		a.limit("depth")
	}
	if !v.IsValid() {
		return reflect.Value{}
	}
	if !v.CanInterface() {
		a.deny("inaccessible value")
	}
	if v.Type().NumMethod() != 0 || reflect.PtrTo(v.Type()).NumMethod() != 0 {
		a.deny("values with methods")
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		return a.clone(v.Elem(), depth+1)
	}
	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		a.reserveBytes(uint64(v.Type().Size()), 1)
		return reflect.ValueOf(v.Interface())
	case reflect.String:
		a.reserveBytes(uint64(v.Len()), 1)
		return reflect.ValueOf(v.Interface())
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return reflect.Zero(v.Type())
		}
		if v.Kind() == reflect.Slice {
			key := inputIdentity{kind: v.Kind(), typ: v.Type(), pointer: v.Pointer(), length: v.Len()}
			if a.active[key] {
				a.deny("cyclic input")
			}
			a.active[key] = true
			defer delete(a.active, key)
		}
		a.reserveItems(uint64(v.Len()))
		a.reserveBytes(uint64(v.Len()), uint64(v.Type().Elem().Size()))
		var out reflect.Value
		if v.Kind() == reflect.Slice {
			out = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		} else {
			out = reflect.New(v.Type()).Elem()
		}
		for i := 0; i < v.Len(); i++ {
			child := a.clone(v.Index(i), depth+1)
			if child.IsValid() {
				out.Index(i).Set(child)
			}
		}
		return out
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String || v.Type().Key().NumMethod() != 0 || reflect.PtrTo(v.Type().Key()).NumMethod() != 0 {
			a.deny("map keys must be method-free strings")
		}
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		key := inputIdentity{kind: v.Kind(), typ: v.Type(), pointer: v.Pointer()}
		if a.active[key] {
			a.deny("cyclic input")
		}
		a.active[key] = true
		defer delete(a.active, key)
		a.reserveItems(uint64(v.Len()))
		a.reserveBytes(uint64(v.Len()), uint64(v.Type().Key().Size()+v.Type().Elem().Size()))
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iterator := v.MapRange()
		for iterator.Next() {
			name := iterator.Key()
			a.reserveBytes(uint64(name.Len()), 1)
			child := a.clone(iterator.Value(), depth+1)
			if !child.IsValid() {
				child = reflect.Zero(v.Type().Elem())
			}
			out.SetMapIndex(name, child)
		}
		return out
	default:
		a.deny("unsupported " + v.Kind().String())
	}
	return reflect.Value{}
}
