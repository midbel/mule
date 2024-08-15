package mule

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	return Root(), nil
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

type Request struct {
	Common
	Method  string
	Depends []Value
	Before  Value
	After   Value
}

type Value interface {
	Expand(Environment) (string, error)
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
