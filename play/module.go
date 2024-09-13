package play

import (
	"errors"
	"fmt"

	"github.com/midbel/mule/environ"
)

type proxyValue struct {
	Value
	env environ.Environment[Value]
}

func (p proxyValue) Call(args []Value) (Value, error) {
	fn, ok := p.Value.(Function)
	if !ok {
		return nil, fmt.Errorf("variable is not callable")
	}
	fn.Env = Enclosed(p.env)
	return fn.Call(args)
}

var ErrExport = errors.New("symbol not exported")

type module struct {
	Name  string
	Attrs *Object
	Env   environ.Environment[Value]
}

func createModule(ident string) *module {
	return &module{
		Name:  ident,
		Env:   Enclosed(Default()),
		Attrs: createObject(),
	}
}

func (m *module) Type() string {
	return "module"
}

func (m *module) String() string {
	return "module"
}

func (m *module) True() Value {
	return getBool(true)
}

func (m *module) Call(ident string, args []Value) (Value, error) {
	if err := m.isExported(ident); err != nil {
		return nil, err
	}
	val, err := m.Env.Resolve(ident)
	if err != nil {
		return nil, err
	}
	fn, ok := val.(Function)
	if !ok {
		return nil, fmt.Errorf("%s is not callable", ident)
	}
	fn.Env = Enclosed(m.Env)
	return fn.Call(args)
}

func (m *module) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch name := str.String(); name {
	case "name":
		return getString(m.Name), nil
	default:
		if err := m.isExported(name); err != nil {
			return nil, err
		}
		return m.Env.Resolve(name)
	}
}

func (m *module) GetExportedValue(ident string) (Value, error) {
	if err := m.isExported(ident); err != nil {
		return nil, err
	}
	v, err := m.Env.Resolve(ident)
	if err != nil {
		return nil, err
	}
	v = proxyValue{
		Value: v,
		env:   Enclosed(Freeze(m.Env)),
	}
	return v, nil
}

func (m *module) isExported(ident string) error {
	e, ok := m.Env.(interface{ Exports(string) bool })
	if ok && e.Exports(ident) {
		return nil
	}
	return fmt.Errorf("%s: %w", ident, ErrExport)
}
