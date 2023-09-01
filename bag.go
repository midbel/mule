package mule

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/midbel/enjoy/env"
)

type Bag interface {
	Add(string, Word)
	Set(string, Word)
	Clone() Bag
	Merge(Bag) Bag

	Cookie(env.Environ[string]) (*http.Cookie, error)
	Header(env.Environ[string]) (http.Header, error)
	Values(env.Environ[string]) (url.Values, error)
	ValuesWith(env.Environ[string], url.Values) (url.Values, error)

	pairs() []pair
}

type pair struct {
	Key  string
	List []Word
}

type stdBag map[string][]Word

func Standard() Bag {
	return make(stdBag)
}

func (b stdBag) Add(key string, value Word) {
	b[key] = append(b[key], value)
}

func (b stdBag) Set(key string, value Word) {
	b[key] = []Word{value}
}

func (b stdBag) Clone() Bag {
	g := make(stdBag)
	for k, vs := range b {
		g[k] = append(g[k], vs...)
	}
	return g
}

func (b stdBag) Merge(other Bag) Bag {
	g := make(stdBag)
	maps.Copy(g, b)

	for _, p := range other.pairs() {
		if _, ok := g[p.Key]; ok {
			continue
		}
		g[p.Key] = append(g[p.Key], p.List...)
	}
	return g
}

func (b stdBag) Header(e env.Environ[string]) (http.Header, error) {
	all := make(http.Header)
	for k, vs := range b {
		for i := range vs {
			str, err := vs[i].Expand(e)
			if err != nil {
				return nil, err
			}
			all.Add(k, str)
		}
	}
	return all, nil
}

func (b stdBag) Values(e env.Environ[string]) (url.Values, error) {
	all := make(url.Values)
	for k, vs := range b {
		for i := range vs {
			str, err := vs[i].Expand(e)
			if err != nil {
				return nil, err
			}
			all.Add(k, str)
		}
	}
	return all, nil
}

func (b stdBag) ValuesWith(e env.Environ[string], other url.Values) (url.Values, error) {
	all, err := b.Values(e)
	if err != nil {
		return nil, err
	}
	for k, vs := range other {
		all[k] = append(all[k], vs...)
	}
	return all, nil
}

func (b stdBag) Cookie(e env.Environ[string]) (*http.Cookie, error) {
	var (
		cook http.Cookie
		err  error
	)
	for k, vs := range b {
		if len(vs) == 0 {
			continue
		}
		switch k {
		case "name":
			cook.Name, err = vs[0].Expand(e)
		case "value":
			cook.Value, err = vs[0].Expand(e)
		case "path":
			cook.Path, err = vs[0].Expand(e)
		case "domain":
			cook.Domain, err = vs[0].Expand(e)
		case "expires":
		case "max-age":
			cook.MaxAge, err = vs[0].ExpandInt(e)
		case "secure":
			cook.Secure, err = vs[0].ExpandBool(e)
		case "http-only":
			cook.HttpOnly, err = vs[0].ExpandBool(e)
		default:
			return nil, fmt.Errorf("%s: invalid cookie property")
		}
		if err != nil {
			return nil, err
		}
	}
	return &cook, nil
}

func (b stdBag) pairs() []pair {
	var list []pair
	for k, vs := range b {
		p := pair{
			Key:  k,
			List: slices.Clone(vs),
		}
		list = append(list, p)
	}
	return list
}

type namedBag struct {
	name string
	Bag
}

type frozenBag struct {
	Bag
}

func Freeze(b Bag) Bag {
	if _, ok := b.(frozenBag); ok {
		return b
	}
	return frozenBag{
		Bag: b,
	}
}

func (b frozenBag) Merge(_ Bag) Bag {
	return b
}
