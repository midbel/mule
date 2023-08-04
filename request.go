package mule

import (
	"io"
	"fmt"
	"strings"
	"net/http"
	"net/url"
)

type Request struct {
	Info
	Order   int
	Default bool

	method  string
	depends []string
	retry   int
	timeout int

	location *url.URL
	user     string
	pass     string
	query    url.Values
	headers  http.Header
	body     string

	cookies []http.Cookie

	expect func(*http.Response) error
}

func Prepare(name, method string) Request {
	info := Info{
		Name: name,
	}
	return Request{
		Info:    info,
		method:  method,
		expect:  expectNothing,
		headers: make(http.Header),
		query:   make(url.Values),
	}
}

func (r Request) Prepare() (*http.Request, error) {
	var body io.Reader
	if len(r.body) == 0 {
		body = strings.NewReader(r.body)
	}
	req, err := http.NewRequest(r.method, r.location.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header = r.headers.Clone()
	raw := req.URL.Query()
	for k, vs := range r.query {
		for _, v := range vs {
			raw.Add(k, v)
		}
	}
	req.URL.RawQuery = raw.Encode()
	if req.Header.Get("Authorization") == "" && r.user != "" && r.pass != "" {
		req.SetBasicAuth(r.user, r.pass)
	}

	for _, c := range r.cookies {
		if err := c.Valid(); err != nil {
			return nil, err
		}
		req.AddCookie(&c)
	}

	return req, nil
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
