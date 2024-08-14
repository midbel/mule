package mule

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Environment interface {
	Define(string, []string)
	Resolve(string) ([]string, error)
}

type Env struct {
	parent Environment
	values map[string][]string
}

func Empty() Environment {
	return Enclosed(nil)
}

func Enclosed(parent Environment) Environment {
	return &Env{
		parent: parent,
		values: make(map[string][]string),
	}
}

func (e *Env) Resolve(ident string) ([]string, error) {
	vs, ok := e.values[ident]
	if ok {
		return vs, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, fmt.Errorf("%s: undefined variable", ident)
}

func (e *Env) Define(ident string, values []string) {
	e.values[ident] = values
}

type Common struct {
	Name string

	URL     url.URL
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
	Environment
	// scripts to be run
	BeforeAll  string
	BeforeEach string
	AfterAll   string
	AfterEach  string

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
		Common: info,
	}
}

type Request struct {
	Common
	Method  string
	Depends []string
	Expect  int
	Before  string
	After   string
}
