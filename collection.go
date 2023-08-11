package mule

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/midbel/enjoy/env"
)

type Info struct {
	Name    string
	Summary string
	Desc    string
}

type Collection struct {
	Info

	parent *Collection

	env         env.Environ[string]
	headers     Bag
	query       Bag
	requests    []Request
	collections []*Collection
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

func (c *Collection) Execute(name string, w io.Writer) error {
	r, err := c.Find(name)
	if err != nil {
		return err
	}
	others, err := r.Depends(c)
	if err != nil {
		return err
	}
	for _, n := range others {
		if err := c.Execute(n, w); err != nil {
			return err
		}
	}

	req, err := r.Prepare(c)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.Copy(w, res.Body)
	return err
}

func (c *Collection) Find(name string) (Request, error) {
	var (
		rest  string
		found bool
		req   Request
	)
	name, rest, found = strings.Cut(name, ".")
	if !found {
		req, err := c.GetRequest(name)
		if err != nil {
			return req, err
		}
		return req, nil
	}
	sub, err := c.GetCollection(name)
	if err != nil {
		return req, err
	}
	return sub.Find(rest)
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
