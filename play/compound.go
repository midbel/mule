package play

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/midbel/mule/jwt"
)

const (
	frozenObject = 1 << iota
	sealedObject
	lockedObject
)

type Field struct {
	Value
	writable     bool
	enumerable   bool
	configurable bool
}

func fieldByAssignment(value Value) Value {
	if _, ok := value.(Field); ok {
		return value
	}
	return Field{
		Value:        value,
		writable:     true,
		enumerable:   true,
		configurable: true,
	}
}

type Object struct {
	Fields map[Value]Value
	locked int
}

func NewObject() *Object {
	return createObject()
}

func createObject() *Object {
	return &Object{
		Fields: make(map[Value]Value),
	}
}

func (o *Object) Type() string {
	return "object"
}

func (o *Object) String() string {
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

func (o *Object) True() Value {
	return getBool(len(o.Fields) != 0)
}

func (o *Object) Not() Value {
	return getBool(len(o.Fields) == 0)
}

func (o *Object) At(ix Value) (Value, error) {
	v, ok := o.Fields[ix]
	if !ok {
		return Void{}, nil
	}
	if f, ok := v.(Field); ok {
		return f.Value, nil
	}
	return v, nil
}

func (o *Object) Call(ident string, args []Value) (Value, error) {
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

func (o *Object) SetAt(prop, value Value) error {
	return o.Set(prop, value)
}

func (o *Object) Set(prop, value Value) error {
	v, ok := o.Fields[prop]
	if !ok {
		if !o.canBeExtended() {
			return fmt.Errorf("object can not be extended")
		}
		o.Fields[prop] = fieldByAssignment(value)
		return nil
	}
	f, ok := v.(Field)
	if !ok {
		return ErrEval
	}
	if !f.writable {
		return fmt.Errorf("%s: property is not writable", prop)
	}
	f.Value = value
	o.Fields[prop] = f
	return nil
}

func (o *Object) Del(prop Value) error {
	if o.isFrozen() || o.isSealed() {
		return fmt.Errorf("property can not be delete from sealed/frozen object")
	}
	v, ok := o.Fields[prop]
	if !ok {
		return nil
	}
	f, ok := v.(Field)
	if !ok {
		delete(o.Fields, prop)
	}
	if !f.configurable {
		return fmt.Errorf("property can not be deleted")
	}
	delete(o.Fields, prop)
	return nil
}

func (o *Object) DelAt(prop Value) error {
	return o.Del(prop)
}

func (o *Object) Get(prop Value) (Value, error) {
	v, ok := o.Fields[prop]
	if !ok {
		var x Void
		return x, nil
	}
	f, ok := v.(Field)
	if ok {
		return f.Value, nil
	}
	return v, nil
}

func (o *Object) Entries() Value {
	arr := createArray()
	for k, v := range o.Fields {
		a := createArray()
		a.Values = append(a.Values, k)
		a.Values = append(a.Values, v)

		arr.Values = append(arr.Values, a)
	}
	return arr
}

func (o *Object) Values() []Value {
	var vs []Value
	for k := range o.Fields {
		v := o.Fields[k]
		if f, ok := v.(Field); ok {
			v = f.Value
		}
		vs = append(vs, v)
	}
	return vs
}

func (o *Object) canBeExtended() bool {
	return o.locked == 0
}

func (o *Object) isFrozen() bool {
	return (o.locked & frozenObject) == frozenObject
}

func (o *Object) isSealed() bool {
	return (o.locked & sealedObject) == sealedObject
}

type Array struct {
	*Object
	Values []Value
}

func NewArray() *Array {
	return createArray()
}

func (a *Array) Append(val Value) {
	a.Values = append(a.Values, val)
}

func createArray() *Array {
	return &Array{
		Object: createObject(),
	}
}

func (a *Array) Type() string {
	return "array"
}

func (a *Array) String() string {
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

func (a *Array) True() Value {
	return getBool(len(a.Values) != 0)
}

func (a *Array) Not() Value {
	return getBool(len(a.Values) == 0)
}

func (a *Array) At(ix Value) (Value, error) {
	x, ok := ix.(Float)
	if !ok {
		return nil, ErrEval
	}
	if x.value >= 0 && int(x.value) < len(a.Values) {
		return a.Values[int(x.value)], nil
	}
	return nil, ErrIndex
}

func (a *Array) SetAt(ix Value, value Value) error {
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

func (a *Array) DelAt(prop Value) error {
	return nil
}

func (a *Array) Get(ident Value) (Value, error) {
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

func (a *Array) Call(ident string, args []Value) (Value, error) {
	var fn func([]Value) (Value, error)
	switch ident {
	case "at":
		fn = a.at
	case "concat":
		fn = a.concat
	case "entries":
		fn = a.entries
	case "every":
		fn = a.every
	case "fill":
		fn = a.fill
	case "filter":
		fn = a.filter
	case "find":
		fn = a.find
	case "findIndex":
		fn = a.findIndex
	case "findLast":
		fn = a.findLast
	case "findLastIndex":
		fn = a.findLastIndex
	case "flat":
		fn = a.flat
	case "flatMap":
		fn = a.flatMap
	case "forEach":
		fn = a.forEach
	case "includes":
		fn = a.includes
	case "indexOf":
		fn = a.indexOf
	case "lastIndexOf":
		fn = a.lastIndexOf
	case "join":
		fn = a.join
	case "map":
		fn = a.mapArray
	case "pop":
		fn = a.pop
	case "push":
		fn = a.push
	case "reduce":
		fn = a.reduce
	case "reduceRight":
		fn = a.reduceRight
	case "reverse":
		fn = a.reverse
	case "shift":
		fn = a.shift
	case "slice":
		fn = a.slice
	case "splice":
		fn = a.slice
	case "some":
		fn = a.some
	case "unshift":
		fn = a.unshift
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	if fn == nil {
		return nil, ErrImpl
	}
	return fn(args)
}

func (a *Array) Entries() Value {
	arr := createArray()
	for i := range a.Values {
		x := createArray()
		x.Values = append(x.Values, getFloat(float64(i)))
		x.Values = append(x.Values, a.Values[i])
		arr.Values = append(arr.Values, x)
	}
	return arr
}

func (a *Array) List() []Value {
	return a.Values
}

func (a *Array) Return() {
	return
}

func (a *Array) at(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
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

func (a *Array) concat(args []Value) (Value, error) {
	if len(args) == 0 {
		return a, nil
	}
	for i := range args {
		if other, ok := args[i].(*Array); ok {
			a.Values = slices.Concat(a.Values, other.Values)
		} else {
			a.Values = append(a.Values, args[i])
		}
	}
	return a, nil
}

func (a *Array) entries(args []Value) (Value, error) {
	return nil, nil
}

func (a *Array) every(args []Value) (Value, error) {
	if len(args) == 0 {
		return getBool(false), nil
	}
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

func (a *Array) fill(args []Value) (Value, error) {
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

func (a *Array) filter(args []Value) (Value, error) {
	if len(args) == 0 {
		return a, nil
	}
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

func (a *Array) find(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
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

func (a *Array) findIndex(args []Value) (Value, error) {
	if len(args) == 0 {
		return getFloat(-1), nil
	}
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

func (a *Array) findLast(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
	find, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	values := slices.Clone(a.Values)
	slices.Reverse(values)
	for i := range values {
		args := []Value{
			values[i],
			NewFloat(float64(len(a.Values) - (i + 1))),
			a,
		}
		ok, err := find.Call(args)
		if err != nil {
			return nil, err
		}
		if isTrue(ok) {
			return values[i], nil
		}
	}
	return Void{}, nil
}

func (a *Array) findLastIndex(args []Value) (Value, error) {
	if len(args) == 0 {
		return getFloat(-1), nil
	}
	find, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	values := slices.Clone(a.Values)
	slices.Reverse(values)
	for i := range values {
		args := []Value{
			values[i],
			NewFloat(float64(len(a.Values) - (i + 1))),
			a,
		}
		ok, err := find.Call(args)
		if err != nil {
			return nil, err
		}
		if isTrue(ok) {
			return NewFloat(float64(len(a.Values) - (i + 1))), nil
		}
	}
	return Void{}, nil
}

func (a *Array) flat(args []Value) (Value, error) {
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
		arr, ok := v.(*Array)
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

func (a *Array) flatMap(args []Value) (Value, error) {
	if len(args) == 0 {
		return a, nil
	}
	transform, ok := args[0].(Callable)
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
		ret, err := transform.Call(args)
		if err != nil {
			return nil, err
		}
		if x, ok := ret.(*Array); ok {
			arr.Values = append(arr.Values, x.Values...)
		} else {
			arr.Values = append(arr.Values, ret)
		}
	}
	return arr, nil
}

func (a *Array) forEach(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
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

func (a *Array) includes(args []Value) (Value, error) {
	if len(args) == 0 {
		return getBool(false), nil
	}
	var beg int
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		beg = int(x.value)
		if beg < 0 {
			beg = len(a.Values) + beg
		}
		if beg >= len(a.Values) || beg < 0 {
			return getBool(false), nil
		}
	}
	for i := range a.Values[beg:] {
		if isEqual(args[0], a.Values[beg+i]) {
			return getBool(true), nil
		}
	}
	return getBool(false), nil
}

func (a *Array) indexOf(args []Value) (Value, error) {
	if len(args) == 0 {
		return getFloat(-1), nil
	}
	var beg int
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		beg = int(x.value)
		if beg < 0 {
			beg = len(a.Values) + beg
		}
		if beg >= len(a.Values) || beg < 0 {
			return getBool(false), nil
		}
	}
	for i := range a.Values[beg:] {
		if isEqual(a.Values[beg+i], args[0]) {
			return getFloat(float64(beg + i)), nil
		}
	}
	return getFloat(-1), nil
}

func (a *Array) lastIndexOf(args []Value) (Value, error) {
	if len(args) == 0 {
		return getFloat(-1), nil
	}
	var beg int
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		beg = int(x.value)
		if beg < 0 {
			beg = len(a.Values) + beg
		}
		if beg >= len(a.Values) || beg < 0 {
			return getBool(false), nil
		}
	}
	values := slices.Clone(a.Values[beg:])
	slices.Reverse(values)
	for i := range values {
		if isEqual(values[i], args[0]) {
			ix := len(a.Values) - (i + 1)
			return getFloat(float64(ix)), nil
		}
	}
	return getFloat(-1), nil
}

func (a *Array) join(args []Value) (Value, error) {
	var (
		list []string
		sep  = " "
	)
	if len(args) >= 1 {
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return nil, ErrType
		}
		sep = str.String()
	}
	for i := range a.Values {
		str, ok := a.Values[i].(fmt.Stringer)
		if !ok {
			return nil, ErrType
		}
		list = append(list, str.String())
	}
	return getString(strings.Join(list, sep)), nil
}

func (a *Array) mapArray(args []Value) (Value, error) {
	if len(args) == 0 {
		return a, nil
	}
	transform, ok := args[0].(Callable)
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
		ret, err := transform.Call(args)
		if err != nil {
			return nil, err
		}
		arr.Values = append(arr.Values, ret)
	}
	return arr, nil
}

func (a *Array) pop(args []Value) (Value, error) {
	if len(a.Values) == 0 {
		return Void{}, nil
	}
	ret := a.Values[len(a.Values)-1]
	a.Values = a.Values[:len(a.Values)-1]
	return ret, nil
}

func (a *Array) push(args []Value) (Value, error) {
	a.Values = slices.Concat(a.Values, args)
	n := len(a.Values)
	return getFloat(float64(n)), nil
}

func (a *Array) reduce(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
	apply, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	var (
		ret = args[1]
		err error
	)
	for i := range a.Values {
		args := []Value{
			ret,
			a.Values[i],
			getFloat(float64(i)),
			a,
		}
		ret, err = apply.Call(args)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (a *Array) reduceRight(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
	apply, ok := args[0].(Callable)
	if !ok {
		return nil, ErrType
	}
	var (
		err    error
		ret    = args[1]
		values = slices.Clone(a.Values)
	)
	slices.Reverse(values)
	for i := range values {
		args := []Value{
			ret,
			a.Values[i],
			getFloat(float64(i)),
			a,
		}
		ret, err = apply.Call(args)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (a *Array) reverse(args []Value) (Value, error) {
	slices.Reverse(a.Values)
	return a, nil
}

func (a *Array) shift(args []Value) (Value, error) {
	if len(a.Values) == 0 {
		return Void{}, nil
	}
	ret := a.Values[0]
	a.Values = a.Values[1:]
	return ret, nil
}

func (a *Array) slice(args []Value) (Value, error) {
	var (
		beg int
		end = len(a.Values)
	)
	if len(args) >= 1 {
		x, ok := args[0].(Float)
		if !ok {
			return nil, ErrType
		}
		beg = int(x.value)
		if beg < 0 {
			beg = len(a.Values) + beg
		}
	}
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		end = int(x.value)
		if end < 0 {
			end = len(a.Values) + end
		}
	}
	arr := createArray()
	arr.Values = slices.Clone(a.Values[beg:end])
	return arr, nil
}

func (a *Array) splice(args []Value) (Value, error) {
	var (
		start int
		size  int
		list  []Value
	)
	x, ok := args[0].(Float)
	if !ok {
		return nil, ErrType
	}
	if start = int(x.value); start < 0 {
		start = len(a.Values) + start
	}
	if len(args) >= 2 {
		x, ok := args[0].(Float)
		if !ok {
			return nil, ErrType
		}
		if size = int(x.value); size < 0 {
			return nil, fmt.Errorf("negative count")
		}
		if start+size >= len(a.Values) {
			a.Values, list = a.Values[:start], a.Values[start:]
			size = 0
		} else {
			list = a.Values[start : start+size]
			a.Values = append(a.Values[:start], a.Values[start+size:]...)
		}
	} else {
		a.Values = a.Values[:start]

		arr := createArray()
		arr.Values = a.Values[start:]
		return arr, nil
	}
	if len(args) >= 3 {
		rest := slices.Clone(args[2:])
		a.Values = append(a.Values[:start], append(rest, a.Values[start+size:]...)...)
	}

	arr := createArray()
	arr.Values = list
	return arr, nil
}

func (a *Array) some(args []Value) (Value, error) {
	if len(args) == 0 {
		return getBool(false), nil
	}
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
		if isTrue(ok) {
			return getBool(true), nil
		}
	}
	return getBool(false), nil
}

func (a *Array) unshift(args []Value) (Value, error) {
	a.Values = slices.Concat(args, a.Values)
	n := len(a.Values)
	return getFloat(float64(n)), nil
}

type callableFunc func([]Value) (Value, error)

func (fn callableFunc) Call(args []Value) (Value, error) {
	return fn(args)
}

func asCallable(fn func([]Value) (Value, error)) Callable {
	return callableFunc(fn)
}

type global struct {
	name  string
	fnset map[string]Callable
}

func (g global) Type() string {
	return "object"
}

func (g global) True() Value {
	return getBool(true)
}

func (g global) String() string {
	return g.name
}

func (g global) Call(ident string, args []Value) (Value, error) {
	call, ok := g.fnset[ident]
	if !ok {
		return nil, fmt.Errorf("%s.%s: undefined function", g.name, ident)
	}
	return call.Call(args)
}

func makeObject() Value {
	g := global{
		name:  "Object",
		fnset: make(map[string]Callable),
	}
	g.fnset["seal"] = asCallable(objectSeal)
	g.fnset["freeze"] = asCallable(objectFreeze)
	g.fnset["isSealed"] = asCallable(objectIsSealed)
	g.fnset["isFrozen"] = asCallable(objectIsFrozen)
	g.fnset["create"] = nil
	g.fnset["assign"] = nil
	g.fnset["entries"] = nil
	g.fnset["keys"] = asCallable(objectKeys)
	g.fnset["values"] = asCallable(objectValues)
	g.fnset["is"] = asCallable(objectIs)
	g.fnset["groupBy"] = nil
	g.fnset["preventExtensions"] = asCallable(objectPreventExtensions)
	g.fnset["isExtensible"] = asCallable(objectIsExtensible)
	g.fnset["propertyIsEnumerable"] = nil
	return g
}

func objectIsExtensible(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	return getBool(obj.canBeExtended()), nil
}

func objectPreventExtensions(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	if obj.isFrozen() || obj.isSealed() {
		return obj, nil
	}
	obj.locked |= lockedObject
	return obj, nil
}

func objectSeal(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	if obj.isFrozen() {
		return obj, nil
	}
	obj.locked |= sealedObject
	for k, d := range obj.Fields {
		f, ok := d.(Field)
		if !ok {
			continue
		}

		f.writable = true
		f.configurable = false
		f.enumerable = false

		obj.Fields[k] = f
	}
	return obj, nil
}

func objectFreeze(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	if obj.isFrozen() {
		return obj, nil
	}
	obj.locked |= lockedObject
	for k, d := range obj.Fields {
		f, ok := d.(Field)
		if !ok {
			continue
		}

		f.writable = false
		f.configurable = false
		f.enumerable = false

		obj.Fields[k] = f
	}
	return obj, nil
}

func objectIsSealed(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	if obj.canBeExtended() {
		return getBool(false), nil
	}
	for k := range obj.Fields {
		f, ok := obj.Fields[k].(Field)
		if !ok || f.configurable {
			return getBool(false), nil
		}
	}
	return getBool(true), nil
}

func objectIsFrozen(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	if obj.canBeExtended() {
		return getBool(false), nil
	}
	for k := range obj.Fields {
		f, ok := obj.Fields[k].(Field)
		if !ok || f.configurable || f.writable {
			return getBool(false), nil
		}
	}
	return getBool(true), nil
}

func objectKeys(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	arr := createArray()
	for k := range obj.Fields {
		arr.Values = append(arr.Values, k)
	}
	return arr, nil
}

func objectValues(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	obj, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	arr := createArray()
	arr.Values = obj.Values()
	return arr, nil
}

func objectIs(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, ErrArgument
	}
	obj1, ok := args[0].(*Object)
	if !ok {
		return nil, ErrType
	}
	obj2, ok := args[1].(*Object)
	if !ok {
		return nil, ErrType
	}
	return getBool(obj1 == obj2), nil
}

func makeArray() Value {
	g := global{
		name:  "Array",
		fnset: make(map[string]Callable),
	}
	g.fnset["isArray"] = asCallable(arrayIsArray)
	g.fnset["from"] = nil
	g.fnset["of"] = nil
	return g
}

func arrayIsArray(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	_, ok := args[0].(*Array)
	return getBool(ok), nil
}

func makeJson() Value {
	g := global{
		name:  "JSON",
		fnset: make(map[string]Callable),
	}

	g.fnset["parse"] = asCallable(jsonParse)
	g.fnset["stringify"] = asCallable(jsonString)

	return g
}

func jsonParse(args []Value) (Value, error) {
	if len(args) != 1 {
		return Void{}, ErrArgument
	}
	str, ok := args[0].(String)
	if !ok {
		return args[0], nil
	}
	var (
		obj interface{}
		buf = strings.NewReader(str.value)
	)
	if err := json.NewDecoder(buf).Decode(&obj); err != nil {
		return Void{}, err
	}
	return NativeToValues(obj)
}

func jsonString(args []Value) (Value, error) {
	if len(args) != 1 {
		return Void{}, ErrArgument
	}
	v, err := ValuesToNative(args[0])
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return getString(buf.String()), nil
}

func ValuesToNative(arg Value) (interface{}, error) {
	switch a := arg.(type) {
	case String:
		return a.value, nil
	case Float:
		return a.value, nil
	case Bool:
		return a.value, nil
	case *Array:
		var arr []interface{}
		for i := range a.Values {
			v, err := ValuesToNative(a.Values[i])
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case *Object:
		arr := make(map[string]interface{})
		for k, v := range a.Fields {
			vv, err := ValuesToNative(v)
			if err != nil {
				return nil, err
			}
			arr[fmt.Sprintf("%s", k)] = vv
		}
		return arr, nil
	default:
		return nil, fmt.Errorf("type can not be converted to json")
	}
}

func NativeToValues(obj interface{}) (Value, error) {
	switch v := obj.(type) {
	case string:
		return getString(v), nil
	case float64:
		return getFloat(v), nil
	case bool:
		return getBool(v), nil
	case []interface{}:
		arr := createArray()
		for i := range v {
			a, err := NativeToValues(v[i])
			if err != nil {
				return nil, err
			}
			arr.Values = append(arr.Values, a)
		}
		return arr, nil
	case map[string]interface{}:
		obj := createObject()
		for kv, vv := range v {
			a, err := NativeToValues(vv)
			if err != nil {
				return nil, err
			}
			obj.Fields[getString(kv)] = a
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("%v: unsupported JSON type", obj)
	}
}

var jwtConfig = &jwt.Config{
	Secret: "supersecretapikey11!",
	Alg:    jwt.HS256,
	Ttl:    time.Hour * 24,
}

func makeJWT() Value {
	g := global{
		name:  "JWT",
		fnset: make(map[string]Callable),
	}
	g.fnset["decode"] = asCallable(jwtDecode)
	g.fnset["encode"] = asCallable(jwtEncode)

	return g
}

func jwtDecode(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, ErrArgument
	}
	str, ok := args[0].(String)
	if !ok {
		return Void{}, ErrEval
	}
	return Void{}, jwt.Decode(str.String(), jwtConfig)
}

func jwtEncode(args []Value) (Value, error) {
	if len(args) != 1 {
		return Void{}, ErrArgument
	}
	str, err := jwt.Encode(args[0], jwtConfig)
	return getString(str), err
}

func makeMath() Value {
	g := global{
		name:  "Math",
		fnset: make(map[string]Callable),
	}

	g.fnset["abs"] = nil
	g.fnset["ceil"] = nil
	g.fnset["cos"] = nil
	g.fnset["exp"] = nil
	g.fnset["floor"] = nil
	g.fnset["log"] = nil
	g.fnset["round"] = nil
	g.fnset["max"] = nil
	g.fnset["min"] = nil
	g.fnset["pow"] = nil
	g.fnset["random"] = nil
	g.fnset["sin"] = nil
	g.fnset["tan"] = nil
	g.fnset["trunc"] = nil

	return g
}

func makeConsole() Value {
	g := global{
		name:  "Array",
		fnset: make(map[string]Callable),
	}
	g.fnset["log"] = asCallable(consoleLog)
	g.fnset["error"] = asCallable(consoleError)
	g.fnset["warning"] = nil
	return g
}

func consoleLog(args []Value) (Value, error) {
	return writeConsole(os.Stdout, args)
}

func consoleError(args []Value) (Value, error) {
	return writeConsole(os.Stderr, args)
}

func writeConsole(w io.Writer, args []Value) (Value, error) {
	for i := range args {
		var (
			val = args[i]
			str string
		)
		if call, ok := val.(interface {
			Call(string, []Value) (Value, error)
		}); ok {
			v, err := call.Call("toString", []Value{})
			if err == nil || errors.Is(err, ErrReturn) {
				val = v
			}
		}
		if s, ok := val.(fmt.Stringer); ok {
			str = s.String()
		} else {
			str = fmt.Sprint(val)
		}
		fmt.Fprint(w, str)
		fmt.Fprint(w, " ")
	}
	fmt.Fprintln(w)
	return Void{}, nil
}
