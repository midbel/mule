package play

import (
	"bytes"
	"fmt"
)

type Field struct {
	Value
	writable     bool
	enumerable   bool
	configurable bool
}

type Object struct {
	Fields map[Value]Value
	Frozen bool
}

func createObject() Object {
	return Object{
		Fields: make(map[Value]Value),
	}
}

func (o Object) String() string {
	var (
		buf bytes.Buffer
		ix  int
	)
	buf.WriteRune(lcurly)
	for k, v := range o.Fields {
		if ix > 0 {
			buf.WriteRune(comma)
			buf.WriteRune(space)
		}
		fmt.Fprint(&buf, k)
		buf.WriteRune(colon)
		buf.WriteRune(space)

		_, quoted := v.(String)
		if quoted {
			buf.WriteRune(dquote)
		}
		fmt.Fprint(&buf, v)
		if quoted {
			buf.WriteRune(dquote)
		}
		ix++
	}
	buf.WriteRune(rcurly)
	return buf.String()
}

func (o Object) True() Value {
	return getBool(len(o.Fields) != 0)
}

func (o Object) Not() Value {
	return getBool(len(o.Fields) == 0)
}

func (o Object) At(ix Value) (Value, error) {
	return o.Fields[ix], nil
}

func (o Object) Call(ident string, args []Value) (Value, error) {
	fn, ok := o.Fields[getString(ident)]
	if !ok {
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	call, ok := fn.(Function)
	if !ok {
		return nil, fmt.Errorf("%s: property not callable", ident)
	}
	if !call.Arrow {
		call.Env.Define("this", o)
	}
	return call.Call(args)
}

func (o Object) SetAt(prop, value Value) error {
	return o.Set(prop, value)
}

func (o Object) Set(prop, value Value) error {
	o.Fields[prop] = value
	return nil
}

func (o Object) Get(prop Value) (Value, error) {
	v, ok := o.Fields[prop]
	if !ok {
		var x Void
		return x, nil
	}
	return v, nil
}

func (o Object) Values() []Value {
	var vs []Value
	for k := range o.Fields {
		vs = append(vs, o.Fields[k])
	}
	return vs
}

type Array struct {
	Object
	Values []Value
}

func createArray() Array {
	return Array{
		Object: createObject(),
	}
}

func (a Array) String() string {
	var buf bytes.Buffer
	buf.WriteRune(lsquare)
	for i := range a.Values {
		if i > 0 {
			buf.WriteRune(comma)
			buf.WriteRune(space)
		}
		fmt.Fprint(&buf, a.Values[i])
	}
	buf.WriteRune(rsquare)
	return buf.String()
}

func (a Array) True() Value {
	return getBool(len(a.Values) != 0)
}

func (a Array) Not() Value {
	return getBool(len(a.Values) == 0)
}

func (a Array) At(ix Value) (Value, error) {
	x, ok := ix.(Float)
	if !ok {
		return nil, ErrEval
	}
	if x.value >= 0 && int(x.value) < len(a.Values) {
		return a.Values[int(x.value)], nil
	}
	return nil, ErrIndex
}

func (a Array) SetAt(ix Value, value Value) error {
	x, ok := ix.(Float)
	if !ok {
		return ErrEval
	}
	if x.value >= 0 && int(x.value) < len(a.Values) {
		a.Values[int(x.value)] = value
		return nil
	}
	return ErrIndex
}

func (a Array) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch name := str.String(); name {
	case "length":
		n := len(a.Values)
		return getFloat(float64(n)), nil
	default:
		return a.Object.Get(ident)
	}
}

func (a Array) Call(ident string, args []Value) (Value, error) {
	switch ident {
	case "at":
	case "concat":
	case "entries":
	case "every":
	case "filter":
	case "find":
	case "findIndex":
	case "flat":
	case "forEach":
	case "includes":
	case "indexOf":
	case "join":
	case "map":
	case "pop":
	case "push":
	case "reduce":
	case "reverse":
	case "shift":
	case "slice":
	case "splice":
	case "all":
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	return nil, ErrImpl
}

func (a Array) List() []Value {
	return a.Values
}

func (a Array) Return() {
	return
}
