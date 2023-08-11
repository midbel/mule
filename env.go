package mule

import (
	"net/url"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/eval"
	"github.com/midbel/enjoy/value"
)

const (
	reqUri    = "requestUri"
	reqStatus = "requestStatus"
	reqName   = "requestName"
)

type MuleEnv struct {
	ctx env.Environ[string]
	env.Environ[value.Value]
}

func Combine(ctx env.Environ[string]) MuleEnv {
	return MuleEnv{
		ctx:     env.Immutable(ctx),
		Environ: env.EnclosedEnv[value.Value](eval.Default()),
	}
}

func (m MuleEnv) SetRequestURI(uri *url.URL) {
	m.Define(reqUri, value.CreateString(uri.String()), true)
}

func (m MuleEnv) SetRequestName(name string) {
	m.Define(reqName, value.CreateString(name), true)
}

func (m MuleEnv) SetResponseCode(code int) {
	f := value.CreateFloat(float64(code))
	m.Define(reqStatus, f, true)
}

func (m MuleEnv) Resolve(ident string) (value.Value, error) {
	v, err := m.Environ.Resolve(ident)
	if err == nil {
		return v, err
	}
	s, err := m.ctx.Resolve(ident)
	if err == nil {
		return value.CreateString(s), nil
	}
	return nil, err
}
