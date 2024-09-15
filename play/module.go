package play

import (
	"fmt"

	"github.com/midbel/mule/environ"
)

type module struct {
	Name    string
	Attrs   *Object
	Env     environ.Environment[Value]
	Exports map[string]string
}

func createModule(ident string) *module {
	return &module{
		Name:    ident,
		Env:     Enclosed(Default()),
		Attrs:   createObject(),
		Exports: make(map[string]string),
	}
}

func (m *module) Export(ident, alias string, val Value) error {
	if _, ok := m.Exports[ident]; ok {
		return fmt.Errorf("%s: symbol already exported", ident)
	}
	m.Exports[alias] = ident
	if val == nil {
		_, err := m.Resolve(ident)
		return err
	}
	return m.Define(ident, val)
}

func (m *module) Import(ident string) (Value, error) {
	id, ok := m.Exports[ident]
	if !ok {
		return nil, fmt.Errorf("%s: %w", ident, ErrExport)
	}
	v, err := m.Env.Resolve(id)
	if err != nil {
		return nil, err
	}
	_ = v
	return ptrValue(id, m.Env), nil
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

func (m *module) Resolve(ident string) (Value, error) {
	return m.Env.Resolve(ident)
}

func (m *module) Define(ident string, value Value) error {
	return m.Env.Define(ident, value)
}

func (m *module) Call(ident string, args []Value) (Value, error) {
	if err := m.isExported(ident); err != nil {
		return nil, err
	}
	val, err := m.Env.Resolve(m.Exports[ident])
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
		return m.Env.Resolve(m.Exports[name])
	}
}

func (m *module) isExported(ident string) error {
	_, ok := m.Exports[ident]
	if !ok {
		return fmt.Errorf("%s: %w", ident, ErrExport)
	}
	return nil
}
