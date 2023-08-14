package mule

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	body     Word

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

func (r Request) Execute(ev env.Environ[string]) (*http.Response, error) {
	req, err := r.Prepare(ev)
	if err != nil {
		return nil, err
	}

	ev.Define(reqUri, req.URL.String(), true)
	ev.Define(reqName, r.Name, true)

	var rev env.Environ[value.Value]
	if r, ok := ev.(interface {
		Reverse() env.Environ[value.Value]
	}); ok {
		rev = r.Reverse()
	} else {
		rev = env.EmptyEnv[value.Value]()
	}

	if r.before != nil {
		if _, err := r.before.Eval(rev); err != nil {
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
		ev.Define(resBody, str.String(), true)
		ev.Define(reqStatus, strconv.Itoa(res.StatusCode), true)
		if _, err := r.after.Eval(rev); err != nil {
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

func (r Request) Prepare(ev env.Environ[string]) (*http.Request, error) {
	req, err := r.getRequest(ev)
	if err != nil {
		return nil, err
	}
	return req, r.setHeaders(req, ev)
}

func (r Request) getRequest(ev env.Environ[string]) (*http.Request, error) {
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
	query, err := r.query.ValuesWith(ev, uri.Query())
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
