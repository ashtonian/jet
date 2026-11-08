package jet

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/CloudyKit/fastprinter"
)

var (
	formatterType = reflect.TypeOf((*fmt.Formatter)(nil)).Elem()
	errorType     = reflect.TypeOf((*error)(nil)).Elem()
)

func (st *Runtime) printValue(w io.Writer, v reflect.Value) (int, error) {
	if st.execution.ctx == nil {
		return fastprinter.PrintValue(w, v)
	}
	p := &st.printer
	// Write borrows formatted bytes until it returns. Reentrant printing must
	// leave the active workspace intact, including its scalar backing array.
	if p.runtime != nil {
		p = &dataPrinter{}
	}
	*p = dataPrinter{runtime: st, writer: w}
	defer func() { *p = dataPrinter{} }()
	err := p.value(v, false)
	return p.written, err
}

type dataPrinter struct {
	runtime *Runtime
	writer  io.Writer
	written int
	scratch [64]byte
}

func (p *dataPrinter) Write(b []byte) (int, error) {
	p.runtime.checkpoint(uint64(len(b)))
	n, err := p.writer.Write(b)
	p.written += n
	return n, err
}

func (p *dataPrinter) text(s string) error {
	p.runtime.checkpoint(uint64(len(s)))
	_, err := fastprinter.PrintString(p, s)
	return err
}

func (p *dataPrinter) value(v reflect.Value, nested bool) error {
	st := p.runtime
	st.enter()
	defer st.leave()
	v = indirectInterface(v)
	st.beforeData(v)
	if !v.IsValid() {
		return p.text("<nil>")
	}
	if b, ok := builtinForValue(v); ok {
		_, err := fastprinter.PrintValue(p, b.legacy)
		return err
	}
	if typ := v.Type(); typ.NumMethod() != 0 && (typ.Implements(formatterType) || typ.Implements(stringerType) || typ.Implements(errorType)) {
		// Native formatting hooks remain unrestricted host capabilities.
		st.beforeNative(v)
		var err error
		if nested {
			_, err = fmt.Fprint(p, v.Interface())
		} else {
			_, err = fastprinter.PrintValue(p, v)
		}
		st.checkpoint(0)
		return err
	}
	var scalar []byte
	switch k := v.Kind(); {
	case k == reflect.String:
		return p.text(v.String())
	case isInt(k):
		scalar = strconv.AppendInt(p.scratch[:0], v.Int(), 10)
	case isUint(k):
		scalar = strconv.AppendUint(p.scratch[:0], v.Uint(), 10)
	case k == reflect.Bool:
		scalar = strconv.AppendBool(p.scratch[:0], v.Bool())
	case isFloat(k):
		if !nested {
			_, err := fastprinter.PrintValue(p, v)
			return err
		}
		scalar = strconv.AppendFloat(p.scratch[:0], v.Float(), 'g', -1, v.Type().Bits())
	case k == reflect.Array || k == reflect.Slice:
		if !nested && k == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			_, err := p.Write(v.Bytes())
			return err
		}
		if err := p.text("["); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				if err := p.text(" "); err != nil {
					return err
				}
			}
			if err := p.value(v.Index(i), true); err != nil {
				return err
			}
		}
		return p.text("]")
	case k == reflect.Map && v.Type().Key().Kind() == reflect.String:
		keys := st.sortedStringKeys(v)
		if err := p.text("map["); err != nil {
			return err
		}
		for i, key := range keys {
			if i > 0 {
				if err := p.text(" "); err != nil {
					return err
				}
			}
			if err := p.text(key); err != nil {
				return err
			}
			if err := p.text(":"); err != nil {
				return err
			}
			keyValue := reflect.ValueOf(key).Convert(v.Type().Key())
			if err := p.value(v.MapIndex(keyValue), true); err != nil {
				return err
			}
		}
		return p.text("]")
	default:
		st.beforeNative(v)
		_, err := fastprinter.PrintValue(p, v)
		st.checkpoint(0)
		return err
	}
	_, err := p.Write(scalar)
	return err
}

func (st *Runtime) sortedStringKeys(v reflect.Value) []string {
	st.reserveSlots(v.Len(), int64(reflect.TypeOf("").Size()))
	keys := make([]string, 0, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		st.checkpoint(1)
		key := iter.Key().String()
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		st.checkpoint(uint64(len(keys[i])) + uint64(len(keys[j])) + 1)
		return keys[i] < keys[j]
	})
	return keys
}

type valueBuilder struct {
	runtime *Runtime
	strings.Builder
}

func (w *valueBuilder) Write(b []byte) (int, error) {
	w.runtime.reserveValue(int64(len(b)))
	return w.Builder.Write(b)
}

func (st *Runtime) concatFormattedValue(left string, right reflect.Value) string {
	w := &valueBuilder{runtime: st}
	st.checkpoint(uint64(len(left)))
	st.reserveValue(int64(len(left)))
	w.Builder.WriteString(left)
	// Concatenation uses fmt's nested scalar/byte-slice conventions.
	p := &dataPrinter{runtime: st, writer: w}
	if err := p.value(right, true); err != nil {
		panic(err)
	}
	return w.String()
}
