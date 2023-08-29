package mule

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/value"
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
	body     Body

	cookies []Bag
	expect  func(*http.Response) error

	before value.Evaluable
	after  value.Evaluable
}

func Prepare(name, method string) Request {
	info := Info{
		Name: name,
	}
	return Request{
		Info:    info,
		method:  method,
		expect:  expectNothing,
		headers: make(Bag),
		query:   make(Bag),
	}
}

func (r Request) Execute(root *Collection) (*http.Response, error) {
	req, err := r.Prepare(root)
	if err != nil {
		return nil, err
	}

	ctx := prepareContext(root)
	ctx.Define(reqUri, value.CreateString(req.URL.String()), true)
	ctx.Define(reqName, value.CreateString(r.Name), true)

	if r.before != nil {
		if _, err := r.before.Eval(ctx); err != nil {
			return nil, err
		}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	var (
		str strings.Builder
		buf bytes.Buffer
		ws  = io.MultiWriter(&str, &buf)
	)
	if _, err := io.Copy(ws, res.Body); err != nil {
		return nil, err
	}
	res.Body.Close()
	res.Body = io.NopCloser(&buf)

	if r.after != nil {
		ctx.Define(resBody, value.CreateString(str.String()), true)
		ctx.Define(reqStatus, value.CreateFloat(float64(res.StatusCode)), true)
		if _, err := r.after.Eval(ctx); err != nil {
			return nil, err
		}
	}
	return res, r.expect(res)
}

func (r Request) Depends(ev env.Environ[string]) ([]string, error) {
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

func (r Request) Prepare(root *Collection) (*http.Request, error) {
	req, err := r.getRequest(root)
	if err != nil {
		return nil, err
	}
	return req, r.setHeaders(req, root)
}

func (r Request) getRequest(root *Collection) (*http.Request, error) {
	var body io.Reader
	if r.body != nil {
		tmp, err := r.body.Open()
		if err != nil {
			return nil, err
		}
		defer tmp.Close()
		body = tmp
	}
	uri, err := r.location.ExpandURL(root)
	if err != nil {
		return nil, err
	}
	if uri.Host == "" && root.base != nil {
		parent, err := root.base.ExpandURL(root)
		if err != nil {
			return nil, err
		}
		uri.Host = parent.Host
		uri.Scheme = parent.Scheme
	}
	query, err := r.query.ValuesWith(root, uri.Query())
	if err != nil {
		return nil, err
	}
	uri.RawQuery = query.Encode()
	return http.NewRequest(r.method, uri.String(), body)
}

func (r Request) setHeaders(req *http.Request, ev env.Environ[string]) error {
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

func (r Request) attachCookies(req *http.Request, ev env.Environ[string]) error {
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

type Body interface {
	Open() (io.ReadCloser, error)
}

func PrepareBody(str string) (Body, error) {
	s, err := os.Stat(str)
	if err == nil && s.Mode().IsRegular() {
		return stringBody(str), nil
	}
	return stringBody(str), nil
}

type stringBody string

func (b stringBody) Open() (io.ReadCloser, error) {
	r := strings.NewReader(string(b))
	return io.NopCloser(r), nil
}

type fileBody string

func (b fileBody) Open() (io.ReadCloser, error) {
	return os.Open(string(b))
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
