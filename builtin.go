package jet

import "reflect"

type builtinContract uint8

const (
	builtinString builtinContract = iota + 1
	builtinInterpreter
	builtinOutput
	builtinJSON
	builtinDiagnostic
)

// Only immutable descriptors installed at initialization are builtin bindings.
// A function name, named function type or program counter is not authorization.
type builtinDescriptor struct {
	name     string
	legacy   reflect.Value
	execute  Func
	contract builtinContract
	writes   bool
}

var (
	builtinType      = reflect.TypeOf((*builtinDescriptor)(nil))
	builtinBindings  = make(map[*builtinDescriptor]bool)
	meteredVariables map[string]reflect.Value
	intType          = reflect.TypeOf(int(0))
)

func builtinForValue(v reflect.Value) (*builtinDescriptor, bool) {
	if !v.IsValid() || v.Type() != builtinType || v.IsNil() {
		return nil, false
	}
	b := v.Interface().(*builtinDescriptor)
	return b, builtinBindings[b]
}

func legacyBuiltinValue(v reflect.Value) reflect.Value {
	if b, ok := builtinForValue(v); ok {
		return b.legacy
	}
	return v
}

func stringBuiltin(legacy interface{}, execute Func) *builtinDescriptor {
	return &builtinDescriptor{legacy: reflect.ValueOf(legacy), execute: execute, contract: builtinString}
}

func functionBuiltin(fn Func) *builtinDescriptor {
	return &builtinDescriptor{legacy: reflect.ValueOf(fn), execute: fn, contract: builtinInterpreter}
}

func diagnosticBuiltin(fn Func) *builtinDescriptor {
	b := functionBuiltin(fn)
	b.contract = builtinDiagnostic
	return b
}

func safeBuiltin(fn SafeWriter) *builtinDescriptor {
	return &builtinDescriptor{
		legacy: reflect.ValueOf(fn), contract: builtinOutput, writes: true,
		execute: func(a Arguments) reflect.Value {
			w := &escapeWriter{rawWriter: a.runtime.Writer, safeWriter: fn}
			for i := 0; i < a.NumOfArguments(); i++ {
				if _, err := a.runtime.printValue(w, a.Get(i)); err != nil {
					panic(err)
				}
			}
			return reflect.Value{}
		},
	}
}

func (a Arguments) stringArg(i int) string {
	v := a.Get(i)
	if !v.IsValid() {
		a.Panicf("argument %d is not a valid string", i)
	}
	if v.Type() != stringType {
		v = a.runtime.convertValue(v, stringType)
	}
	return v.String()
}

func (a Arguments) intArg(i int) int {
	v := a.Get(i)
	if !v.IsValid() {
		a.Panicf("argument %d is not a valid integer", i)
	}
	if v.Type() != intType {
		v = a.runtime.convertValue(v, intType)
	}
	return int(v.Int())
}

func jsonBuiltin(legacy interface{}, execute Func) *builtinDescriptor {
	return &builtinDescriptor{legacy: reflect.ValueOf(legacy), execute: execute, contract: builtinJSON}
}
