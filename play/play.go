package play

import (
	"errors"
)

var (
	ErrBreak    = errors.New("break")
	ErrContinue = errors.New("continue")
	ErrReturn   = errors.New("return")
	ErrThrow    = errors.New("throw")
	ErrEval     = errors.New("node can not be evalualed in current context")
	ErrOp       = errors.New("unsupported operation")
	ErrType     = errors.New("incompatible type")
	ErrZero     = errors.New("division by zero")
	ErrConst    = errors.New("constant variable can not be reassigned")
	ErrIndex    = errors.New("index out of bound")
	ErrArgument = errors.New("invalid number of arguments")
	ErrImpl     = errors.New("not yet implemented")
)

type Value interface {
	True() Value
}

type Iterator interface {
	List() []Value
	Return()
}

type Event struct {
	Name   string
	Detail Value
}

type EventHandler interface {
	Emit(string) error
	Register(string, Value) error
}

type defaultEventHandler struct {
	events map[string][]Value
}

func NewEventHandler() EventHandler {
	return &defaultEventHandler{
		events: make(map[string][]Value),
	}
}

func (h *defaultEventHandler) Register(name string, value Value) error {
	_, ok := value.(Function)
	if !ok {
		return ErrEval
	}
	h.events[name] = append(h.events[name], value)
	return nil
}

func (h *defaultEventHandler) Emit(name string) error {
	all := h.events[name]
	if len(all) == 0 {
		return nil
	}
	for _, a := range all {
		c, ok := a.(interface{ Call([]Value) (Value, error) })
		if !ok {
			continue
		}
		c.Call([]Value{})
	}
	return nil
}

func isTrue(val Value) bool {
	b, ok := val.True().(Bool)
	if !ok {
		return false
	}
	return b.value
}

func isEqual(fst Value, snd Value) bool {
	eq, ok := fst.(interface{ Equal(Value) (Value, error) })
	if !ok {
		return ok
	}
	res, _ := eq.Equal(snd)
	return isTrue(res)
}
