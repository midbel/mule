package mule

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Env struct {
	values map[string][]string
}

func Environ() *Env {
	return &Env{
		values: make(map[string][]string),
	}
}

func (e *Env) Resolve(ident string) ([]string, error) {
	vs, ok := e.values[ident]
	if !ok {
		return nil, fmt.Errorf("%s: undefined variable", ident)
	}
	return vs, nil
}

func (e *Env) Define(ident string, values []string) {
	e.values[ident] = values
}

type Common struct {
	Name string

	Url     string
	User    string
	Pass    string
	Retry   int
	Timeout int

	Headers http.Header
	Query   url.Values
	Tls     *tls.Config
	Body    io.Reader
}

type Collection struct {
	Common
	Env *Env
	// scripts to be run
	BeforeAll  string
	BeforeEach string
	AfterAll   string
	AfterEach  string

	// collection of requests
	Requests []*Request

	parent      *Collection
	Collections []*Collection
}

func Open(file string) (*Collection, error) {
	return nil, nil
}

func Empty(name string) *Collection {
	return Enclosed(name, nil)
}

func Enclosed(name string, parent *Collection) *Collection {
	info := Common{
		Name: name,
	}
	return &Collection{
		Common: info,
		Env:    Environ(),
	}
}

func (c *Collection) Resolve(ident string) ([]string, error) {
	vs, err := c.Env.Resolve(ident)
	if err == nil {
		return vs, nil
	}
	if c.parent != nil {
		return c.parent.Resolve(ident)
	}
	return nil, err
}

func (c *Collection) Define(ident string, values []string) {
	c.Env.Define(ident, values)
}

type Request struct {
	Common
	Depends []string
	Expect  int
	Before  string
	After   string
}
