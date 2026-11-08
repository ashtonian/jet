package jet

import (
	"bytes"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"io"
	"reflect"
	"strconv"

	"github.com/CloudyKit/fastprinter"
)

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	jsonNumberType    = reflect.TypeOf(json.Number(""))
)

func builtinMarshalJSON(a Arguments) reflect.Value {
	a.RequireNumOfArguments("json", 1, 1)
	value, err := a.runtime.marshalDataJSON(a.Get(0))
	if err != nil {
		return reflect.ValueOf([]byte(nil))
	}
	return reflect.ValueOf(value)
}

func builtinWriteJSON(a Arguments) reflect.Value {
	a.RequireNumOfArguments("writeJson", 1, 1)
	return reflect.ValueOf(jsonOutput{value: a.Get(0)})
}

type jsonOutput struct{ value reflect.Value }

func (v jsonOutput) Render(st *Runtime) {
	b, err := st.marshalDataJSON(v.value)
	if err != nil {
		panic(err)
	}
	if _, err = st.Writer.Write(b); err != nil {
		panic(err)
	}
	if _, err = st.Writer.Write([]byte{'\n'}); err != nil {
		panic(err)
	}
}

type valueBuffer struct {
	runtime *Runtime
	buffer  bytes.Buffer
}

func (w *valueBuffer) Write(b []byte) (int, error) {
	w.runtime.checkpoint(uint64(len(b)))
	size, ok := addValueSizes(int64(w.buffer.Len()), int64(len(b)))
	if !ok || uint64(size) > uint64(^uint(0)>>1) {
		w.runtime.failValueSize()
	}
	w.runtime.reserveValue(int64(len(b)))
	return w.buffer.Write(b)
}

func (st *Runtime) marshalDataJSON(v reflect.Value) ([]byte, error) {
	w := &valueBuffer{runtime: st}
	encoder := &dataJSONEncoder{runtime: st, writer: w}
	if err := encoder.value(v); err != nil {
		return nil, err
	}
	return w.buffer.Bytes(), nil
}

type dataJSONEncoder struct {
	runtime *Runtime
	writer  io.Writer
}

func (e *dataJSONEncoder) text(s string) error {
	_, err := fastprinter.PrintString(e.writer, s)
	return err
}

func (e *dataJSONEncoder) stringValue(s string) error {
	st := e.runtime
	st.checkpoint(uint64(len(s)))
	size, ok := jsonStringValueBound(s)
	st.valueLength(size, ok)
	st.reserveValue(size)
	st.checkpoint(uint64(size))
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = e.writer.Write(b)
	return err
}

func (e *dataJSONEncoder) nativeValue(v reflect.Value) error {
	st := e.runtime
	st.beforeNative(v)
	b, err := json.Marshal(v.Interface())
	st.checkpoint(0)
	if err != nil {
		return err
	}
	_, err = e.writer.Write(b)
	return err
}

func (e *dataJSONEncoder) value(v reflect.Value) error {
	st := e.runtime
	st.enter()
	defer st.leave()
	v = legacyBuiltinValue(indirectInterface(v))
	if !v.IsValid() {
		return e.text("null")
	}
	if v.Type().Implements(jsonMarshalerType) || v.Type().Implements(textMarshalerType) {
		return e.nativeValue(v)
	}
	if v.CanAddr() && (v.Addr().Type().Implements(jsonMarshalerType) || v.Addr().Type().Implements(textMarshalerType)) {
		return e.nativeValue(v.Addr())
	}
	if v.Type() == jsonNumberType {
		st.checkpoint(uint64(v.Len()))
		n := v.Len()
		if n == 0 {
			n = 1
		}
		st.reserveSlots(n, 2)
		return e.nativeValue(v)
	}
	var scratch [64]byte
	var scalar []byte
	switch k := v.Kind(); {
	case k == reflect.String:
		return e.stringValue(v.String())
	case k == reflect.Bool:
		scalar = strconv.AppendBool(scratch[:0], v.Bool())
	case isInt(k):
		scalar = strconv.AppendInt(scratch[:0], v.Int(), 10)
	case isUint(k):
		scalar = strconv.AppendUint(scratch[:0], v.Uint(), 10)
	case isFloat(k):
		// JSON's exponent selection/normalization is owned by encoding/json.
		st.reserveValue(128)
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return err
		}
		_, err = e.writer.Write(b)
		return err
	case k == reflect.Array || k == reflect.Slice:
		if k == reflect.Slice && v.IsNil() {
			return e.text("null")
		}
		byteMethod := reflect.PtrTo(v.Type().Elem()).Implements(jsonMarshalerType) || reflect.PtrTo(v.Type().Elem()).Implements(textMarshalerType)
		if k == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 && !byteMethod {
			st.checkpoint(uint64(v.Len()))
			if err := e.text(`"`); err != nil {
				return err
			}
			encoder := base64.NewEncoder(base64.StdEncoding, e.writer)
			if _, err := encoder.Write(v.Bytes()); err != nil {
				return err
			}
			if err := encoder.Close(); err != nil {
				return err
			}
			return e.text(`"`)
		}
		if err := e.text("["); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				if err := e.text(","); err != nil {
					return err
				}
			}
			if err := e.value(v.Index(i)); err != nil {
				return err
			}
		}
		return e.text("]")
	case k == reflect.Map && v.Type().Key().Kind() == reflect.String && !v.Type().Key().Implements(textMarshalerType):
		if v.IsNil() {
			return e.text("null")
		}
		keys := st.sortedStringKeys(v)
		if err := e.text("{"); err != nil {
			return err
		}
		for i, key := range keys {
			if i > 0 {
				if err := e.text(","); err != nil {
					return err
				}
			}
			if err := e.stringValue(key); err != nil {
				return err
			}
			if err := e.text(":"); err != nil {
				return err
			}
			keyValue := reflect.ValueOf(key).Convert(v.Type().Key())
			if err := e.value(v.MapIndex(keyValue)); err != nil {
				return err
			}
		}
		return e.text("}")
	default:
		return e.nativeValue(v)
	}
	_, err := e.writer.Write(scalar)
	return err
}
