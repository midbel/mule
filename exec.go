package mule

import (
	"io"
	"net/http"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/value"
)

type Executer interface {
	Execute(io.Writer, env.Environ[value.Value]) error
}

type Resolver interface {
	Find(string, Resolver) (Executer, error)
}

type Body interface {
	Open() (io.ReadCloser, error)
}

func ExecuterFromRequest(req Request, col *Collection, resolv Resolver) (Executer, error) {
	var (
		sg  single
		err error
	)
	sg.Name = req.Name
	sg.expect = expectNothing
	sg.req, err = req.getRequest(col.env)
	if err != nil {
		return nil, err
	}
	depends, err := req.Depends(col.env)
	if err != nil {
		return nil, err
	}
	for _, name := range  depends {
		e, err := resolv.Find(name, resolv)
		if err != nil {
			return nil, err
		}
		sg.deps = append(sg.deps, e)
	}
	return sg, nil
}

func ExecuterFromCollection(col *Collection, resolv Resolver) (Executer, error) {
	ch := chain{
		Name:       col.Name,
		before:     col.before,
		after:      col.after,
		beforeEach: col.beforeEach,
		afterEach:  col.afterEach,
	}
	for _, r := range col.requests {
		e, err := ExecuterFromRequest(r, col, resolv)
		if err != nil {
			return nil, err
		}
		ch.executers = append(ch.executers, e)
	}
	return ch, nil
}

type single struct {
	Name   string
	req    *http.Request
	deps   []Executer
	before []value.Evaluable
	after  []value.Evaluable

	expect ExpectFunc
}

func (s single) Execute(w io.Writer, ev env.Environ[value.Value]) error {
	res, err := http.DefaultClient.Do(s.req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err = s.expect(res); err != nil {
		return err
	}
	_, err = io.Copy(w, res.Body)
	return err
}

type chain struct {
	Name       string
	executers  []Executer
	beforeEach value.Evaluable
	before     value.Evaluable
	afterEach  value.Evaluable
	after      value.Evaluable
}

func (c chain) Execute(w io.Writer, ev env.Environ[value.Value]) error {
	c.executeScript(c.before, ev)
	for _, e := range c.executers {
		c.executeScript(c.beforeEach, ev)
		if err := e.Execute(w, ev); err != nil {
			return err
		}
		c.executeScript(c.afterEach, ev)
	}
	c.executeScript(c.after, ev)
	return nil
}

func (c chain) executeScript(e value.Evaluable, ev env.Environ[value.Value]) error {
	if e == nil {
		return nil
	}
	_, err := e.Eval(ev)
	return err
}
