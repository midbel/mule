package mule

import (
	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/eval"
	"github.com/midbel/enjoy/value"
)

const (
	reqUri    = "requestUri"
	reqStatus = "requestStatus"
	reqName   = "requestName"
	resBody   = "responseBody"
)

type scriptEnv struct {
	vars env.Environ[string]
	env.Environ[value.Value]
}

func Script(str env.Environ[string], val env.Environ[value.Value]) env.Environ[value.Value] {
	return scriptEnv{
		vars:    str,
		Environ: val,
	}
}

func (e scriptEnv) Reverse() env.Environ[string] {
	return Mule(e.vars, e.Environ)
}

func (e scriptEnv) Resolve(ident string) (value.Value, error) {
	v, err := e.Environ.Resolve(ident)
	if err == nil {
		return v, err
	}
	s, err := e.vars.Resolve(ident)
	if err != nil {
		return nil, err
	}
	return value.CreateString(s), nil
}

type muleEnv struct {
	vars env.Environ[value.Value]
	env.Environ[string]
}

func DefaultMule(str env.Environ[string]) env.Environ[string] {
	e := env.EnclosedEnv[value.Value](eval.Default())
	return Mule(str, e)
}

func Mule(str env.Environ[string], val env.Environ[value.Value]) env.Environ[string] {
	return muleEnv{
		vars:    val,
		Environ: str,
	}
}

func (e muleEnv) Reverse() env.Environ[value.Value] {
	return Script(e.Environ, e.vars)
}

func (e muleEnv) Resolve(ident string) (string, error) {
	s, err := e.Environ.Resolve(ident)
	if err == nil {
		return s, err
	}
	v, err := e.vars.Resolve(ident)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}
