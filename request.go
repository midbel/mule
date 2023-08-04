package mule

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/midbel/mule/env"
)

type Request struct {
	Info
	Order   int
	Default bool

	method  string
	depends []Word
	retry   Word
	timeout Word

	location Word
	user     Word
	pass     Word
	query    Bag
	headers  Bag
	body     Word

	cookies []Bag

	// cookies []http.Cookie
	// expect func(*http.Response) error
}

func Prepare(name, method string) Request {
	info := Info{
		Name: name,
	}
	return Request{
		Info:    info,
		method:  method,
		headers: make(Bag),
		query:   make(Bag),
	}
}

func (r Request) Depends(ev env.Env) ([]string, error) {
	var list []string
	for i := range r.depends {
		str, err := r.depends[i].Expand(ev)
		if err != nil {
			return nil, err
		}
		list = append(list, str)
	}
	return list, nil
}

func (r Request) Prepare(ev env.Env) (*http.Request, error) {
	req, err := r.getRequest(ev)
	if err != nil {
		return nil, err
	}
	return req, r.setHeaders(req, ev)
}

func (r Request) getRequest(ev env.Env) (*http.Request, error) {
	var body io.Reader
	if r.body != nil {
		tmp, err := r.body.Expand(ev)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(tmp)
	}
	uri, err := r.location.ExpandURL(ev)
	if err != nil {
		return nil, err
	}
	return http.NewRequest(r.method, uri.String(), body)
}

func (r Request) setHeaders(req *http.Request, ev env.Env) error {
	hdr, err := r.headers.Header(ev)
	if err != nil {
		return err
	}
	req.Header = hdr
	if hdr.Get("Authorization") == "" && r.user != nil && r.pass != nil {
		u, err := r.user.Expand(ev)
		if err != nil {
			return err
		}
		p, err := r.pass.Expand(ev)
		if err != nil {
			return err
		}
		req.SetBasicAuth(u, p)
	}
	return r.attachCookies(req, ev)
}

func (r Request) attachCookies(req *http.Request, ev env.Env) error {
	for _, c := range r.cookies {
		cook, err := c.Cookie(ev)
		if err != nil {
			return err
		}
		if err = cook.Valid(); err != nil {
			return err
		}
		req.AddCookie(cook)
	}
	return nil
}

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
	return b
}

func (b Bag) Header(e env.Env) (http.Header, error) {
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

func (b Bag) Values(e env.Env) (url.Values, error) {
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

func (b Bag) Cookie(e env.Env) (*http.Cookie, error) {
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

type ExpectFunc func(*http.Response) error

func expectNothing(_ *http.Response) error {
	return nil
}

func expectCode(code int) (ExpectFunc, error) {
	if code < 100 || code >= 599 {
		return nil, fmt.Errorf("http status code out of range")
	}
	return func(r *http.Response) error {
		if r.StatusCode == code {
			return nil
		}
		return fmt.Errorf("expected %d status code! got %d", code, r.StatusCode)
	}, nil
}

func expectCodeRange(ident string) (ExpectFunc, error) {
	var fc, tc int
	switch ident {
	case "info":
		fc, tc = 100, 199
	case "success":
		fc, tc = 200, 299
	case "redirect":
		fc, tc = 300, 399
	case "bad-request":
		fc, tc = 400, 499
	case "server-error":
		fc, tc = 500, 599
	default:
		return nil, fmt.Errorf("%s: not recognized")
	}
	return func(r *http.Response) error {
		if r.StatusCode >= fc && r.StatusCode <= tc {
			return nil
		}
		return fmt.Errorf("expected status code in range %d - %d! got %d", fc, tc, r.StatusCode)
	}, nil
}

func mergeHeaders(fst, snd http.Header) http.Header {
	if len(fst) > 0 || len(snd) == 0 {
		return fst
	}
	return snd
}

func mergeQuery(fst, snd url.Values) url.Values {
	if len(fst) > 0 || len(snd) == 0 {
		return fst
	}
	return snd
}
