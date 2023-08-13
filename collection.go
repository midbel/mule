package mule

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/value"
)

type Info struct {
	Name    string
	Summary string
	Desc    string
}

type Collection struct {
	Info

	parent *Collection

	base        Word
	env         env.Environ[string]
	headers     Bag
	query       Bag
	requests    []Request
	collections []*Collection

	after      value.Evaluable
	afterEach  value.Evaluable
	before     value.Evaluable
	beforeEach value.Evaluable
}

func Open(file string) (*Collection, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return NewParser(r).Parse()
}

func Empty(name string) *Collection {
	return Enclosed(name, nil)
}

func Enclosed(name string, parent *Collection) *Collection {
	info := Info{
		Name: name,
	}
	return &Collection{
		Info:   info,
		parent: parent,
		env:    env.EmptyEnv[string](),
	}
}

func (c *Collection) Path() []string {
	var (
		parts []string
		sub   = c
	)
	for sub != nil {
		parts = append(parts, sub.Name)
		sub = sub.parent
	}
	return parts
}

func (c *Collection) Run(name string, w io.Writer) error {
	x, err := c.Find(name, c)
	if err != nil {
		return err
	}
	return x.Execute(w, env.EmptyEnv[value.Value]())
}

func (c *Collection) Find(name string, resolv Resolver) (Executer, error) {
	var (
		rest  string
		found bool
	)
	name, rest, found = strings.Cut(name, ".")
	if !found {
		req, err := c.GetRequest(name)
		if err == nil {
			return c.FromRequest(req, resolv)
		}
		if c.Name == name {
			return c.FromSelf(resolv)
		}
		return nil, err
	}
	sub, err := c.GetCollection(name)
	if err != nil {
		return nil, err
	}
	sub.headers = sub.headers.Merge(c.headers)
	return sub.Find(rest, resolv)
}

func (c *Collection) FromRequest(req Request, resolv Resolver) (Executer, error) {
	req.headers = req.headers.Merge(c.headers)

	sg := single{
		Name:   req.Name,
		expect: expectNothing,
	}

	var err error
	if sg.req, err = req.Prepare(c.env); err != nil {
		return nil, err
	}
	if req.expect != nil {
		sg.expect = req.expect
	}

	depends, err := req.Depends(c.env)
	if err != nil {
		return nil, err
	}
	for _, name := range depends {
		e, err := resolv.Find(name, resolv)
		if err != nil {
			return nil, err
		}
		sg.deps = append(sg.deps, e)
	}
	return sg, nil
}

func (c *Collection) FromSelf(resolv Resolver) (Executer, error) {
	ch := chain{
		Name:       c.Name,
		before:     c.before,
		after:      c.after,
		beforeEach: c.beforeEach,
		afterEach:  c.afterEach,
	}
	for _, r := range c.requests {
		e, err := c.FromRequest(r, resolv)
		if err != nil {
			return nil, err
		}
		ch.executers = append(ch.executers, e)
	}
	return ch, nil
}

func (c *Collection) GetCollection(name string) (*Collection, error) {
	sort.Slice(c.collections, func(i, j int) bool {
		return c.collections[i].Name > c.collections[j].Name
	})
	i := sort.Search(len(c.collections), func(i int) bool {
		return c.collections[i].Name <= name
	})

	ok := i < len(c.collections) && c.collections[i].Name == name
	if !ok {
		return nil, fmt.Errorf("%s: collection not defined", name)
	}
	return c.collections[i], nil
}

func (c *Collection) GetRequest(name string) (Request, error) {
	sort.Slice(c.requests, func(i, j int) bool {
		return c.requests[i].Name > c.requests[j].Name
	})
	i := sort.Search(len(c.requests), func(i int) bool {
		return c.requests[i].Name <= name
	})
	var (
		req Request
		ok  = i < len(c.requests) && c.requests[i].Name == name
	)
	if !ok {
		return req, fmt.Errorf("%s: request not defined", name)
	}
	return c.requests[i], nil
}

func (c *Collection) Resolve(key string) (string, error) {
	v, err := c.env.Resolve(key)
	if err == nil {
		return v, err
	}
	if c.parent != nil {
		return c.parent.Resolve(key)
	}
	return "", fmt.Errorf("%s: variable not defined", key)
}

func (c *Collection) Define(key, value string, _ bool) error {
	c.env.Define(key, value, false)
	return nil
}

func (c *Collection) Assign(key, value string) error {
	return nil
}

func (c *Collection) AddRequest(req Request) {
	c.requests = append(c.requests, req)
}

func (c *Collection) AddCollection(col *Collection) {
	if col.parent == nil {
		col.parent = c
	}
	c.collections = append(c.collections, col)
}
