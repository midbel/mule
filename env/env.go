package env

import (
	"errors"
	"fmt"
)

var ErrNotDefined = errors.New("variable not defined")

type Env[T any] interface {
	Define(string, T)
	Assign(string, T) error
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
	return &environ[T]{
		parent: parent,
		values: make(map[string]T),
	}
}

func (e *environ[T]) Define(key string, value T) {
	e.values[key] = value
}

func (e *environ[T]) Assign(key string, value T) error {
	_, ok := e.values[key]
	if !ok && e.parent != nil {
		return e.parent.Assign(key, value)
	}
	if !ok {
		return fmt.Errorf("%s: %w", key, ErrNotDefined)
	}
	e.Define(key, value)
	return nil
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
