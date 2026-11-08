package jet

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

func (st *Runtime) mapKey(receiver, key reflect.Value) (reflect.Value, error) {
	if !key.IsValid() || !key.CanInterface() {
		return reflect.Value{}, fmt.Errorf("map index must be a valid accessible value")
	}
	target := receiver.Type().Key()
	if key.Type() != target {
		if !key.Type().ConvertibleTo(target) {
			return reflect.Value{}, fmt.Errorf("cannot use %s as map key of type %s", key.Type(), target)
		}
		key = st.convertValue(key, target)
	}
	if !st.comparableMapKey(key) {
		return reflect.Value{}, fmt.Errorf("map key contains an unhashable value")
	}
	return key, nil
}

// Interface keys can contain unhashable dynamic values despite their comparable type.
func (st *Runtime) comparableMapKey(v reflect.Value) bool {
	st.enter()
	defer st.leave()
	if !v.Type().Comparable() {
		return false
	}
	switch v.Kind() {
	case reflect.Interface:
		return v.IsNil() || st.comparableMapKey(v.Elem())
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			st.checkpoint(1)
			if !st.comparableMapKey(v.Index(i)) {
				return false
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			st.checkpoint(1)
			if !st.comparableMapKey(v.Field(i)) {
				return false
			}
		}
	case reflect.String:
		st.checkpoint(uint64(v.Len()))
	}
	return true
}

// Promoted fields can cross nil embedded pointers, which FieldByIndex would panic on.
func (st *Runtime) fieldByIndex(v reflect.Value, index []int) (reflect.Value, error) {
	for _, i := range index {
		st.checkpoint(1)
		var isNil bool
		v, isNil = st.indirect(v)
		if isNil {
			return reflect.Value{}, fmt.Errorf("nil pointer evaluating embedded struct field")
		}
		v = v.Field(i)
	}
	if !v.CanInterface() {
		return reflect.Value{}, fmt.Errorf("cannot access unexported struct field")
	}
	return v, nil
}

func (st *Runtime) sliceValue(base reflect.Value, start, end Expression) (reflect.Value, error) {
	if !base.IsValid() {
		return reflect.Value{}, fmt.Errorf("cannot slice an invalid value")
	}
	switch base.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
	default:
		return reflect.Value{}, fmt.Errorf("cannot slice %s", base.Type())
	}
	first, last := 0, base.Len()
	for i, node := range []Expression{start, end} {
		if node == nil {
			continue
		}
		value := st.evalPrimaryExpressionGroup(node)
		n, err := boundedIndex(value, base.Len(), true)
		if err != nil {
			return reflect.Value{}, err
		}
		if i == 0 {
			first = n
		} else {
			last = n
		}
	}
	if first > last {
		return reflect.Value{}, fmt.Errorf("slice bounds out of range [%d:%d] with length %d", first, last, base.Len())
	}
	if base.Kind() == reflect.Array && !base.CanAddr() {
		st.reserveSlots(base.Len(), int64(base.Type().Elem().Size()))
		st.checkpoint(uint64(base.Type().Size()))
		copy := reflect.New(base.Type()).Elem()
		copy.Set(base)
		st.checkpoint(0)
		base = copy
	}
	return base.Slice(first, last), nil
}

func boundedIndex(index reflect.Value, length int, inclusive bool) (int, error) {
	upper := length
	if !inclusive {
		upper--
	}
	var x int64
	switch k := index.Kind(); {
	case isInt(k):
		x = index.Int()
	case isUint(k):
		if upper < 0 || index.Uint() > uint64(upper) {
			return 0, fmt.Errorf("index out of range")
		}
		x = int64(index.Uint())
	case isFloat(k):
		value := math.Trunc(index.Float())
		if value != value || value < 0 || value >= math.Ldexp(1, strconv.IntSize-1) || value > float64(upper) {
			return 0, fmt.Errorf("index out of range")
		}
		x = int64(value)
	default:
		return 0, fmt.Errorf("cannot index slice/array/string with %s", getTypeString(index))
	}
	if x < 0 || upper < 0 || uint64(x) > uint64(upper) {
		return 0, fmt.Errorf("index out of range: %d", x)
	}
	return int(x), nil
}
