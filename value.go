package mule

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/midbel/mule/environ"
)

type Value interface {
	Expand(environ.Environment[Value]) (string, error)
	clone() Value
}

type literal string

func createLiteral(str string) Value {
	return literal(str)
}

func (i literal) clone() Value {
	return i
}

func (i literal) Expand(_ environ.Environment[Value]) (string, error) {
	return string(i), nil
}

type variable string

func createVariable(str string) Value {
	return variable(str)
}

func (v variable) clone() Value {
	return v
}

func (v variable) Expand(e environ.Environment[Value]) (string, error) {
	val, err := e.Resolve(string(v))
	if err != nil {
		return "", err
	}
	return val.Expand(e)
}

const (
	replaceFirst = 1 << iota
	replaceAll
	replaceSuffix
	replacePrefix
)

type replace struct {
	value  Value
	before Value
	after  Value
	op     int8
}

func (r replace) clone() Value {
	return r
}

func (r replace) Expand(e environ.Environment[Value]) (string, error) {
	value, err := r.value.Expand(e)
	if err != nil {
		return "", err
	}
	before, err := r.before.Expand(e)
	if err != nil {
		return "", err
	}
	after, err := r.after.Expand(e)
	if err != nil {
		return "", err
	}
	switch r.op {
	case replaceFirst:
		value = strings.Replace(value, before, after, 1)
	case replaceAll:
		value = strings.ReplaceAll(value, before, after)
	case replaceSuffix:
		tmp, ok := strings.CutSuffix(value, before)
		if ok {
			value = tmp + after
		}
	case replacePrefix:
		tmp, ok := strings.CutPrefix(value, before)
		if ok {
			value = before + tmp
		}
	}
	return value, nil
}

type substr struct {
	value Value
	start Value
	end   Value
}

func (s substr) clone() Value {
	return s
}

func (s substr) Expand(e environ.Environment[Value]) (string, error) {
	return "", nil
}

const (
	suffixTrim = 1 << iota
	prefixTrim
	suffixLongTrim
	prefixLongTrim
)

type trim struct {
	value Value
	word  Value
	op    int8
}

func (t trim) clone() Value {
	return t
}

func (t trim) Expand(e environ.Environment[Value]) (string, error) {
	return "", nil
	value, err := t.value.Expand(e)
	if err != nil {
		return "", err
	}
	word, err := t.word.Expand(e)
	if err != nil {
		return "", err
	}
	switch t.op {
	case suffixTrim:
		value = strings.TrimSuffix(value, word)
	case prefixTrim:
		value = strings.TrimPrefix(value, word)
	case suffixLongTrim:
		for strings.HasSuffix(value, word) {
			value = strings.TrimSuffix(value, word)
		}
	case prefixLongTrim:
		for strings.HasPrefix(value, word) {
			value = strings.TrimPrefix(value, word)
		}
	}
	return value, nil
}

const (
	lowerFirstCase = 1 << iota
	upperFirstCase
	lowerAllCase
	upperAllCase
)

type changecase struct {
	value Value
	op    int8
}

func (c changecase) clone() Value {
	return c
}

func (c changecase) Expand(e environ.Environment[Value]) (string, error) {
	value, err := c.value.Expand(e)
	if err != nil {
		return "", err
	}
	switch c.op {
	case lowerFirstCase:
	case upperFirstCase:
	case lowerAllCase:
		value = strings.ToLower(value)
	case upperAllCase:
		value = strings.ToUpper(value)
	}
	return "", nil
}

type compound []Value

func (c compound) clone() Value {
	var cs compound
	for i := range c {
		cs = append(cs, c[i].clone())
	}
	return cs
}

func (c compound) Expand(e environ.Environment[Value]) (string, error) {
	var parts []string
	for i := range c {
		v, err := c[i].Expand(e)
		if err != nil {
			return "", err
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, ""), nil
}

type Set map[string][]Value

func (s Set) Headers(env environ.Environment[Value]) (http.Header, error) {
	hs := make(http.Header)
	for k := range s {
		for _, v := range s[k] {
			str, err := v.Expand(env)
			if err != nil {
				return nil, err
			}
			hs.Add(k, str)
		}
	}
	return hs, nil
}

func (s Set) Map(env environ.Environment[Value]) (map[string]interface{}, error) {
	vs := make(map[string]interface{})
	for k := range s {
		var arr []string
		for _, v := range s[k] {
			str, err := v.Expand(env)
			if err != nil {
				return nil, err
			}
			arr = append(arr, str)
		}
		var dat interface{} = arr
		if len(arr) == 1 {
			dat = arr[0]
		}
		vs[k] = dat
	}
	return vs, nil
}

func (s Set) Query(env environ.Environment[Value]) (url.Values, error) {
	vs := make(url.Values)
	for k := range s {
		for _, v := range s[k] {
			str, err := v.Expand(env)
			if err != nil {
				return nil, err
			}
			vs.Add(k, str)
		}
	}
	return vs, nil
}

func (s Set) Merge(other Set) Set {
	var ns = make(Set)
	for k := range s {
		ns[k] = slices.Clone(s[k])
	}
	for k := range other {
		ns[k] = slices.Concat(ns[k], slices.Clone(other[k]))
	}
	return ns
}
