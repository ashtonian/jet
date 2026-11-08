package jet

import (
	"html"
	"net/url"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

func builtinLower(a Arguments) reflect.Value { return builtinCase(a, false) }
func builtinUpper(a Arguments) reflect.Value { return builtinCase(a, true) }

func builtinCase(a Arguments, upper bool) reflect.Value {
	a.RequireNumOfArguments("case conversion", 1, 1)
	s := a.stringArg(0)
	st := a.runtime
	st.checkpoint(uint64(len(s)))
	var size int64
	changed := false
	for _, r := range s {
		mapped := unicode.ToLower(r)
		if upper {
			mapped = unicode.ToUpper(r)
		}
		changed = changed || mapped != r
		var ok bool
		size, ok = addValueSizes(size, int64(utf8.RuneLen(mapped)))
		if !ok {
			st.failValueSize()
		}
	}
	changed = changed || size != int64(len(s))
	if !changed {
		return reflect.ValueOf(s)
	}
	st.valueLength(size, true)
	st.checkpoint(uint64(len(s)) + uint64(size))
	if upper {
		return reflect.ValueOf(strings.ToUpper(s))
	}
	return reflect.ValueOf(strings.ToLower(s))
}

func builtinHasPrefix(a Arguments) reflect.Value { return builtinAffix(a, false) }
func builtinHasSuffix(a Arguments) reflect.Value { return builtinAffix(a, true) }

func builtinAffix(a Arguments, suffix bool) reflect.Value {
	a.RequireNumOfArguments("prefix/suffix", 2, 2)
	s, affix := a.stringArg(0), a.stringArg(1)
	n := len(affix)
	if n > len(s) {
		n = len(s)
	}
	a.runtime.checkpoint(uint64(n))
	if suffix {
		return reflect.ValueOf(strings.HasSuffix(s, affix))
	}
	return reflect.ValueOf(strings.HasPrefix(s, affix))
}

func builtinRepeat(a Arguments) reflect.Value {
	a.RequireNumOfArguments("repeat", 2, 2)
	s, count := a.stringArg(0), a.intArg(1)
	if count < 0 {
		a.Panicf("repeat(): count must be nonnegative")
	}
	if count == 0 || len(s) == 0 {
		return reflect.ValueOf("")
	}
	if count == 1 {
		return reflect.ValueOf(s)
	}
	size, ok := multiplyValueSize(int64(len(s)), int64(count))
	n := a.runtime.valueLength(size, ok)
	a.runtime.checkpoint(uint64(n))
	return reflect.ValueOf(strings.Repeat(s, count))
}

func builtinReplace(a Arguments) reflect.Value {
	a.RequireNumOfArguments("replace", 4, 4)
	s, old, replacement, count := a.stringArg(0), a.stringArg(1), a.stringArg(2), a.intArg(3)
	if count == 0 {
		return reflect.ValueOf(s)
	}
	st := a.runtime
	st.checkpoint(uint64(len(old)) + uint64(len(replacement)))
	if old == replacement {
		return reflect.ValueOf(s)
	}
	st.checkpoint(uint64(len(s)) + uint64(len(old)))
	var matches int
	if old == "" {
		n, ok := addValueSizes(int64(utf8.RuneCountInString(s)), 1)
		if !ok || uint64(n) > uint64(^uint(0)>>1) {
			st.failValueSize()
		}
		matches = int(n)
	} else {
		matches = strings.Count(s, old)
	}
	if count >= 0 && matches > count {
		matches = count
	}
	if matches == 0 {
		return reflect.ValueOf(s)
	}
	size, ok := replacedValueSize(int64(len(s)), int64(len(old)), int64(len(replacement)), int64(matches))
	n := st.valueLength(size, ok)
	st.checkpoint(uint64(len(s)) + uint64(n))
	return reflect.ValueOf(strings.Replace(s, old, replacement, count))
}

func builtinSplit(a Arguments) reflect.Value {
	a.RequireNumOfArguments("split", 2, 2)
	s, separator := a.stringArg(0), a.stringArg(1)
	st := a.runtime
	st.checkpoint(uint64(len(s)) + uint64(len(separator)))
	var count int
	if separator == "" {
		count = utf8.RuneCountInString(s)
	} else {
		n, ok := addValueSizes(int64(strings.Count(s, separator)), 1)
		if !ok || uint64(n) > uint64(^uint(0)>>1) {
			st.failValueSize()
		}
		count = int(n)
	}
	st.reserveSlots(count, int64(reflect.TypeOf("").Size()))
	st.checkpoint(uint64(len(s)) + uint64(count))
	return reflect.ValueOf(strings.Split(s, separator))
}

func builtinTrimSpace(a Arguments) reflect.Value {
	a.RequireNumOfArguments("trimSpace", 1, 1)
	s := a.stringArg(0)
	a.runtime.checkpoint(uint64(len(s)))
	return reflect.ValueOf(strings.TrimSpace(s))
}

func builtinHTML(a Arguments) reflect.Value {
	a.RequireNumOfArguments("html", 1, 1)
	s := a.stringArg(0)
	st := a.runtime
	st.checkpoint(uint64(len(s)))
	size, ok := htmlValueSize(s)
	if ok && size == int64(len(s)) {
		return reflect.ValueOf(s)
	}
	n := st.valueLength(size, ok)
	st.checkpoint(uint64(n))
	return reflect.ValueOf(html.EscapeString(s))
}

func builtinURL(a Arguments) reflect.Value {
	a.RequireNumOfArguments("url", 1, 1)
	s := a.stringArg(0)
	st := a.runtime
	st.checkpoint(uint64(len(s)))
	size, ok := queryValueSize(s)
	if ok && size == int64(len(s)) && !strings.Contains(s, " ") {
		return reflect.ValueOf(s)
	}
	n := st.valueLength(size, ok)
	st.checkpoint(uint64(n))
	return reflect.ValueOf(url.QueryEscape(s))
}
