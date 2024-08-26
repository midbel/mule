package mule

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

type Environment interface {
	Define(string, Value)
	Resolve(string) (Value, error)
}

type Env struct {
	parent Environment
	values map[string]Value
}

func Empty() Environment {
	return Enclosed(nil)
}

func Enclosed(parent Environment) Environment {
	return &Env{
		parent: parent,
		values: make(map[string]Value),
	}
}

func (e *Env) Resolve(ident string) (Value, error) {
	vs, ok := e.values[ident]
	if ok {
		return vs, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, fmt.Errorf("%s: undefined variable", ident)
}

func (e *Env) Define(ident string, value Value) {
	e.values[ident] = value
}

type Common struct {
	Name string

	URL      Value
	Auth     Authorization
	Retry    Value
	Timeout  Value
	Redirect Value
	Body     Value

	Headers Set
	Query   Set
	Tls     *tls.Config
}

type Collection struct {
	Common
	Environment
	// scripts to be run
	BeforeAll  Value
	BeforeEach Value
	AfterAll   Value
	AfterEach  Value

	// collection of requests
	Requests    []*Request
	Collections []*Collection
}

func Open(file string) (*Collection, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	p, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

func Root() *Collection {
	return Make("", nil)
}

func Make(name string, parent Environment) *Collection {
	info := Common{
		Name: name,
	}
	return &Collection{
		Common:      info,
		Environment: Enclosed(nil),
	}
}

func (c Collection) Resolve(ident string) (Value, error) {
	switch {
	case c.URL != nil && ident == "url":
		return c.URL, nil
	default:
	}
	return c.Environment.Resolve(ident)
}

func (c *Collection) Execute() error {
	return nil
}

func (c *Collection) Run(name string, w io.Writer) error {
	var (
		rest  string
		found bool
	)
	name, rest, found = strings.Cut(name, ".")
	if !found {
		req, err := c.GetRequest(name)
		if err != nil {
			other, err := c.GetCollection(name)
			if err != nil {
				return err
			}
			return other.Execute()
		}
		return req.Execute(c)
	}
	other, err := c.GetCollection(name)
	if err != nil {
		return err
	}
	return other.Run(rest, w)
}

func (c *Collection) GetCollection(name string) (*Collection, error) {
	ix := slices.IndexFunc(c.Collections, func(other *Collection) bool {
		return other.Name == name
	})

	if ix < 0 {
		return nil, fmt.Errorf("%s: collection not found", name)
	}

	sub := *c.Collections[ix]

	sub.URL = getUrl(c.URL, sub.URL, sub)
	sub.Body = getValue(sub.Body, c.Body)
	sub.Headers = sub.Headers.Merge(c.Headers)
	sub.Query = sub.Query.Merge(c.Query)

	return &sub, nil
}

func (c *Collection) GetRequest(name string) (*Request, error) {
	ix := slices.IndexFunc(c.Requests, func(other *Request) bool {
		return other.Name == name
	})

	if ix < 0 {
		return nil, fmt.Errorf("%s: request not found", name)
	}

	req := *c.Requests[ix]

	req.URL = getUrl(c.URL, req.URL, c)
	req.Body = getValue(req.Body, c.Body)
	req.Headers = req.Headers.Merge(c.Headers)
	req.Query = req.Query.Merge(c.Query)

	return &req, nil
}

type Request struct {
	Common
	Method  string
	Depends []Value
	Before  Value
	After   Value
}

func (r *Request) Execute(env Environment) error {
	target, err := r.target(env)
	if err != nil {
		return err
	}

	var body io.Reader
	if r.Body != nil {
		b, err := r.Body.Expand(env)
		if err != nil {
			return err
		}
		body = strings.NewReader(b)
	}
	if r.Before != nil {

	}
	req, err := http.NewRequest(r.Method, target, body)
	if err != nil {
		return err
	}
	if req.Header, err = r.Headers.Headers(env); err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf(http.StatusText(res.StatusCode))
	}
	if r.After != nil {

	}
	return nil
}

func (r *Request) target(env Environment) (string, error) {
	target, err := r.URL.Expand(env)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}

	vs, err := r.Query.Query(env)
	if err != nil {
		return "", err
	}
	if u.RawQuery == "" {
		u.RawQuery = vs.Encode()
	}
	return u.String(), nil
}

type Body interface {
	Value
	Compressed() bool
	ContentType() string
}

type xmlBody struct {}

func (b xmlBody) Expand(env Environment) (string, error) {
	return "", nil
}

func (b xmlBody) Compressed() bool {
	return false
}

func (b xmlBody) ContentType() string {
	return "text/xml"
}

type jsonBody struct {}

func (b jsonBody) Expand(env Environment) (string, error) {
	return "", nil
}

func (b jsonBody) Compressed() bool {
	return false
}

func (b jsonBody) ContentType() string {
	return "application/json"
}

type octetstreamBody struct {}

func (b octetstreamBody) Expand(env Environment) (string, error) {
	return "", nil
}

func (b octetstreamBody) Compressed() bool {
	return false
}

func (b octetstreamBody) ContentType() string {
	return "application/octet-stream"
}

type textBody struct {}

func (b textBody) Expand(env Environment) (string, error) {
	return "", nil
}

func (b textBody) Compressed() bool {
	return false
}

func (b textBody) ContentType() string {
	return "text/plain"
}

type urlencodedBody struct {}

func (b urlencodedBody) Expand(env Environment) (string, error) {
	return "", nil
}

func (b urlencodedBody) Compressed() bool {
	return false
}

func (b urlencodedBody) ContentType() string {
	return "application/x-www-form-urlencoded"
}

type Value interface {
	Expand(Environment) (string, error)
}

type Authorization interface {
	Value
	Method() string
}

type basic struct {
	User Value
	Pass Value
}

func (b basic) Method() string {
	return "Basic"
}

func (b basic) Expand(env Environment) (string, error) {
	return "", nil
}

type bearer struct {
	Token Value
}

func (b bearer) Method() string {
	return "Bearer"
}

func (b bearer) Expand(env Environment) (string, error) {
	return b.Token.Expand(env)
}

func getUrl(left, right Value, env Environment) Value {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}

	if str, err := right.Expand(env); err == nil {
		u, err := url.Parse(str)
		if err == nil && u.Host != "" {
			return right
		}
	}

	var cs compound
	if c, ok := left.(compound); ok {
		cs = append(cs, c...)
	} else {
		cs = append(cs, left)
	}
	if c, ok := right.(compound); ok {
		cs = append(cs, c...)
	} else {
		cs = append(cs, right)
	}
	return cs
}

func getValue(left, right Value) Value {
	if left != nil {
		return left
	}
	return right
}

type literal string

func createLiteral(str string) Value {
	return literal(str)
}

func (i literal) Expand(_ Environment) (string, error) {
	return string(i), nil
}

type variable string

func createVariable(str string) Value {
	return variable(str)
}

func (v variable) Expand(e Environment) (string, error) {
	val, err := e.Resolve(string(v))
	if err != nil {
		return "", err
	}
	return val.Expand(e)
}

type compound []Value

func (c compound) Expand(e Environment) (string, error) {
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

type call struct {
	ident string
	args  []interface{}
}

func (c call) Expand(env Environment) (string, error) {
	switch c.ident {
	case "readfile":
	case "jsonify":
	case "xmlify":
	case "urlencoded":
		return c.getUrlEncoded(env)
	default:
		return "", fmt.Errorf("%s function unknown", c.ident)
	}
	return "", nil
}

func (c call) getUrlEncoded(env Environment) (string, error) {
	if len(c.args) != 1 {
		return "", fmt.Errorf("urlencoded: invalid number of argument")
	}
	if v, ok := c.args[0].(Value); ok {
		res, err := v.Expand(env)
		if err == nil {
			res = url.QueryEscape(res)
		}
		return res, err
	}
	s, ok := c.args[0].(Set)
	if !ok {
		return "", fmt.Errorf("urlencoded: unsupported argument type")
	}
	q, err := s.Query(env)
	if err != nil {
		return "", err
	}
	return q.Encode(), nil
}

type Set map[string][]Value

func (s Set) Headers(env Environment) (http.Header, error) {
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

func (s Set) Query(env Environment) (url.Values, error) {
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

func (s Set) UrlEncoded(env Environment) (io.ReadCloser, error) {
	q, err := s.Query(env)
	if err != nil {
		return nil, err
	}
	r := strings.NewReader(q.Encode())
	return io.NopCloser(r), nil
}

func (s Set) Json(env Environment) (io.ReadCloser, error) {
	return nil, nil
}

func (s Set) Xml(env Environment) (io.ReadCloser, error) {
	return nil, nil
}
