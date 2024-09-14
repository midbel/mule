package play

import (
	"fmt"

	"github.com/midbel/mule/environ"
)

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

func (m *module) Resolve(ident string) (Value, error) {
	return m.Env.Resolve(ident)
}

func (m *module) Define(ident string, value Value) error {
	return m.Env.Define(ident, value)
}

func (m *module) GetExportedValue(ident string) (Value, error) {
	if err := m.isExported(ident); err != nil {
		return nil, err
	}
	return m.Env.Resolve(ident)
}

func (m *module) Call(ident string, args []Value) (Value, error) {
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
		return m.Env.Resolve(name)
	}
}

func (m *module) isExported(ident string) error {
	e, ok := m.Env.(interface{ Exports(string) error })
	if !ok {
		return fmt.Errorf("%s: %w", ident, ErrExport)
	}
	return e.Exports(ident)
}
