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

type inputOptions struct {
	MaxBytes int64
	MaxItems uint64
	MaxSteps uint64
	MaxDepth int
}

type inputAdmission struct {
	ctx          context.Context
	opts         inputOptions
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

func admitInputs(ctx context.Context, vars VarMap, data interface{}, opts inputOptions) (out VarMap, snapshot interface{}, err error) {
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
	a.reserveItems(uint64(len(vars)))
	a.reserveBytes(uint64(len(vars)), uint64(5*(strconv.IntSize/8)))
	out = make(VarMap, len(vars))
	for name, value := range vars {
		a.reserveBytes(uint64(len(name)), 1)
		copied := a.copy(value, 1)
		out[name] = reflect.ValueOf(copied)
	}
	snapshot = a.copy(reflect.ValueOf(data), 1)
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

func (a *inputAdmission) copy(v reflect.Value, depth int) interface{} {
	a.step(1)
	if depth > a.opts.MaxDepth {
		a.limit("depth")
	}
	if !v.IsValid() {
		return nil
	}
	if !v.CanInterface() {
		a.deny("inaccessible value")
	}
	// Inspect types without converting or formatting their values: those paths
	// can themselves invoke String, MarshalJSON, Render, Range or method code.
	if v.Type().NumMethod() != 0 {
		a.deny("values with methods")
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return a.copy(v.Elem(), depth+1)
	}
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int:
		return int(v.Int())
	case reflect.Int8:
		return int8(v.Int())
	case reflect.Int16:
		return int16(v.Int())
	case reflect.Int32:
		return int32(v.Int())
	case reflect.Int64:
		return v.Int()
	case reflect.Uint:
		return uint(v.Uint())
	case reflect.Uint8:
		return uint8(v.Uint())
	case reflect.Uint16:
		return uint16(v.Uint())
	case reflect.Uint32:
		return uint32(v.Uint())
	case reflect.Uint64:
		return v.Uint()
	case reflect.Float32:
		return float32(v.Float())
	case reflect.Float64:
		return v.Float()
	case reflect.String:
		text := v.String()
		a.reserveBytes(uint64(len(text)), 1)
		// Go strings are immutable; copying collection headers detaches mutable data.
		return text
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return []interface{}(nil)
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
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 && v.Type().Elem().NumMethod() == 0 {
			a.reserveBytes(uint64(v.Len()), 1)
			out := make([]byte, v.Len())
			for i := range out {
				a.step(1)
				out[i] = byte(v.Index(i).Uint())
			}
			return out
		}
		a.reserveBytes(uint64(v.Len()), uint64(2*(strconv.IntSize/8)))
		out := make([]interface{}, v.Len())
		for i := range out {
			out[i] = a.copy(v.Index(i), depth+1)
		}
		return out
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			a.deny("map keys must be strings")
		}
		if v.Type().Key().NumMethod() != 0 {
			a.deny("map keys with methods")
		}
		if v.IsNil() {
			return map[string]interface{}(nil)
		}
		key := inputIdentity{kind: v.Kind(), typ: v.Type(), pointer: v.Pointer()}
		if a.active[key] {
			a.deny("cyclic input")
		}
		a.active[key] = true
		defer delete(a.active, key)
		a.reserveItems(uint64(v.Len()))
		a.reserveBytes(uint64(v.Len()), uint64(4*(strconv.IntSize/8)))
		out := make(map[string]interface{}, v.Len())
		iterator := v.MapRange()
		for iterator.Next() {
			name := iterator.Key().String()
			a.reserveBytes(uint64(len(name)), 1)
			out[name] = a.copy(iterator.Value(), depth+1)
		}
		return out
	default:
		a.deny("unsupported " + v.Kind().String())
	}
	return nil
}
