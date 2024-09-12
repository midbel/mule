package play

import (
	"errors"
	"fmt"

	"github.com/midbel/mule/environ"
)

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

var ErrFrozen = errors.New("read only")

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

type ReadOnlyValue interface {
	Value
	ReadOnly() bool
}

type ptr struct {
	Ident string
	Value
}

func ptrValue(ident string, value Value) Value {
	return ptr{
		Ident: ident,
		Value: value,
	}
}

type Env struct {
	parent environ.Environment[Value]
	values map[string]Value
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

func (e *Env) Resolve(ident string) (Value, error) {
	v, ok := e.values[ident]
	if ok {
		if p, ok := v.(ptr); ok {
			e, ok := p.Value.(environ.Environment[Value])
			if ok {
				return e.Resolve(p.Ident)
			}
		}
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, fmt.Errorf("%s: %w", ident, environ.ErrDefined)
}

func (e *Env) Define(ident string, value Value) error {
	v, err := e.Resolve(ident)
	if err == nil {
		x, ok := v.(envValue)
		if ok && x.Const {
			return fmt.Errorf("%s: %w", ident, ErrConst)
		}
		if p, ok := v.(ptr); ok {
			v = p.Value
		}
		if r, ok := v.(ReadOnlyValue); ok && r.ReadOnly() {
			return fmt.Errorf("%s: %w", ident, ErrConst)
		}
	}
	e.values[ident] = value
	return nil
}
