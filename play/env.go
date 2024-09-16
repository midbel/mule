package play

import (
	"errors"
	"fmt"

	"github.com/midbel/mule/environ"
)

type ptr struct {
	Ident string
	env   environ.Environment[Value]
}

func ptrValue(ident string, env environ.Environment[Value]) Value {
	return ptr{
		Ident: ident,
		env:   env,
	}
}

func (_ ptr) True() Value {
	return getBool(true)
}

type envValue struct {
	Const bool
	Value
}

func constValue(val Value) Value {
	return createValueForEnv(val, true)
}

func letValue(val Value) Value {
	return createValueForEnv(val, false)
}

func createValueForEnv(val Value, ro bool) Value {
	if _, ok := val.(envValue); ok {
		return val
	}
	return envValue{
		Value: val,
		Const: ro,
	}
}

var (
	ErrFrozen = errors.New("read only")
	ErrExport = errors.New("symbol not exported")
)

type frozenEnv struct {
	environ.Environment[Value]
}

func Freeze(env environ.Environment[Value]) environ.Environment[Value] {
	return &frozenEnv{
		Environment: env,
	}
}

func (e *frozenEnv) Define(_ string, _ Value) error {
	return ErrFrozen
}

type Env struct {
	parent environ.Environment[Value]
	values map[string]Value
}

func Combine(es ...environ.Environment[Value]) environ.Environment[Value] {
	if len(es) == 0 {
		return Empty()
	}
	return Empty()
}

func Empty() environ.Environment[Value] {
	return Enclosed(nil)
}

func Enclosed(parent environ.Environment[Value]) environ.Environment[Value] {
	return &Env{
		parent: parent,
		values: make(map[string]Value),
	}
}

func (e *Env) Clone() environ.Environment[Value] {
	return e
}

func (e *Env) Define(ident string, value Value) error {
	v, err := e.Resolve(ident)
	if err == nil {
		x, ok := v.(envValue)
		if ok && x.Const {
			return fmt.Errorf("%s: %w", ident, ErrConst)
		}
	}
	if e.parent != nil {
		return e.parent.Define(ident, value)
	}
	e.values[ident] = value
	return nil
}

func (e *Env) Resolve(ident string) (Value, error) {
	v, err := e.resolve(ident)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (e *Env) resolve(ident string) (Value, error) {
	if v, ok := e.values[ident]; ok {
		if p, ok := v.(ptr); ok {
			return p.env.Resolve(p.Ident)
		}
		if ev, ok := v.(envValue); ok {
			return ev.Value, nil
		}
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, e.undefined(ident)
}

func (e *Env) undefined(ident string) error {
	return fmt.Errorf("%s: %w", ident, environ.ErrDefined)
}

func (e *Env) unexported(ident string) error {
	return fmt.Errorf("%s: %w", ident, ErrExport)
}
