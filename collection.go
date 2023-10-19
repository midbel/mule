package mule

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/value"
)

type Info struct {
	Name     string
	Usage    string
	Help     string
	Version  string
	Disabled bool
}

type Collection struct {
	Info

	parent *Collection

	config      *tls.Config
	base        Word
	user        Word
	pass        Word
	env         env.Environ[string]
	headers     Bag
	query       Bag
	requests    []Request
	collections []*Collection

	afterEach  []value.Evaluable
	beforeEach []value.Evaluable
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

func (c *Collection) Collections() []string {
	var list []string
	for _, i := range c.collections {
		if i.Disabled {
			continue
		}
		list = append(list, i.Name)
		list = append(list, i.Collections()...)
	}
	return list
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
	if c.Disabled {
		return fmt.Errorf("%s: collection disabled", c.Name)
	}
	var (
		rest  string
		found bool
	)
	name, rest, found = strings.Cut(name, ".")
	if !found {
		q, err := c.GetRequest(name)
		if err != nil {
			return err
		}
		return c.execute(q, w)
	}
	other, err := c.GetCollection(name)
	if err != nil {
		return err
	}
	return other.Run(rest, w)
}

func (c *Collection) execute(q Request, w io.Writer) error {
	depends, err := q.Depends(c)
	if err != nil {
		return err
	}
	ctx, err := MuleContext(c)
	if err != nil {
		return err
	}
	for _, d := range depends {
		if err := c.Run(d, w); err != nil {
			return err
		}
	}
	res, err := q.Execute(ctx)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(w, res.Body)
	return nil
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
	other := c.collections[i]
	if other.headers != nil {
		other.headers = other.headers.Merge(c.headers)
	} else {
		other.headers = c.headers
	}
	return other, nil
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
	req = c.requests[i]
	if req.Disabled {
		return req, fmt.Errorf("%s: request disabled", name)
	}
	if c.base != nil {
		var ws compound
		ws = append(ws, c.base)
		if w, ok := req.location.(compound); ok {
			ws = append(ws, w...)
		} else {
			ws = append(ws, req.location)
		}
		req.location = ws
	}
	req.query = req.query.Merge(c.query)
	req.headers = req.headers.Merge(c.headers)
	return req, nil
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
