package mule

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

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
	config  *tls.Config

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
		headers: Standard(),
		query:   Standard(),
	}
}

func (r Request) Execute(ctx *Context) (*http.Response, error) {
	req, err := r.Prepare(ctx.root)
	if err != nil {
		return nil, err
	}
	if req.Body != nil {
		defer req.Body.Close()
	}

	ctx.RegisterProp("request", createRequestValue(req))
	ctx.RegisterProp("response", value.Undefined())

	mule := muleEnv(ctx)
	mule.Define(reqUri, value.CreateString(req.URL.String()), true)
	mule.Define(reqName, value.CreateString(r.Name), true)

	if err := r.executeBefore(ctx.root, mule); err != nil {
		return nil, err
	}
	var (
		elapsed time.Duration
		client  = r.getClient(ctx.root.config)
		now     = time.Now()
	)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	elapsed = time.Since(now)
	defer res.Body.Close()

	ctx.RegisterProp("response", createResponseValue(res))

	var (
		tmp bytes.Buffer
		str bytes.Buffer
	)
	if _, err := io.Copy(io.MultiWriter(&tmp, &str), res.Body); err != nil {
		return nil, err
	}
	body := strings.TrimSpace(str.String())
	mule.Define(reqDuration, value.CreateFloat(elapsed.Seconds()), true)
	mule.Define(resStatus, value.CreateFloat(float64(res.StatusCode)), true)
	mule.Define(resBody, value.CreateString(body), true)
	if err := r.executeAfter(ctx.root, mule); err != nil {
		return nil, err
	}
	res.Body = io.NopCloser(&tmp)
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
	if r.user == nil && root.user != nil {
		r.user = root.user
	}
	if r.pass == nil && root.pass != nil {
		r.pass = root.pass
	}
	req, err := r.getRequest(root)
	if err != nil {
		return nil, err
	}
	return req, r.setHeaders(req, root)
}

func (r Request) getClient(root *tls.Config) http.Client {
	var client http.Client
	if cfg := r.getTLS(root); cfg != nil {
		client.Transport = &http.Transport{
			TLSClientConfig: cfg,
		}
	}
	return client
}

func (r Request) getTLS(parent *tls.Config) *tls.Config {
	if r.config != nil {
		return r.config
	}
	return parent
}

func (r Request) getRequest(root *Collection) (*http.Request, error) {
	var body io.Reader
	if r.body != nil {
		tmp, err := r.body.Open()
		if err != nil {
			return nil, err
		}
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

func (r Request) executeScripts(scripts []value.Evaluable, ctx env.Environ[value.Value]) error {
	for _, s := range scripts {
		if _, err := s.Eval(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r Request) executeBefore(root *Collection, ctx env.Environ[value.Value]) error {
	tmp := slices.Clone(root.beforeEach)
	if r.before != nil {
		tmp = append(tmp, r.before)
	}
	return r.executeScripts(tmp, ctx)
}

func (r Request) executeAfter(root *Collection, ctx env.Environ[value.Value]) error {
	tmp := slices.Clone(root.afterEach)
	if r.after != nil {
		tmp = append(tmp, r.after)
	}
	return r.executeScripts(tmp, ctx)
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
