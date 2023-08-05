package env

import (
	"fmt"
	"errors"
)

var ErrNotDefined = errors.New("variable not defined")

type Env[T any] interface {
	Define(string, T)
	Resolve(string) (T, error)
}

type environ[T any] struct {
	parent Env[T]
	values map[string]T
}

func EmptyEnv[T any]() Env[T] {
	return EnclosedEnv[T](nil)
}

func EnclosedEnv[T any](parent Env[T]) Env[T] {
	return &environ[T] {
		parent: parent,
		values: make(map[string]T),
	}
}

func (e *environ[T]) Define(key string, value T) {
	e.values[key] = value
}

func (e *environ[T]) Resolve(key string) (T, error) {
	v, ok := e.values[key]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(key)
	}
	return v, fmt.Errorf("%s: %w", key, ErrNotDefined)
}
