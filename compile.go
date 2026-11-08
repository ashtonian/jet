package jet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"sync"
)

// ParseOptions bounds one compilation and its dependency graph. Zero is unlimited.
// Depth counts active template and recursive parser frames. Nodes count AST
// construction (including repeated parsing in development mode). Steps count
// scanning, tokens, nodes and graph visits; exact totals may change by version.
type ParseOptions struct {
	MaxSourceBytes      int64
	MaxTotalSourceBytes int64
	MaxNodes            uint64
	MaxSteps            uint64
	MaxDepth            int
	MaxTemplates        uint64
}

var (
	ErrInvalidParseOptions   = errors.New("jet: invalid parse options")
	ErrParseSourceLimit      = errors.New("jet: source byte limit exceeded")
	ErrParseTotalSourceLimit = errors.New("jet: total source byte limit exceeded")
	ErrParseNodeLimit        = errors.New("jet: parse node limit exceeded")
	ErrParseStepLimit        = errors.New("jet: parse step limit exceeded")
	ErrParseDepthLimit       = errors.New("jet: parse depth limit exceeded")
	ErrParseTemplateLimit    = errors.New("jet: template count limit exceeded")
	ErrTemplateCycle         = errors.New("cyclic template dependency")
)

// ParseLimitError identifies a compilation allowance that was exhausted.
type ParseLimitError struct {
	Kind  string
	Limit uint64
}

func (e *ParseLimitError) Error() string {
	return fmt.Sprintf("jet: parse %s limit exceeded (%d)", e.Kind, e.Limit)
}
func (e *ParseLimitError) Unwrap() error {
	switch e.Kind {
	case "source bytes":
		return ErrParseSourceLimit
	case "total source bytes":
		return ErrParseTotalSourceLimit
	case "nodes":
		return ErrParseNodeLimit
	case "steps":
		return ErrParseStepLimit
	case "depth":
		return ErrParseDepthLimit
	case "templates":
		return ErrParseTemplateLimit
	default:
		return nil
	}
}

// ParseContext parses inline source synchronously. Dependencies use the same
// admission. A failed operation never publishes a partial template.
func (s *Set) ParseContext(ctx context.Context, name, source string, opts ParseOptions) (t *Template, err error) {
	a, err := s.parseAdmission(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer a.release()
	defer func() {
		if err != nil {
			t = nil
		}
	}()
	defer a.recover(&err)
	if s.restricted != nil && len(name) > 1024 {
		return nil, ErrCapabilityDenied
	}
	name, err = normalizeParseName(name)
	if err != nil {
		return nil, err
	}
	t, err = s.parse(name, source, false, a)
	if err == nil {
		a.check()
	}
	return t, err
}

// GetTemplateContext loads and admits a template graph, including cached ASTs.
// Loader calls and reads are owned synchronously: an arbitrary blocking Loader
// cannot be interrupted. ContextLoader can cooperate without a detached worker.
func (s *Set) GetTemplateContext(ctx context.Context, name string, opts ParseOptions) (t *Template, err error) {
	a, err := s.parseAdmission(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer a.release()
	defer func() {
		if err != nil {
			t = nil
		}
	}()
	defer a.recover(&err)
	t, err = s.getSiblingTemplate(name, "/", true, a)
	if err == nil {
		a.check()
	}
	return t, err
}

func recoverCompilation(err *error) { compilationPanic(recover(), err) }

type compilationAbort struct{ error }

func (e compilationAbort) Unwrap() error { return e.error }

func compilationPanic(p interface{}, err *error) {
	if p == nil {
		return
	}
	if abort, ok := p.(compilationAbort); ok {
		*err = abort.error
		return
	}
	panic(p)
}

func (a *parseAdmission) recover(err *error) {
	p := recover()
	if p != nil && a.terminal != nil {
		if _, ok := p.(runtime.Error); ok {
			panic(p)
		}
		if _, ok := p.(error); !ok {
			panic(p)
		}
	} else {
		compilationPanic(p, err)
	}
	// Cleanup cannot replace an observed resource or cancellation failure.
	if a.terminal != nil {
		*err = a.terminal
	}
}

// This state belongs to one synchronous operation, never a Set or cached AST.
type parseAdmission struct {
	ctx          context.Context
	opts         ParseOptions
	sources      []parseSource
	sourceIndex  map[string]int
	cached       []*Template
	cachedIndex  map[*Template]bool
	smallSources [4]parseSource
	smallCached  [4]*Template
	total        int64
	nodes        uint64
	steps        uint64
	depth        int
	parsing      int
	terminal     error
}

type parseSource struct {
	name   string
	size   int64
	active bool
}

type parseMetadata struct {
	sourceBytes int64
	nodes       uint64
	depth       int
	// Nonzero only for a fully compiled, acyclic graph owned by this Set.
	graphDepth int
}

var parseAdmissions = sync.Pool{New: func() interface{} { return new(parseAdmission) }}

func newParseAdmission(ctx context.Context, opts ParseOptions) (*parseAdmission, error) {
	a := parseAdmissions.Get().(*parseAdmission)
	if err := a.init(ctx, opts); err != nil {
		a.release()
		return nil, err
	}
	return a, nil
}

func (a *parseAdmission) release() {
	// Retain only fixed storage. Large graphs and caller references leave the pool.
	*a = parseAdmission{}
	parseAdmissions.Put(a)
}

func (a *parseAdmission) init(ctx context.Context, opts ParseOptions) error {
	if ctx == nil {
		return ErrInvalidParseOptions
	}
	if err := validateParseOptions(opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	a.ctx, a.opts = ctx, opts
	return nil
}

func (a *parseAdmission) check() {
	if a == nil {
		return
	}
	if a.terminal != nil {
		panic(compilationAbort{a.terminal})
	}
	if err := a.ctx.Err(); err != nil {
		a.stop(err)
	}
}

func (a *parseAdmission) stop(err error) {
	if a.terminal == nil {
		a.terminal = err
	}
	panic(compilationAbort{a.terminal})
}

func (a *parseAdmission) limit(kind string, limit uint64) {
	a.stop(&ParseLimitError{Kind: kind, Limit: limit})
}

func (a *parseAdmission) step(n uint64) {
	if a == nil {
		return
	}
	a.check()
	if a.opts.MaxSteps > 0 && (a.steps > a.opts.MaxSteps || n > a.opts.MaxSteps-a.steps) {
		a.limit("steps", a.opts.MaxSteps)
	}
	if n > math.MaxUint64-a.steps {
		a.limit("steps", math.MaxUint64)
	}
	a.steps += n
}

func (a *parseAdmission) enter() {
	a.step(1)
	if a.depth == int(^uint(0)>>1) || (a.opts.MaxDepth > 0 && a.depth >= a.opts.MaxDepth) {
		a.limit("depth", uint64(a.opts.MaxDepth))
	}
	a.depth++
}

func (a *parseAdmission) findSource(name string) (int, bool) {
	if a.sourceIndex != nil {
		i, ok := a.sourceIndex[name]
		return i, ok
	}
	for i := range a.sources {
		if a.sources[i].name == name {
			return i, true
		}
	}
	return 0, false
}

func (a *parseAdmission) prepareSource(name string) int {
	a.step(1)
	if i, ok := a.findSource(name); ok {
		if a.sources[i].active {
			panic(compilationAbort{fmt.Errorf("%w: %s", ErrTemplateCycle, parseDiagnostic(name))})
		}
		return i
	}
	if a.opts.MaxTemplates > 0 && uint64(len(a.sources)) >= a.opts.MaxTemplates {
		a.limit("templates", a.opts.MaxTemplates)
	}
	if a.sources == nil {
		a.sources = a.smallSources[:0]
	}
	i := len(a.sources)
	a.sources = append(a.sources, parseSource{name: name})
	if a.sourceIndex != nil {
		a.sourceIndex[name] = i
	} else if len(a.sources) > len(a.smallSources) {
		a.step(uint64(len(a.sources)))
		a.sourceIndex = make(map[string]int, len(a.sources))
		for j := range a.sources {
			a.sourceIndex[a.sources[j].name] = j
		}
	}
	return i
}

func (a *parseAdmission) source(name string, size int64) int {
	i := a.prepareSource(name)
	if a.opts.MaxSourceBytes > 0 && size > a.opts.MaxSourceBytes {
		a.limit("source bytes", uint64(a.opts.MaxSourceBytes))
	}
	old := a.sources[i].size
	if size <= old {
		return i
	}
	extra := size - old
	if a.opts.MaxTotalSourceBytes > 0 && extra > a.opts.MaxTotalSourceBytes-a.total {
		a.limit("total source bytes", uint64(a.opts.MaxTotalSourceBytes))
	}
	if extra > math.MaxInt64-a.total {
		a.limit("total source bytes", math.MaxInt64)
	}
	a.total += extra
	a.sources[i].size = size
	return i
}

func (a *parseAdmission) hasCached(t *Template) bool {
	if a.cachedIndex != nil {
		return a.cachedIndex[t]
	}
	for _, old := range a.cached {
		if old == t {
			return true
		}
	}
	return false
}

func (a *parseAdmission) markCached(t *Template) {
	if a.cached == nil {
		a.cached = a.smallCached[:0]
	}
	a.cached = append(a.cached, t)
	if a.cachedIndex != nil {
		a.cachedIndex[t] = true
	} else if len(a.cached) > len(a.smallCached) {
		a.step(uint64(len(a.cached)))
		a.cachedIndex = make(map[*Template]bool, len(a.cached))
		for _, old := range a.cached {
			a.cachedIndex[old] = true
		}
	}
}

func (a *parseAdmission) addNodes(n uint64) {
	a.step(1)
	if a.opts.MaxNodes > 0 && n > a.opts.MaxNodes-a.nodes {
		a.limit("nodes", a.opts.MaxNodes)
	}
	if n > math.MaxUint64-a.nodes {
		a.limit("nodes", math.MaxUint64)
	}
	a.nodes += n
}

func (t *Template) countNode() {
	if t.admission == nil {
		return
	}
	t.admission.addNodes(1)
	t.metadata.nodes++
}

func (t *Template) enterParse() {
	if t.admission == nil {
		return
	}
	t.admission.enter()
	t.parseDepth++
	if t.parseDepth+1 > t.metadata.depth {
		t.metadata.depth = t.parseDepth + 1
	}
}

func (t *Template) leaveParse() {
	if t.admission == nil {
		return
	}
	t.parseDepth--
	t.admission.depth--
}

func (a *parseAdmission) admitCached(t *Template) {
	// Successful compilation certifies the whole immutable graph. With no
	// limits or active parse ancestors there is nothing to re-account.
	if a.opts == (ParseOptions{}) && a.depth == 0 && t.metadata.graphDepth > 0 {
		a.check()
		return
	}
	i := a.source(t.Name, t.metadata.sourceBytes)
	height := t.metadata.graphDepth
	if height == 0 {
		height = t.metadata.depth
	}
	// The transitive height checks every diamond path before skipping a visit.
	if a.opts.MaxDepth > 0 && height > a.opts.MaxDepth-a.depth {
		a.limit("depth", uint64(a.opts.MaxDepth))
	}
	seen := a.hasCached(t)
	// A cold parse can introduce another version of a previously seen name.
	// Walk its cached dependencies to compare them with every active ancestor.
	if t.metadata.graphDepth > 0 && seen && a.parsing == 0 {
		return
	}
	if !seen {
		a.addNodes(t.metadata.nodes)
		a.markCached(t)
	}
	if t.extends == nil && len(t.imports) == 0 {
		return
	}
	a.enter()
	defer func() { a.depth-- }()
	a.sources[i].active = true
	defer func() { a.sources[i].active = false }()
	if t.extends != nil {
		a.admitCached(t.extends)
	}
	for _, dependency := range t.imports {
		a.admitCached(dependency)
	}
}

func (t *Template) recordGraphDepth() {
	height := t.metadata.depth
	include := func(child *Template) bool {
		t.admission.step(1)
		if child.set != t.set || child.metadata.graphDepth == 0 {
			return false
		}
		if child.metadata.graphDepth == int(^uint(0)>>1) {
			t.admission.limit("depth", uint64(child.metadata.graphDepth))
		}
		if h := child.metadata.graphDepth + 1; h > height {
			height = h
		}
		return true
	}
	if t.extends != nil && !include(t.extends) {
		return
	}
	for _, child := range t.imports {
		if !include(child) {
			return
		}
	}
	t.metadata.graphDepth = height
}

func parseDiagnostic(s string) string {
	const max = 160
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func (a *parseAdmission) readSource(name string, reader io.Reader) string {
	source := a.prepareSource(name)
	var buf bytes.Buffer
	var chunk [4096]byte
	empty := 0
	for {
		a.step(1)
		size := int64(buf.Len())
		readSize := len(chunk)
		if cap := a.opts.MaxSourceBytes; cap > 0 && cap-size < int64(readSize) {
			readSize = int(cap-size) + 1
		}
		if cap := a.opts.MaxTotalSourceBytes; cap > 0 {
			allowed := cap - a.total + a.sources[source].size - size
			if allowed < int64(readSize) {
				readSize = int(allowed) + 1
			}
		}
		n, err := reader.Read(chunk[:readSize])
		a.check()
		if n < 0 || n > readSize {
			panic(compilationAbort{errors.New("jet: loader returned an invalid read count")})
		}
		if n > 0 {
			a.step(uint64(n))
			a.source(name, size+int64(n))
			buf.Write(chunk[:n])
			empty = 0
		} else {
			empty++
			if err == nil && empty >= 100 {
				panic(compilationAbort{io.ErrNoProgress})
			}
		}
		if err == io.EOF {
			return buf.String()
		}
		if err != nil {
			panic(compilationAbort{err})
		}
	}
}

func (t *Template) nextExpressionDepth(children ...Node) int {
	if t.admission == nil {
		return 0
	}
	height := 1
	for _, child := range children {
		if depth := t.expressionDepth[child] + 1; depth > height {
			height = depth
		}
	}
	ancestors := t.admission.depth - t.parseDepth
	if t.admission.opts.MaxDepth > 0 && height > t.admission.opts.MaxDepth-ancestors {
		t.admission.limit("depth", uint64(t.admission.opts.MaxDepth))
	}
	if height+1 > t.metadata.depth {
		t.metadata.depth = height + 1
	}
	return height
}

func (t *Template) recordExpression(node Node, depth int) {
	if t.admission != nil {
		t.expressionDepth[node] = depth
	}
}

func (t *Template) recordCallDepth(call *CallExprNode) {
	if t.admission == nil {
		return
	}
	depth := t.nextExpressionDepth(call.BaseExpr)
	for _, arg := range call.CallArgs.Exprs {
		t.admission.step(1)
		if h := t.nextExpressionDepth(arg); h > depth {
			depth = h
		}
	}
	t.recordExpression(call, depth)
}

func validateParseOptions(opts ParseOptions) error {
	if opts.MaxSourceBytes < 0 || opts.MaxTotalSourceBytes < 0 || opts.MaxDepth < 0 {
		return ErrInvalidParseOptions
	}
	return nil
}
