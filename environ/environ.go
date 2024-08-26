package environ

import "fmt"

type Environment[T any] interface {
	Define(string, T)
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

func (e *Env[T]) Resolve(ident string) (T, error) {
	vs, ok := e.values[ident]
	if ok {
		return vs, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	var t T
	return t, fmt.Errorf("%s: undefined variable", ident)
}

func (e *Env[T]) Define(ident string, value T) {
	e.values[ident] = value
}
