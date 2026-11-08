package jet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"reflect"
	"sync"
	"text/template"
)

// Set is responsible to load, parse and cache templates.
// Every Jet template is associated with a Set.
type Set struct {
	loader          Loader
	cache           Cache
	escapee         SafeWriter    // escapee to use at runtime
	globals         VarMap        // global scope for this template set
	gmx             *sync.RWMutex // global variables map mutex
	extensions      []string
	developmentMode bool
	leftDelim       string
	rightDelim      string
	leftComment     string
	rightComment    string
	restricted      *restrictedPolicy
	defaultHTML     bool
}

// Option is the type of option functions that can be used in NewSet().
type Option func(*Set)

// NewSet returns a new Set relying on loader. NewSet panics if a nil Loader is passed.
func NewSet(loader Loader, opts ...Option) *Set {
	if loader == nil {
		panic(errors.New("jet: NewSet() must not be called with a nil loader"))
	}

	s := &Set{
		loader:      loader,
		cache:       &cache{},
		escapee:     template.HTMLEscape,
		defaultHTML: true,
		globals:     VarMap{},
		gmx:         &sync.RWMutex{},
		extensions: []string{
			"", // in case the path is given with the correct extension already
			".jet",
			".html.jet",
			".jet.html",
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithCache returns an option function that sets the cache to use for template parsing results.
// Use InDevelopmentMode() to disable caching of parsed templates. By default, Jet uses a
// concurrency-safe in-memory cache that holds templates forever.
func WithCache(c Cache) Option {
	if c == nil {
		panic(errors.New("jet: WithCache() must not be called with a nil cache"))
	}
	return func(s *Set) {
		s.checkMutable()
		s.cache = c
	}
}

// WithSafeWriter returns an option function that sets the escaping function to use when executing
// templates. By default, Jet uses a writer that takes care of HTML escaping. Pass nil to disable escaping.
func WithSafeWriter(w SafeWriter) Option {
	return func(s *Set) {
		s.checkMutable()
		s.escapee = w
		s.defaultHTML = false
	}
}

// WithDelims returns an option function that sets the delimiters to the specified strings.
// Parsed templates will inherit the settings. Not setting them leaves them at the default: `{{` and `}}`.
func WithDelims(left, right string) Option {
	return func(s *Set) {
		s.checkMutable()
		s.leftDelim = left
		s.rightDelim = right
	}
}

// WithCommentDelims returns an option function that sets the comment delimiters to the specified strings.
// Parsed templates will inherit the settings. Not setting them leaves them at the default: `{*` and `*}`.
func WithCommentDelims(left, right string) Option {
	return func(s *Set) {
		s.checkMutable()
		s.leftComment = left
		s.rightComment = right
	}
}

// WithTemplateNameExtensions returns an option function that sets the extensions to try when looking
// up template names in the cache or loader. Default extensions are `""` (no extension), `".jet"`,
// `".html.jet"`, `".jet.html"`. Extensions will be tried in the order they are defined in the slice.
// WithTemplateNameExtensions panics when you pass in a nil or empty slice.
func WithTemplateNameExtensions(extensions []string) Option {
	if len(extensions) == 0 {
		panic(errors.New("jet: WithTemplateNameExtensions() must not be called with a nil or empty slice of extensions"))
	}
	return func(s *Set) {
		s.checkMutable()
		s.extensions = extensions
	}
}

// InDevelopmentMode returns an option function that toggles development mode on, meaning the cache will
// always be bypassed and every template lookup will go to the loader.
func InDevelopmentMode() Option {
	return DevelopmentMode(true)
}

// DevelopmentMode returns an option function that sets development mode on or off. "On" means the cache will
// always be bypassed and every template lookup will go to the loader.
func DevelopmentMode(mode bool) Option {
	return func(s *Set) {
		s.checkMutable()
		s.developmentMode = mode
	}
}

// GetTemplate tries to find (and parse, if not yet parsed) the template at the specified path.
//
// For example, GetTemplate("catalog/products.list") with extensions set to []string{"", ".html.jet",".jet"}
// will try to look for:
//  1. catalog/products.list
//  2. catalog/products.list.html.jet
//  3. catalog/products.list.jet
//
// in the set's templates cache, and if it can't find the template it will try to load the same paths via
// the loader, and, if parsed successfully, cache the template (unless running in development mode).
func (s *Set) GetTemplate(templatePath string) (t *Template, err error) {
	return s.getSiblingTemplate(templatePath, "/", true, nil)
}

func (s *Set) getSiblingTemplate(templatePath, siblingPath string, cacheAfterParsing bool, parsing *parseAdmission) (t *Template, err error) {
	if s.restricted != nil && (len(templatePath) > 1024 || len(siblingPath) > 1024) {
		return nil, ErrCapabilityDenied
	}
	templatePath = path.Clean(filepath.ToSlash(templatePath))
	siblingPath = filepath.ToSlash(siblingPath)
	if !path.IsAbs(templatePath) {
		siblingDir := path.Dir(siblingPath)
		templatePath = path.Join(siblingDir, templatePath)
	}
	if err := s.authorizeName(templatePath); err != nil {
		return nil, err
	}
	return s.getTemplate(templatePath, cacheAfterParsing, parsing)
}

// same as GetTemplate, but doesn't cache a template when found through the loader.
func (s *Set) getTemplate(templatePath string, cacheAfterParsing bool, parsing *parseAdmission) (t *Template, err error) {
	if parsing == nil {
		if s.restricted == nil && !s.developmentMode {
			if t, ok := s.getTemplateFromCache(templatePath); ok && t.metadata.graphDepth > 0 {
				return t, nil
			}
		}
		parsing, _ = s.parseAdmission(context.Background(), ParseOptions{})
		defer parsing.release()
	}
	defer parsing.recover(&err)
	parsing.step(1)
	if !s.developmentMode {
		t, found := s.getTemplateFromCache(templatePath)
		if found {
			parsing.admitCached(t)
			return t, nil
		}
	}

	t, err = s.getTemplateFromLoader(templatePath, cacheAfterParsing, parsing)
	if err == nil && cacheAfterParsing && !s.developmentMode {
		s.cache.Put(templatePath, t)
	}
	return t, err
}

func (s *Set) getTemplateFromCache(templatePath string) (t *Template, ok bool) {
	// check path with all possible extensions in cache
	for _, extension := range s.extensions {
		canonicalPath := templatePath + extension
		if !s.nameAllowed(canonicalPath) {
			continue
		}
		if t := s.cache.Get(canonicalPath); t != nil && t.set == s {
			return t, true
		}
	}
	return nil, false
}

func (s *Set) getTemplateFromLoader(templatePath string, cacheAfterParsing bool, parsing *parseAdmission) (t *Template, err error) {
	// check path with all possible extensions in loader
	for _, extension := range s.extensions {
		canonicalPath := templatePath + extension
		if !s.nameAllowed(canonicalPath) {
			continue
		}
		parsing.step(1)
		var found bool
		if loader, ok := s.loader.(ContextLoader); ok {
			found, err = loader.ExistsContext(parsing.ctx, canonicalPath)
			if err != nil {
				return nil, err
			}
		} else {
			found = s.loader.Exists(canonicalPath)
		}
		parsing.check()
		if found {
			return s.loadFromFile(canonicalPath, cacheAfterParsing, parsing)
		}
	}
	return nil, fmt.Errorf("template %s could not be found", templatePath)
}

func (s *Set) loadFromFile(templatePath string, cacheAfterParsing bool, parsing *parseAdmission) (template *Template, err error) {
	parsing.prepareSource(templatePath)
	var f io.ReadCloser
	if loader, ok := s.loader.(ContextLoader); ok {
		f, err = loader.OpenContext(parsing.ctx, templatePath)
	} else {
		f, err = s.loader.Open(templatePath)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := f.Close()
		if err == nil && closeErr != nil {
			template = nil
			err = closeErr
		}
	}()
	parsing.check()
	content := parsing.readSource(templatePath, f)
	return s.parse(templatePath, content, cacheAfterParsing, parsing)
}

// Parse parses `contents` as if it were located at `templatePath`, but won't put the result into the cache.
// Any referenced template (e.g. via `extends` or `import` statements) will be tried to be loaded from the cache.
// If a referenced template has to be loaded and parsed, it will also not be put into the cache after parsing.
func (s *Set) Parse(templatePath, contents string) (template *Template, err error) {
	if s.restricted != nil && len(templatePath) > 1024 {
		return nil, ErrCapabilityDenied
	}
	templatePath, err = normalizeParseName(templatePath)
	if err != nil {
		return nil, err
	}
	return s.parse(templatePath, contents, false, nil)
}

func normalizeParseName(templatePath string) (string, error) {
	templatePath = filepath.ToSlash(templatePath)
	switch path.Base(templatePath) {
	case ".", "/":
		return "", errors.New("template path has no base name")
	}
	return path.Join("/", templatePath), nil
}

// AddGlobal adds a global variable into the Set,
// overriding any value previously set under the specified key.
// It returns the Set it was called on to allow for method chaining.
func (s *Set) AddGlobal(key string, i interface{}) *Set {
	s.checkMutable()
	s.gmx.Lock()
	defer s.gmx.Unlock()
	s.globals[key] = reflect.ValueOf(i)
	return s
}

// LookupGlobal returns the global variable previously set under the specified key.
// It returns the nil interface and false if no variable exists under that key.
func (s *Set) LookupGlobal(key string) (val interface{}, found bool) {
	if s.restricted != nil {
		value, ok := s.restricted.globals[key]
		if !ok {
			return nil, false
		}
		snapshot, _, err := admitInputs(context.Background(), VarMap{key: value}, nil, s.restricted.input)
		if err != nil {
			panic(err)
		}
		return snapshot[key], true
	}
	s.gmx.RLock()
	defer s.gmx.RUnlock()
	val, found = s.globals[key]
	return
}

// AddGlobalFunc adds a global function into the Set,
// overriding any function previously set under the specified key.
// It returns the Set it was called on to allow for method chaining.
func (s *Set) AddGlobalFunc(key string, fn Func) *Set {
	return s.AddGlobal(key, fn)
}
