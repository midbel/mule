package play

import (
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

func (m *module) ReadOnly() bool {
	return true
}

func (m *module) Eval(n Node) (Value, error) {
	return eval(n, m.Env)
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
