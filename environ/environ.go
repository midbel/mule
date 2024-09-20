package environ

import (
	"errors"
	"fmt"
	"maps"
)

var (
	ErrDefined = errors.New("undefined variable")
	ErrExist   = errors.New("variable already exists")
)

type Environment[T any] interface {
	Define(string, T) error
	Resolve(string) (T, error)
}

type Env[T any] struct {
	parent Environment[T]
	values map[string]T
}

func Empty[T any]() Environment[T] {
	return Enclosed[T](nil)
}

func Enclosed[T any](parent Environment[T]) Environment[T] {
	return &Env[T]{
		parent: parent,
		values: make(map[string]T),
	}
}

func (e *Env[T]) Identifiers() []string {
	var all []string
	for k := range maps.Keys(e.values) {
		all = append(all, k)
	}
	return all
}

func (e *Env[T]) Resolve(ident string) (T, error) {
	vs, ok := e.values[ident]
	if ok {
		return vs, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	var t T
	return t, fmt.Errorf("%s: %w", ident, ErrDefined)
}

func (e *Env[T]) Define(ident string, value T) error {
	e.values[ident] = value
	return nil
}
