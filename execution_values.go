package jet

import (
	"reflect"
	"strconv"
	"strings"
)

// ConsumeSteps checks cancellation and consumes work from the active render.
// Cooperative callbacks must call it before the corresponding work. It does not
// authorize a callback for restricted execution or preempt native code.
func (st *Runtime) ConsumeSteps(steps uint64) { st.checkpoint(steps) }

// ReserveValueBytes reserves cumulative logical payload before construction.
// Negative requests and exhausted allowances terminate with ErrValueLimit.
// A reservation does not bound allocations performed before this call.
func (st *Runtime) ReserveValueBytes(size int64) { st.reserveValue(size) }

func (st *Runtime) failValueSize() {
	st.terminate(&LimitError{Kind: "value", Limit: uint64(st.execution.options.MaxValueBytes), cause: ErrValueLimit})
}

func (st *Runtime) reserveValue(size int64) {
	st.checkpoint(0)
	if size < 0 {
		st.failValueSize()
	}
	st.reserve(&st.execution.values, uint64(st.execution.options.MaxValueBytes), uint64(size), "value", ErrValueLimit)
}

func (st *Runtime) reserveSlots(count int, width int64) {
	size, ok := multiplyValueSize(int64(count), width)
	if !ok {
		st.failValueSize()
	}
	st.checkpoint(uint64(count))
	st.reserveValue(size)
}

func (st *Runtime) diagnosticValue(v reflect.Value) interface{} {
	if st.execution.ctx != nil {
		return getTypeString(v)
	}
	return v
}

func (st *Runtime) diagnosticText(s string) string {
	if st.execution.ctx != nil {
		return "<string>"
	}
	return s
}

func (st *Runtime) declareValue(sc *scope, name string, value reflect.Value) {
	st.checkpoint(uint64(len(name)) + 1)
	if _, exists := sc.variables[name]; !exists {
		st.reserveValue(int64(reflect.TypeOf("").Size() + reflect.TypeOf(reflect.Value{}).Size()))
	}
	sc.variables[name] = value
}

func (st *Runtime) valueLength(size int64, ok bool) int {
	if !ok || size < 0 || uint64(size) > uint64(^uint(0)>>1) {
		st.failValueSize()
	}
	st.reserveValue(size)
	return int(size)
}

func (st *Runtime) concatString(left string, right reflect.Value) string {
	var scratch [64]byte
	var scalar []byte
	var text string
	var payload []byte
	switch k := right.Kind(); {
	case k == reflect.String:
		text = right.String()
	case k == reflect.Slice && right.Type().Elem().Kind() == reflect.Uint8:
		payload = right.Bytes()
	case isInt(k):
		scalar = strconv.AppendInt(scratch[:0], right.Int(), 10)
	case isUint(k):
		scalar = strconv.AppendUint(scratch[:0], right.Uint(), 10)
	case isFloat(k):
		scalar = strconv.AppendFloat(scratch[:0], right.Float(), 'g', -1, right.Type().Bits())
	case k == reflect.Bool:
		scalar = strconv.AppendBool(scratch[:0], right.Bool())
	default:
		return st.concatFormattedValue(left, right)
	}
	size, ok := addValueSizes(int64(len(left)), int64(len(text)+len(payload)+len(scalar)))
	n := st.valueLength(size, ok)
	st.checkpoint(uint64(n))
	var out strings.Builder
	out.Grow(n)
	out.WriteString(left)
	out.WriteString(text)
	out.Write(payload)
	out.Write(scalar)
	return out.String()
}

func (st *Runtime) convertValue(value reflect.Value, target reflect.Type) reflect.Value {
	if b, ok := builtinForValue(value); ok && target.Kind() == reflect.Func {
		value = b.legacy
	}
	if st.execution.ctx == nil {
		return value.Convert(target)
	}
	if value.Kind() == reflect.String && target.Kind() == reflect.Slice {
		st.checkpoint(uint64(value.Len()))
		st.reserveSlots(value.Len(), int64(target.Elem().Size()))
	} else if target.Kind() == reflect.String {
		switch value.Kind() {
		case reflect.Slice:
			st.checkpoint(uint64(value.Len()))
			width := int64(1)
			if value.Type().Elem().Kind() == reflect.Int32 {
				width = 4
			}
			st.reserveSlots(value.Len(), width)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			st.reserveValue(4)
		}
	} else if target.Kind() == reflect.Array && value.Kind() == reflect.Slice {
		st.reserveSlots(target.Len(), int64(target.Elem().Size()))
	}
	return value.Convert(target)
}
