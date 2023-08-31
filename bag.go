package mule

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/midbel/enjoy/env"
)

type Bag map[string][]Word

func (b Bag) Add(key string, value Word) {
	b[key] = append(b[key], value)
}

func (b Bag) Set(key string, value Word) {
	b[key] = []Word{value}
}

func (b Bag) Clone() Bag {
	g := make(Bag)
	for k, vs := range b {
		g[k] = append(g[k], vs...)
	}
	return g
}

func (b Bag) Merge(other Bag) Bag {
	g := make(Bag)
	maps.Copy(g, b)
	for k, v := range other {
		if _, ok := g[k]; ok {
			continue
		}
		g[k] = append(g[k], v...)
	}
	return g
}

func (b Bag) Header(e env.Environ[string]) (http.Header, error) {
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

func (b Bag) Values(e env.Environ[string]) (url.Values, error) {
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

func (b Bag) ValuesWith(e env.Environ[string], other url.Values) (url.Values, error) {
	all, err := b.Values(e)
	if err != nil {
		return nil, err
	}
	for k, vs := range other {
		all[k] = append(all[k], vs...)
	}
	return all, nil
}

func (b Bag) Cookie(e env.Environ[string]) (*http.Cookie, error) {
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

type NamedBag struct {
	Name string
	Bag
}

type frozenBag struct {
	Bag
}
