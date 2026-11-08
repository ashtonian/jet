package jet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
)

// RestrictedOptions selects an immutable data/capability profile. Every execution,
// compilation and input budget must be positive; supported depths are at most 64.
type RestrictedOptions struct {
	Execution                 ExecutionOptions
	Input                     InputOptions
	Globals                   VarMap
	BuiltinAliases            map[string]BuiltinBinding
	OptionalTemplates         []string
	Text                      bool
	LeftDelim, RightDelim     string
	LeftComment, RightComment string
}

// BuiltinBinding is an opaque audited binding, not permission for a Go callback.
type BuiltinBinding struct{ descriptor *builtinDescriptor }

// LookupBuiltin returns an audited default binding suitable for an explicit alias.
// Diagnostic bindings and unknown contracts are unavailable.
func LookupBuiltin(name string) (BuiltinBinding, bool) {
	b, ok := builtinForValue(meteredVariables[name])
	if !ok || !restrictedBuiltinContract(b.contract) {
		return BuiltinBinding{}, false
	}
	return BuiltinBinding{b}, true
}

type restrictedPolicy struct {
	execution ExecutionOptions
	input     InputOptions
	globals   VarMap
	bindings  VarMap
	allowed   map[*builtinDescriptor]bool
	names     map[string]bool
}

func restrictedBuiltinContract(contract builtinContract) bool {
	switch contract {
	case builtinString, builtinInterpreter, builtinOutput, builtinJSON:
		return true
	default:
		return false
	}
}

// NewRestrictedSet admits an immutable template namespace and data-only globals.
// Use Template.RenderRestrictedContext for an owned bounded result. General Go
// callbacks, custom loaders/writers and mutable configuration are not admitted.
func NewRestrictedSet(ctx context.Context, sources map[string]string, opts RestrictedOptions) (*Set, error) {
	e, p, i := opts.Execution, opts.Execution.ParseOptions, opts.Input
	if ctx == nil || e.MaxSteps == 0 || e.MaxDepth <= 0 || e.MaxDepth > 64 || e.MaxValueBytes <= 0 || e.MaxOutputBytes <= 0 || p.MaxSourceBytes <= 0 || p.MaxTotalSourceBytes <= 0 || p.MaxNodes == 0 || p.MaxSteps == 0 || p.MaxDepth <= 0 || p.MaxDepth > 64 || p.MaxTemplates == 0 || i.MaxBytes <= 0 || i.MaxItems == 0 || i.MaxSteps == 0 || i.MaxDepth <= 0 || i.MaxDepth > 64 {
		return nil, fmt.Errorf("%w: restricted execution requires positive budgets and depth <=64", ErrInvalidOptions)
	}
	for _, delimiter := range []string{opts.LeftDelim, opts.RightDelim, opts.LeftComment, opts.RightComment} {
		if len(delimiter) > 1024 {
			return nil, fmt.Errorf("%w: delimiter exceeds 1024 bytes", ErrInvalidOptions)
		}
	}
	globals, _, err := admitInputs(ctx, opts.Globals, nil, i)
	if err != nil {
		return nil, err
	}
	bundle, err := admitTemplateBundle(ctx, sources, p)
	if err != nil {
		return nil, err
	}
	policy := &restrictedPolicy{execution: e, input: i, globals: globals, bindings: make(VarMap, len(meteredVariables)), allowed: make(map[*builtinDescriptor]bool), names: make(map[string]bool, len(sources))}
	for name, value := range meteredVariables {
		b, ok := builtinForValue(value)
		if !ok {
			return nil, fmt.Errorf("%w: invalid builtin registry", ErrCapabilityDenied)
		}
		policy.bindings[name] = value
		if restrictedBuiltinContract(b.contract) {
			policy.allowed[b] = true
		} else if b.contract != builtinDiagnostic {
			return nil, fmt.Errorf("%w: unknown builtin contract", ErrCapabilityDenied)
		}
	}
	remaining := uint64(i.MaxBytes)
	if uint64(len(opts.BuiltinAliases)) > i.MaxItems {
		return nil, ErrInputLimit
	}
	for name, binding := range opts.BuiltinAliases {
		if err := ctx.Err(); err != nil {
			return nil, executionContextError(ctx)
		}
		if len(name) == 0 || uint64(len(name)) > remaining {
			return nil, ErrInputLimit
		}
		remaining -= uint64(len(name))
		b, ok := builtinForValue(reflect.ValueOf(binding.descriptor))
		if !ok || !restrictedBuiltinContract(b.contract) {
			return nil, fmt.Errorf("%w: alias needs an audited builtin identity", ErrCapabilityDenied)
		}
		policy.bindings[name] = reflect.ValueOf(b)
		policy.allowed[b] = true
	}
	for name := range bundle.sources {
		policy.names[name] = true
	}
	if uint64(len(opts.OptionalTemplates)) > p.MaxTemplates {
		return nil, ErrParseTemplateLimit
	}
	for _, name := range opts.OptionalTemplates {
		if err := ctx.Err(); err != nil {
			return nil, executionContextError(ctx)
		}
		if len(name) > 1024 {
			return nil, ErrCapabilityDenied
		}
		canonical, err := normalizeParseName(name)
		if err != nil {
			return nil, err
		}
		if len(canonical) > 1024 {
			return nil, ErrCapabilityDenied
		}
		policy.names[canonical] = true
	}
	if uint64(len(policy.names)) > p.MaxTemplates {
		return nil, ErrParseTemplateLimit
	}
	options := []Option{WithDelims(opts.LeftDelim, opts.RightDelim), WithCommentDelims(opts.LeftComment, opts.RightComment)}
	if opts.Text {
		options = append(options, WithSafeWriter(nil))
	}
	set := NewSet(bundle, options...)
	set.restricted = policy
	return set, nil
}

func (s *Set) checkMutable() {
	if s.restricted != nil {
		panic(fmt.Errorf("%w: restricted Set configuration is immutable", ErrCapabilityDenied))
	}
}

func minPositive(a, b uint64) uint64 {
	if a == 0 || b < a {
		return b
	}
	return a
}
func minPositiveInt(a, b int) int {
	if a == 0 || b < a {
		return b
	}
	return a
}
func minPositiveBytes(a, b int64) int64 {
	if a == 0 || b < a {
		return b
	}
	return a
}
func constrainParse(a, b ParseOptions) ParseOptions {
	a.MaxSourceBytes = minPositiveBytes(a.MaxSourceBytes, b.MaxSourceBytes)
	a.MaxTotalSourceBytes = minPositiveBytes(a.MaxTotalSourceBytes, b.MaxTotalSourceBytes)
	a.MaxNodes = minPositive(a.MaxNodes, b.MaxNodes)
	a.MaxSteps = minPositive(a.MaxSteps, b.MaxSteps)
	a.MaxDepth = minPositiveInt(a.MaxDepth, b.MaxDepth)
	a.MaxTemplates = minPositive(a.MaxTemplates, b.MaxTemplates)
	return a
}
func constrainExecution(a, b ExecutionOptions) ExecutionOptions {
	a.MaxSteps = minPositive(a.MaxSteps, b.MaxSteps)
	a.MaxDepth = minPositiveInt(a.MaxDepth, b.MaxDepth)
	a.MaxValueBytes = minPositiveBytes(a.MaxValueBytes, b.MaxValueBytes)
	a.MaxOutputBytes = minPositiveBytes(a.MaxOutputBytes, b.MaxOutputBytes)
	a.ParseOptions = constrainParse(a.ParseOptions, b.ParseOptions)
	return a
}
func (s *Set) parseAdmission(ctx context.Context, opts ParseOptions) (*parseAdmission, error) {
	if err := validateParseOptions(opts); err != nil {
		return nil, err
	}
	if s.restricted != nil {
		opts = constrainParse(opts, s.restricted.execution.ParseOptions)
	}
	return newParseAdmission(ctx, opts)
}
func (s *Set) nameAllowed(name string) bool { return s.restricted == nil || s.restricted.names[name] }
func (s *Set) authorizeName(name string) error {
	if s.restricted == nil {
		return nil
	}
	if len(name) <= 1024 {
		for _, extension := range s.extensions {
			if s.nameAllowed(name + extension) {
				return nil
			}
		}
	}
	return fmt.Errorf("%w: template name outside admitted namespace", ErrCapabilityDenied)
}

type restrictedSink struct {
	buffer bytes.Buffer
	result string
	owner  *Template
}

func (w *restrictedSink) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

// RenderRestrictedContext returns an execution-owned result, or an empty result
// on any error. It admits a fresh data snapshot and applies the Set's ceilings.
// Caller input/source maps must not be mutated concurrently with admission.
func (t *Template) RenderRestrictedContext(ctx context.Context, vars VarMap, data interface{}) (string, error) {
	if t.set.restricted == nil {
		return "", fmt.Errorf("%w: Set has no restricted profile", ErrCapabilityDenied)
	}
	sink := &restrictedSink{owner: t}
	if err := t.ExecuteContext(ctx, sink, vars, data, ExecutionOptions{}); err != nil {
		return "", err
	}
	return sink.result, nil
}

func (st *Runtime) isRestricted() bool { return st.set != nil && st.set.restricted != nil }

func (st *Runtime) denyCapability(reason string) {
	st.terminate(fmt.Errorf("%w: %s", ErrCapabilityDenied, reason))
}
func (st *Runtime) beforeNative(v reflect.Value) {
	st.checkpoint(1)
	if st.isRestricted() {
		st.denyCapability("native formatting or serialization")
	}
}
func (st *Runtime) beforeData(v reflect.Value) {
	if !st.isRestricted() {
		return
	}
	for v.IsValid() && v.Kind() == reflect.Interface {
		st.checkpoint(1)
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return
	}
	if v.Type().NumMethod() != 0 || reflect.PtrTo(v.Type()).NumMethod() != 0 {
		st.denyCapability("data with methods")
	}
	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String, reflect.Array, reflect.Slice:
		return
	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			return
		}
	}
	st.denyCapability("unsupported data value")
}
func (st *Runtime) allowCall(v reflect.Value) bool {
	if b, ok := builtinForValue(v); ok {
		return st.set.restricted.allowed[b]
	}
	if !v.IsValid() {
		return false
	}
	switch v.Type() {
	case reflect.TypeOf(jsonOutput{}), reflect.TypeOf(hiddenBool(false)), reflect.TypeOf((*intsRanger)(nil)), reflect.TypeOf((*sliceRanger)(nil)), reflect.TypeOf((*mapRanger)(nil)):
		return true
	default:
		return false
	}
}
func (st *Runtime) checkNode(node Node) {
	if !st.isRestricted() {
		return
	}
	switch node.Type() {
	case NodeText, NodeAction, NodeIf, NodeRange, NodeBlock, NodeYield, NodeInclude, NodeTry, NodeReturn, NodeList, NodePipe, NodeCommand, NodeSet, NodeNil, NodeBool, NodeString, NodeIdentifier, NodeField, NodeChain, NodeNumber, NodeUnderscore, NodeAdditiveExpr, NodeMultiplicativeExpr, NodeComparativeExpr, NodeNumericComparativeExpr, NodeLogicalExpr, NodeNotExpr, NodeTernaryExpr, NodeCallExpr, NodeIndexExpr, NodeSliceExpr:
		return
	default:
		st.denyCapability("unsupported AST kind")
	}
}

func inputContextError(ctx context.Context, err error) error {
	if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() != nil {
		return executionContextError(ctx)
	}
	return err
}
