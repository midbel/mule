package play

import (
	"bytes"
	"fmt"
	"slices"
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
	var fn func([]Value) (Value, error)
	switch ident {
	case "at":
		fn = checkArity(1, a.at)
	case "concat":
		fn = checkArity(-1, a.concat)
	case "entries":
		fn = checkArity(0, a.entries)
	case "every":
		fn = checkArity(1, a.every)
	case "fill":
		fn = checkArity(3, a.fill)
	case "filter":
		fn = checkArity(1, a.filter)
	case "find":
		fn = checkArity(1, a.find)
	case "findIndex":
		fn = checkArity(1, a.findIndex)
	case "flat":
		fn = checkArity(1, a.flat)
	case "forEach":
		fn = checkArity(1, a.forEach)
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
	if fn == nil {
		return nil, ErrImpl
	}
	return fn(args)
}

func (a Array) List() []Value {
	return a.Values
}

func (a Array) Return() {
	return
}

func (a Array) at(args []Value) (Value, error) {
	ix, ok := args[0].(Float)
	if !ok {
		return nil, ErrType
	}
	at := int(ix.value)
	if at < 0 {
		at = len(a.Values) + at
	}
	if at >= 0 && at < len(a.Values) {
		return a.Values[at], nil
	}
	return Void{}, nil
}

func (a Array) concat(args []Value) (Value, error) {
	if len(args) == 0 {
		return a, nil
	}
	for i := range args {
		if other, ok := args[i].(Array); ok {
			a.Values = slices.Concat(a.Values, other.Values)
		} else {
			a.Values = append(a.Values, args[i])
		}
	}
	return a, nil
}

func (a Array) entries(args []Value) (Value, error) {
	return nil, nil
}

func (a Array) every(args []Value) (Value, error) {
	check, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	for i := range a.Values {
		args := []Value{
			a.Values[i],
			NewFloat(float64(i)),
			a,
		}
		ok, err := check.Call(args)
		if err != nil {
			return nil, err
		}
		if !isTrue(ok) {
			return getBool(false), nil
		}
	}
	return getBool(true), nil
}

func (a Array) fill(args []Value) (Value, error) {
	if len(args) == 0 {
		return nil, ErrOp
	}
	var (
		fill Value
		beg  int
		end  = len(a.Values) - 1
	)
	if len(args) >= 1 {
		fill = args[0]
	}
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		beg = int(x.value)
		if beg < 0 {
			beg = len(a.Values) + beg
		}
	}
	if len(args) >= 3 {
		x, ok := args[2].(Float)
		if !ok {
			return nil, ErrType
		}
		end = int(x.value)
		if end < 0 {
			end = len(a.Values) + end
		}
	}
	if end <= beg {
		return a, nil
	}
	for i := range a.Values[beg:end] {
		a.Values[beg+i] = fill
	}
	return a, nil
}

func (a Array) filter(args []Value) (Value, error) {
	keep, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	arr := createArray()
	for i := range a.Values {
		args := []Value{
			a.Values[i],
			NewFloat(float64(i)),
			a,
		}
		ok, err := keep.Call(args)
		if err != nil {
			return nil, err
		}
		if !isTrue(ok) {
			continue
		}
		arr.Values = append(arr.Values, a.Values[i])
	}
	return arr, nil
}

func (a Array) find(args []Value) (Value, error) {
	find, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	for i := range a.Values {
		args := []Value{
			a.Values[i],
			NewFloat(float64(i)),
			a,
		}
		ok, err := find.Call(args)
		if err != nil {
			return nil, err
		}
		if isTrue(ok) {
			return a.Values[i], nil
		}
	}
	return Void{}, nil
}

func (a Array) findIndex(args []Value) (Value, error) {
	find, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	for i := range a.Values {
		args := []Value{
			a.Values[i],
			NewFloat(float64(i)),
			a,
		}
		ok, err := find.Call(args)
		if err != nil {
			return nil, err
		}
		if isTrue(ok) {
			return NewFloat(float64(i)), nil
		}
	}
	return Void{}, nil
}

func (a Array) flat(args []Value) (Value, error) {
	var (
		depth   = 1
		flatten func(Value, int) []Value
	)
	if len(args) > 0 {
		d, ok := args[0].(Float)
		if !ok {
			return nil, ErrType
		}
		depth = int(d.value)
	}

	flatten = func(v Value, level int) []Value {
		arr, ok := v.(Array)
		if !ok || level < 0 {
			return []Value{v}
		}
		var vs []Value
		for i := range arr.Values {
			xs := flatten(arr.Values[i], level-1)
			vs = append(vs, xs...)
		}
		return vs
	}
	res := createArray()
	res.Values = flatten(a, depth)
	return res, nil
}

func (a Array) forEach(args []Value) (Value, error) {
	each, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	for i := range a.Values {
		args := []Value{
			a.Values[i],
			NewFloat(float64(i)),
			a,
		}
		_, err := each.Call(args)
		if err != nil {
			return nil, err
		}
	}
	return Void{}, nil
}
