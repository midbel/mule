package play

import (
	"github.com/midbel/mule/environ"
)

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

func (m *module) GetExportedValues() []string {
	return nil
}

func (m *module) GetDefaultExport() string {
	return ""
}
