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
