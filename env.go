package mule

import (
	"os"
	"strings"

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

type envVars struct{}

func createEnvVars() value.Value {
	return envVars{}
}

func (_ envVars) True() bool {
	return true
}

func (_ envVars) Type() string {
	return "object"
}

func (_ envVars) String() string {
	return "<environ>"
}

func (v envVars) Get(prop string) (value.Value, error) {
	n := strings.ToUpper(prop)
	s := os.Getenv(n)
	return value.CreateString(s), nil
}

func (v envVars) Call(fn string, args []value.Value) (value.Value, error) {
	switch fn {
	case "get":
		n := strings.ToUpper(args[0].String())
		s := os.Getenv(n)
		return value.CreateString(s), nil
	default:
		return nil, value.ErrOperation
	}
}

type muleVars struct {
	context env.Environ[string]
}

func createMuleVars(ev env.Environ[string]) value.Value {
	return muleVars{
		context: ev,
	}
}

func (_ muleVars) True() bool {
	return true
}

func (_ muleVars) Type() string {
	return "object"
}

func (_ muleVars) String() string {
	return "<variables>"
}

func (v muleVars) Call(fn string, args []value.Value) (value.Value, error) {
	switch fn {
	case "set":
		err := v.context.Assign(args[0].String(), args[1].String())
		return value.Undefined(), err
	case "get":
		s, err := v.context.Resolve(args[0].String())
		return value.CreateString(s), err
	default:
		return nil, value.ErrOperation
	}
}

func prepareMule(root *Collection) value.Value {
	obj := muleContext{
		Global: value.CreateGlobal("mule"),
		root:   root,
	}
	obj.RegisterProp("variables", createMuleVars(root))
	obj.RegisterProp("environ", createEnvVars())

	return &obj
}

func prepareContext(root *Collection) env.Environ[value.Value] {
	top := eval.Default()
	sub := env.EnclosedEnv[value.Value](top)
	return env.EnclosedEnv[value.Value](env.Immutable(sub))
}

type muleContext struct {
	value.Global
	root *Collection
}

func (c *muleContext) Get(prop string) (value.Value, error) {
	switch prop {
	case "collections":
		var list []value.Value
		for _, c := range c.root.Collections() {
			list = append(list, value.CreateString(c))
		}
		return value.CreateArray(list), nil
	default:
		return value.Undefined(), nil
	}
}
