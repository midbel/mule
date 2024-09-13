package play

import (
	"errors"
	"fmt"

	"github.com/midbel/mule/environ"
)

type envValue struct {
	Const    bool
	Exported bool
	Value
}

func exportLetValue(val Value) Value {
	e := envValue{
		Value:    val,
		Const:    false,
		Exported: true,
	}
	return e
}

func exportConstValue(val Value) Value {
	e := envValue{
		Value:    val,
		Const:    true,
		Exported: true,
	}
	return e
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

func (e *Env) Exports(ident string) bool {
	v, ok := e.values[ident]
	if !ok {
		return false
	}
	x, ok := v.(envValue)
	if ok && x.Exported {
		return true
	}
	return false
}

func (e *Env) resolve(ident string) (Value, error) {
	v, ok := e.values[ident]
	if ok {
		if e, ok := v.(envValue); ok {
			return e.Value, nil
		}
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, fmt.Errorf("%s: %w", ident, environ.ErrDefined)
}
