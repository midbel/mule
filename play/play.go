package play

import (
	"errors"
	"math"

	"github.com/midbel/mule/environ"
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

func execParseInt(args []Value) (Value, error) {
	return nil, nil
}

func execParseFloat(args []Value) (Value, error) {
	return nil, nil
}

func execIsNaN(args []Value) (Value, error) {
	if len(args) != 1 {
		return getBool(true), nil
	}
	v, ok := args[0].(Float)
	if !ok {
		return getBool(false), nil
	}
	return getBool(math.IsNaN(v.value)), nil
}

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

type BuiltinFunc struct {
	Ident string
	Func  func([]Value) (Value, error)
}

func NewBuiltinFunc(ident string, fn func([]Value) (Value, error)) Value {
	return createBuiltinFunc(ident, fn)
}

func createBuiltinFunc(ident string, fn func([]Value) (Value, error)) Value {
	return BuiltinFunc{
		Ident: ident,
		Func:  fn,
	}
}

func (b BuiltinFunc) True() Value {
	return getBool(true)
}

func (b BuiltinFunc) Call(args []Value) (Value, error) {
	return b.Func(args)
}

type Parameter struct {
	Name string
	Value
}

type Function struct {
	Ident string
	Arrow bool
	Args  []Parameter
	Body  Node
	Env   environ.Environment[Value]
}

func (f Function) True() Value {
	return getBool(true)
}

func (f Function) Call(args []Value) (Value, error) {
	for i := range f.Args {
		var arg Value
		if i < len(args) {
			arg = args[i]
			if arg == nil || isNull(arg) || isUndefined(arg) {
				arg = f.Args[i].Value
			}
		} else {
			arg = f.Args[i].Value
		}
		if err := f.Env.Define(f.Args[i].Name, arg); err != nil {
			return nil, err
		}
	}
	arr := createArray()
	arr.Values = append(arr.Values, args...)
	f.Env.Define("arguments", arr)

	return eval(f.Body, f.Env)
}
