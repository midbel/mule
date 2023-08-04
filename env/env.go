package env

import (
	"fmt"
)

type Env interface {
	Define(string, string)
	Resolve(string) (string, error)
}

type Environment struct {
	parent Env
	values map[string]string
}

func EmptyEnv() Env {
	return EnclosedEnv(nil)
}

func EnclosedEnv(parent Env) Env {
	return &Environment{
		parent: parent,
		values: make(map[string]string),
	}
}

func (e *Environment) Define(key, value string) {
	e.values[key] = value
}

func (e *Environment) Resolve(key string) (string, error) {
	v, ok := e.values[key]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(key)
	}
	return "", fmt.Errorf("%s: variable not defined", key)
}
