package mule

import (
	"fmt"
	"os"
	"strings"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/eval"
	"github.com/midbel/enjoy/value"
)

const (
	reqUri    = "requestUri"
	reqName   = "requestName"
	resStatus = "responseStatus"
	resBody   = "responseBody"
)

type Context struct {
	value.Global
	root *Collection
}

func MuleEnv(ctx *Context) env.Environ[value.Value] {
	top := eval.Default()
	sub := env.EnclosedEnv[value.Value](top)
	sub.Define("mule", ctx, true)

	return env.EnclosedEnv[value.Value](env.Immutable(sub))
}

func PrepareContext(root *Collection) *Context {
	obj := Context{
		Global: value.CreateGlobal("mule"),
		root:   root,
	}
	obj.RegisterProp("variables", createMuleVars(root))
	obj.RegisterProp("environ", createEnvVars())

	return &obj
}

func (c *Context) Get(prop string) (value.Value, error) {
	switch prop {
	case "collections":
		var list []value.Value
		for _, c := range c.root.Collections() {
			list = append(list, value.CreateString(c))
		}
		return value.CreateArray(list), nil
	default:
		return c.Global.Get(prop)
	}
}

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
	fmt.Println("call envVars", fn)
	switch fn {
	case "get":
		return v.Get(args[0].String())
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
