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
	User     Value
	Pass     Value
	Retry    Value
	Timeout  Value
	Redirect Value

	Headers Set
	Query   Set
	Tls     *tls.Config
	Body    io.Reader
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
	sub.User = getValue(c.User, sub.User)
	sub.Pass = getValue(c.Pass, sub.Pass)
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
	req.User = getValue(req.User, c.User)
	req.Pass = getValue(req.Pass, c.Pass)
	req.Headers = req.Headers.Merge(c.Headers)
	req.Query = req.Query.Merge(c.Query)

	return &req, nil
}

type Request struct {
	Common
	Body    Value
	Method  string
	Depends []Value
	Before  Value
	After   Value
}

func (r *Request) Execute(env Environment) error {
	str, err := r.URL.Expand(env)
	if err != nil {
		return err
	}
	fmt.Println(">>>", str)
	return nil
}

type Body interface {
	Open() (io.ReadCloser, error)
}

func getBody(str string) Body {
	s, err := os.Stat(str)
	if err == nil && s.Mode().IsRegular() {
		return fileBody(str)
	}
	return stringBody(str)
}

type stringBody string

func (b stringBody) Open() (io.ReadCloser, error) {
	str := strings.NewReader(string(b))
	return io.NopCloser(str), nil
}

type fileBody string

func (b fileBody) Open() (io.ReadCloser, error) {
	return os.Open(string(b))
}

type Value interface {
	Expand(Environment) (string, error)
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

type Set map[string][]Value

func (s Set) Headers() http.Header {
	return nil
}

func (s Set) Query() url.Values {
	return nil
}

func (s Set) Merge(other Set) Set {
	return s
}
