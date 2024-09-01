package play

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/midbel/mule/environ"
)

var (
	ErrBreak    = errors.New("break")
	ErrContinue = errors.New("continue")
	ErrReturn   = errors.New("return")
	ErrThrow    = errors.New("throw")
	ErrEval     = errors.New("node can not be evalualed in current context")
	ErrOp       = errors.New("unsupported operation")
	ErrZero     = errors.New("division by zero")
	ErrConst    = errors.New("constant variable can not be reassigned")
	ErrIndex    = errors.New("index out of bound")
	ErrArgument = errors.New("invalid number of arguments")
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

func isTrue(val Value) bool {
	b, ok := val.True().(Bool)
	if !ok {
		return false
	}
	return b.value
}

type envValue struct {
	Const bool
	Value
}

func constValue(val Value) Value {
	return createValueForEnv(val, true)
}

func letValue(val Value) Value {
	return createValueForEnv(val, false)
}

func createValueForEnv(val Value, ro bool) Value {
	if _, ok := val.(envValue); ok {
		return val
	}
	return envValue{
		Value: val,
		Const: ro,
	}
}

type Env struct {
	parent environ.Environment[Value]
	values map[string]Value
}

func Empty() environ.Environment[Value] {
	return Enclosed(nil)
}

func Enclosed(parent environ.Environment[Value]) environ.Environment[Value] {
	return &Env{
		parent: parent,
		values: make(map[string]Value),
	}
}

func (e *Env) Resolve(ident string) (Value, error) {
	v, ok := e.values[ident]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(ident)
	}
	return nil, fmt.Errorf("%s: %w", ident, environ.ErrDefined)
}

func (e *Env) Define(ident string, value Value) error {
	v, err := e.Resolve(ident)
	if err == nil {
		x, ok := v.(envValue)
		if ok && x.Const {
			return fmt.Errorf("%s: %w", ident, ErrConst)
		}
	}
	e.values[ident] = value
	return nil
}

type BuiltinFunc struct {
	Ident string
	Func  func([]Value) (Value, error)
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

type Void struct{}

func isUndefined(v Value) bool {
	_, ok := v.(Void)
	return ok
}

func (_ Void) String() string {
	return "undefined"
}

func (_ Void) True() Value {
	return getBool(false)
}

func (_ Void) Rev() Value {
	return nan()
}

func (_ Void) Add(_ Value) (Value, error) {
	return nan(), nil
}

func (_ Void) Sub(_ Value) (Value, error) {
	return nan(), nil
}

func (_ Void) Mul(_ Value) (Value, error) {
	return nan(), nil
}

func (_ Void) Div(_ Value) (Value, error) {
	return nan(), nil
}

func (_ Void) Mod(_ Value) (Value, error) {
	return nan(), nil
}

func (_ Void) Pow(_ Value) (Value, error) {
	return nan(), nil
}

func (_ Void) Equal(other Value) (Value, error) {
	if !isUndefined(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

func (_ Void) StrictEqual(other Value) (Value, error) {
	if !isUndefined(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

func (_ Void) NotEqual(other Value) (Value, error) {
	if isUndefined(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

func (_ Void) StrictNotEqual(other Value) (Value, error) {
	if isUndefined(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

type Nil struct{}

func isNull(v Value) bool {
	_, ok := v.(Nil)
	return ok
}

func (_ Nil) String() string {
	return "null"
}

func (_ Nil) True() Value {
	return getBool(false)
}

func (_ Nil) Rev() Value {
	return getFloat(0)
}

func (_ Nil) Add(_ Value) (Value, error) {
	return getFloat(0), nil
}

func (_ Nil) Sub(_ Value) (Value, error) {
	return getFloat(0), nil
}

func (_ Nil) Mul(_ Value) (Value, error) {
	return getFloat(0), nil
}

func (_ Nil) Div(_ Value) (Value, error) {
	return getFloat(0), nil
}

func (_ Nil) Mod(_ Value) (Value, error) {
	return getFloat(0), nil
}

func (_ Nil) Pow(_ Value) (Value, error) {
	return getFloat(0), nil
}

func (_ Nil) Equal(other Value) (Value, error) {
	if !isNull(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

func (_ Nil) StrictEqual(other Value) (Value, error) {
	if !isNull(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

func (_ Nil) NotEqual(other Value) (Value, error) {
	if isNull(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

func (_ Nil) StrictNotEqual(other Value) (Value, error) {
	if isNull(other) {
		return getBool(false), nil
	}
	return getBool(true), nil
}

type Float struct {
	value float64
}

func getFloat(val float64) Float {
	return Float{
		value: val,
	}
}

func nan() Float {
	return getFloat(math.NaN())
}

func (f Float) String() string {
	return strconv.FormatFloat(f.value, 'f', -1, 64)
}

func (f Float) True() Value {
	return getBool(f.value != 0)
}

func (f Float) Not() Value {
	return getBool(f.value == 0)
}

func (f Float) Rev() Value {
	return getFloat(-f.value)
}

func (f Float) Float() Value {
	return f
}

func (f Float) Incr() (Value, error) {
	return f.Add(getFloat(1))
}

func (f Float) Decr() (Value, error) {
	return f.Add(getFloat(-1))
}

func (f Float) Add(other Value) (Value, error) {
	if isUndefined(other) {
		return nan(), nil
	}
	if isNull(other) {
		return getFloat(0), nil
	}
	switch other := other.(type) {
	case Float:
		x := f.value + other.value
		return getFloat(x), nil
	case String:
		str := fmt.Sprintf("%f%s", f.value, other.value)
		return getString(str), nil
	default:
		return nil, ErrOp
	}
}

func (f Float) Sub(other Value) (Value, error) {
	if isUndefined(other) {
		return nan(), nil
	}
	if isNull(other) {
		return getFloat(0), nil
	}
	right, ok := other.(Float)
	if !ok {
		return nil, ErrOp
	}
	x := f.value - right.value
	return getFloat(x), nil
}

func (f Float) Mul(other Value) (Value, error) {
	if isUndefined(other) {
		return nan(), nil
	}
	if isNull(other) {
		return getFloat(0), nil
	}
	right, ok := other.(Float)
	if !ok {
		return nil, ErrOp
	}
	x := f.value * right.value
	return getFloat(x), nil
}

func (f Float) Div(other Value) (Value, error) {
	if isUndefined(other) {
		return nan(), nil
	}
	if isNull(other) {
		return getFloat(0), nil
	}
	right, ok := other.(Float)
	if !ok {
		return nil, ErrOp
	}
	if right.value == 0 {
		return nil, ErrZero
	}
	x := f.value / right.value
	return getFloat(x), nil
}

func (f Float) Mod(other Value) (Value, error) {
	if isUndefined(other) {
		return nan(), nil
	}
	if isNull(other) {
		return getFloat(0), nil
	}
	right, ok := other.(Float)
	if !ok {
		return nil, ErrOp
	}
	if right.value == 0 {
		return nil, ErrZero
	}
	x := math.Mod(f.value, right.value)
	return getFloat(x), nil
}

func (f Float) Pow(other Value) (Value, error) {
	if isUndefined(other) {
		return nan(), nil
	}
	if isNull(other) {
		return getFloat(0), nil
	}
	right, ok := other.(Float)
	if !ok {
		return nil, ErrOp
	}
	x := math.Pow(f.value, right.value)
	return getFloat(x), nil
}

func (f Float) Equal(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case Float:
		return getBool(f.value == other.value), nil
	case String:
		x, err := strconv.ParseFloat(other.value, 64)
		if err != nil {
			return getBool(false), nil
		}
		return getBool(f.value == x), nil
	case Bool:
		var x float64
		if other.value {
			x = 1
		}
		return getBool(f.value == x), nil
	default:
		return nil, ErrOp
	}
}

func (f Float) StrictEqual(other Value) (Value, error) {
	x, ok := other.(Float)
	if !ok {
		return getBool(ok), nil
	}
	return getBool(f.value == x.value), nil
}

func (f Float) NotEqual(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case Float:
		return getBool(f.value != other.value), nil
	case String:
		x, err := strconv.ParseFloat(other.value, 64)
		if err != nil {
			return getBool(true), nil
		}
		return getBool(f.value != x), nil
	case Bool:
		var x float64
		if other.value {
			x = 1
		}
		return getBool(f.value != x), nil
	default:
		return nil, ErrOp
	}
}

func (f Float) StrictNotEqual(other Value) (Value, error) {
	x, ok := other.(Float)
	if !ok {
		return getBool(ok), nil
	}
	return getBool(f.value != x.value), nil
}

func (f Float) LesserThan(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case Float:
		return getBool(f.value < other.value), nil
	case String:
		x, err := strconv.ParseFloat(other.value, 64)
		if err != nil {
			return getBool(false), nil
		}
		return getBool(f.value < x), nil
	case Bool:
		var x float64
		if other.value {
			x = 1
		}
		return getBool(f.value < x), nil
	default:
		return nil, ErrOp
	}
}

func (f Float) LesserEqual(other Value) (Value, error) {
	less, err := f.LesserThan(other)
	if err != nil {
		return nil, err
	}
	if isTrue(less) {
		return less, nil
	}
	return f.Equal(other)
}

func (f Float) GreaterThan(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case Float:
		return getBool(f.value > other.value), nil
	case String:
		x, err := strconv.ParseFloat(other.value, 64)
		if err != nil {
			return getBool(false), nil
		}
		return getBool(f.value > x), nil
	case Bool:
		var x float64
		if other.value {
			x = 1
		}
		return getBool(f.value > x), nil
	default:
		return nil, ErrOp
	}
}

func (f Float) GreaterEqual(other Value) (Value, error) {
	great, err := f.GreaterThan(other)
	if err != nil {
		return nil, err
	}
	if isTrue(great) {
		return great, nil
	}
	return f.Equal(other)
}

type Bool struct {
	value bool
}

func getBool(val bool) Value {
	return Bool{
		value: val,
	}
}

func (b Bool) String() string {
	return strconv.FormatBool(b.value)
}

func (b Bool) True() Value {
	return getBool(b.value)
}

func (b Bool) Not() Value {
	return getBool(!b.value)
}

func (b Bool) Float() Value {
	if b.value {
		return getFloat(1)
	}
	return getFloat(0)
}

func (b Bool) Equal(other Value) (Value, error) {
	return getBool(b.value == isTrue(other)), nil
}

func (b Bool) StrictEqual(other Value) (Value, error) {
	x, ok := other.(Bool)
	if !ok {
		return getBool(ok), nil
	}
	return getBool(b.value == x.value), nil
}

func (b Bool) NotEqual(other Value) (Value, error) {
	return getBool(b.value != isTrue(other)), nil
}

func (b Bool) StrictNotEqual(other Value) (Value, error) {
	x, ok := other.(Bool)
	if !ok {
		return getBool(ok), nil
	}
	return getBool(b.value != x.value), nil
}

type String struct {
	value string
}

func getString(val string) Value {
	return String{
		value: val,
	}
}

func (s String) String() string {
	return s.value
}

func (s String) True() Value {
	return getBool(s.value != "")
}

func (s String) Not() Value {
	return getBool(s.value == "")
}

func (s String) Float() Value {
	v, err := strconv.ParseFloat(s.value, 64)
	if err != nil {
		return getFloat(math.NaN())
	}
	return getFloat(v)
}

func (s String) Add(other Value) (Value, error) {
	switch other := other.(type) {
	case String:
		str := s.value + other.value
		return getString(str), nil
	case Float:
		x := fmt.Sprintf("%s%f", s.value, other.value)
		return getString(x), nil
	default:
		return nil, ErrOp
	}
}

func (s String) Equal(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case String:
		return getBool(s.value == other.value), nil
	case Float:
		str := strconv.FormatFloat(other.value, 'f', -1, 64)
		return getBool(s.value == str), nil
	case Bool:
		str := strconv.FormatBool(other.value)
		return getBool(s.value == str), nil
	default:
		return nil, ErrOp
	}
}

func (s String) StrictEqual(other Value) (Value, error) {
	x, ok := other.(String)
	if !ok {
		return getBool(ok), nil
	}
	return getBool(s.value == x.value), nil
}

func (s String) NotEqual(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case String:
		return getBool(s.value != other.value), nil
	case Float:
		str := strconv.FormatFloat(other.value, 'f', -1, 64)
		return getBool(s.value != str), nil
	case Bool:
		str := strconv.FormatBool(other.value)
		return getBool(s.value != str), nil
	default:
		return nil, ErrOp
	}
}

func (s String) StrictNotEqual(other Value) (Value, error) {
	x, ok := other.(String)
	if !ok {
		return getBool(ok), nil
	}
	return getBool(s.value != x.value), nil
}

func (s String) LesserThan(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case String:
		return getBool(s.value < other.value), nil
	case Float:
		str := strconv.FormatFloat(other.value, 'f', -1, 64)
		return getBool(s.value < str), nil
	case Bool:
		str := strconv.FormatBool(other.value)
		return getBool(s.value < str), nil
	default:
		return nil, ErrOp
	}
}

func (s String) LesserEqual(other Value) (Value, error) {
	less, err := s.LesserThan(other)
	if err != nil {
		return nil, err
	}
	if isTrue(less) {
		return less, nil
	}
	return s.Equal(other)
}

func (s String) GreaterThan(other Value) (Value, error) {
	if isNull(other) || isUndefined(other) {
		return getBool(false), nil
	}
	switch other := other.(type) {
	case String:
		return getBool(s.value > other.value), nil
	case Float:
		str := strconv.FormatFloat(other.value, 'f', -1, 64)
		return getBool(s.value > str), nil
	case Bool:
		str := strconv.FormatBool(other.value)
		return getBool(s.value > str), nil
	default:
		return nil, ErrOp
	}
}

func (s String) GreaterEqual(other Value) (Value, error) {
	great, err := s.GreaterThan(other)
	if err != nil {
		return nil, err
	}
	if isTrue(great) {
		return great, nil
	}
	return s.Equal(other)
}

type Object struct {
	Fields map[Value]Value
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
		return nil, ErrOp
	}
	if x.value >= 0 && int(x.value) < len(a.Values) {
		return a.Values[int(x.value)], nil
	}
	return nil, ErrIndex
}

func (a Array) List() []Value {
	return a.Values
}

func (a Array) Return() {
	return
}

type Math struct{}

func (m Math) String() string {
	return "Math"
}

func (m Math) True() Value {
	return getBool(true)
}

func (m Math) Call(ident string, args []Value) (Value, error) {
	return Void{}, nil
}

type Json struct{}

func (j Json) String() string {
	return "JSON"
}

func (j Json) True() Value {
	return getBool(true)
}

func (j Json) Call(ident string, args []Value) (Value, error) {
	switch ident {
	case "parse":
		return j.parse(args)
	case "stringify":
		return j.stringify(args)
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
}

func (j Json) parse(args []Value) (Value, error) {
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
	return nativeToValues(obj)
}

func (j Json) stringify(args []Value) (Value, error) {
	if len(args) != 1 {
		return Void{}, ErrArgument
	}
	v, err := valuesToNative(args[0])
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return getString(buf.String()), nil
}

func valuesToNative(arg Value) (interface{}, error) {
	switch a := arg.(type) {
	case String:
		return a.value, nil
	case Float:
		return a.value, nil
	case Bool:
		return a.value, nil
	case Array:
		var arr []interface{}
		for i := range a.Values {
			v, err := valuesToNative(a.Values[i])
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case Object:
		arr := make(map[string]interface{})
		for k, v := range a.Fields {
			vv, err := valuesToNative(v)
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

func nativeToValues(obj interface{}) (Value, error) {
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
			a, err := nativeToValues(v[i])
			if err != nil {
				return nil, err
			}
			arr.Values = append(arr.Values, a)
		}
		return arr, nil
	case map[string]interface{}:
		obj := createObject()
		for kv, vv := range v {
			a, err := nativeToValues(vv)
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

type Console struct{}

func (c Console) String() string {
	return "console"
}

func (c Console) True() Value {
	return getBool(true)
}

func (c Console) Call(ident string, args []Value) (Value, error) {
	var w io.Writer
	switch ident {
	case "log":
		w = os.Stdout
	case "error":
		w = os.Stderr
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	for i := range args {
		fmt.Fprint(w, args[i])
		fmt.Fprint(w, " ")
	}
	fmt.Fprintln(w)
	return Void{}, nil
}

func Default() environ.Environment[Value] {
	top := Empty()
	top.Define("console", Console{})
	top.Define("JSON", Json{})
	top.Define("Math", Math{})
	top.Define("parseInt", createBuiltinFunc("parseInt", execParseInt))
	top.Define("parseFloat", createBuiltinFunc("parseFloat", execParseFloat))
	top.Define("isNaN", createBuiltinFunc("isNaN", execIsNaN))

	return top
}

func Eval(r io.Reader) (Value, error) {
	return EvalWithEnv(r, Enclosed(Default()))
}

func EvalWithEnv(r io.Reader, env environ.Environment[Value]) (Value, error) {
	n, err := ParseReader(r)
	if err != nil {
		return nil, err
	}
	return eval(n, env)
}

func eval(n Node, env environ.Environment[Value]) (Value, error) {
	switch n := n.(type) {
	case Body:
		return evalBody(n, env)
	case Null:
		var i Nil
		return i, nil
	case Undefined:
		var v Void
		return v, nil
	case List:
		return evalList(n, env)
	case Map:
		return evalMap(n, env)
	case Group:
		return evalGroup(n, env)
	case Literal[string]:
		return getString(n.Value), nil
	case Literal[float64]:
		return getFloat(n.Value), nil
	case Literal[bool]:
		return getBool(n.Value), nil
	case Identifier:
		return evalIdent(n, env)
	case Index:
		return evalIndex(n, env)
	case Access:
		return evalAccess(n, env)
	case Unary:
		return evalUnary(n, env)
	case Binary:
		return evalBinary(n, env)
	case Assignment:
		return evalAssign(n, env)
	case Let:
		return evalLet(n, env)
	case Const:
		return evalConst(n, env)
	case Increment:
		return evalIncrement(n, env)
	case Decrement:
		return evalDecrement(n, env)
	case If:
		return evalIf(n, env)
	case Switch:
		return evalSwitch(n, env)
	case While:
		return evalWhile(n, env)
	case Do:
		return evalDo(n, env)
	case For:
		return evalFor(n, env)
	case Break:
		return nil, ErrBreak
	case Continue:
		return nil, ErrContinue
	case Try:
		return evalTry(n, env)
	case Throw:
		v, err := eval(n.Node, env)
		if err == nil {
			err = ErrThrow
		}
		return v, err
	case Return:
		v, err := eval(n.Node, env)
		if err == nil {
			err = ErrReturn
		}
		return v, err
	case Call:
		return evalCall(n, env)
	case Func:
		return evalFunc(n, env)
	default:
		return nil, ErrEval
	}
}

func evalBody(b Body, env environ.Environment[Value]) (Value, error) {
	var (
		res Value
		err error
	)
	for _, n := range b.Nodes {
		res, err = eval(n, env)
		if err != nil {
			break
		}
	}
	return res, err
}

func evalFunc(f Func, env environ.Environment[Value]) (Value, error) {
	fn := Function{
		Ident: f.Ident,
		Env:   Enclosed(Default()),
		Body:  f.Body,
		Arrow: f.Arrow,
	}
	for _, a := range f.Args {
		switch a := a.(type) {
		case Identifier:
			p := Parameter{
				Name:  a.Name,
				Value: Void{},
			}
			fn.Args = append(fn.Args, p)
		case Assignment:
			ident, ok := a.Ident.(Identifier)
			if !ok {
				return nil, ErrEval
			}
			val, err := eval(a.Node, env)
			if err != nil {
				return nil, err
			}
			p := Parameter{
				Name:  ident.Name,
				Value: val,
			}
			fn.Args = append(fn.Args, p)
		default:
			return nil, ErrEval
		}
	}
	return fn, env.Define(fn.Ident, fn)
}

func evalTry(t Try, env environ.Environment[Value]) (Value, error) {
	res, err := eval(t.Node, Enclosed(env))
	if err != nil && t.Catch != nil {
		catch, ok := t.Catch.(Catch)
		if !ok {
			return nil, ErrEval
		}
		sub := Enclosed(env)
		ev, err := eval(catch.Err, sub)
		if err != nil {
			return nil, err
		}
		sub.Define("", letValue(ev))
		if _, err := eval(catch.Body, sub); err != nil {
			return nil, err
		}
	}
	if t.Finally != nil {
		if _, err := eval(t.Finally, Enclosed(env)); err != nil {
			return nil, err
		}
	}
	return res, err
}

func evalLet(e Let, env environ.Environment[Value]) (Value, error) {
	a, ok := e.Node.(Assignment)
	if !ok {
		return nil, ErrEval
	}
	ident, ok := a.Ident.(Identifier)
	if !ok {
		return nil, ErrEval
	}
	if _, err := env.Resolve(ident.Name); err == nil {
		return nil, environ.ErrExist
	}
	res, err := eval(a.Node, env)
	if err != nil {
		return nil, err
	}
	return res, env.Define(ident.Name, letValue(res))
}

func evalConst(e Const, env environ.Environment[Value]) (Value, error) {
	a, ok := e.Node.(Assignment)
	if !ok {
		return nil, ErrEval
	}
	ident, ok := a.Ident.(Identifier)
	if !ok {
		return nil, ErrEval
	}
	if _, err := env.Resolve(ident.Name); err == nil {
		return nil, environ.ErrExist
	}
	res, err := eval(a.Node, env)
	if err != nil {
		return nil, err
	}
	return res, env.Define(ident.Name, constValue(res))
}

func evalDo(d Do, env environ.Environment[Value]) (Value, error) {
	var (
		res Value
		err error
		sub = Enclosed(env)
	)
	for {
		err = nil
		res, err = eval(d.Body, Enclosed(sub))
		if err != nil {
			if errors.Is(err, ErrBreak) {
				err = nil
				break
			}
			if errors.Is(err, ErrContinue) {
				continue
			}
			return nil, err
		}
		tmp, err1 := eval(d.Cdt, sub)
		if err1 != nil {
			return nil, err
		}
		if !isTrue(tmp) {
			break
		}
	}
	return res, err
}

func evalWhile(w While, env environ.Environment[Value]) (Value, error) {
	var (
		res Value
		err error
		sub = Enclosed(env)
	)
	for {
		tmp, err1 := eval(w.Cdt, sub)
		if err1 != nil {
			return nil, err
		}
		if !isTrue(tmp) {
			break
		}
		err = nil
		res, err = eval(w.Body, Enclosed(sub))
		if err != nil {
			if errors.Is(err, ErrBreak) {
				err = nil
				break
			}
			if errors.Is(err, ErrContinue) {
				continue
			}
			return nil, err
		}
	}
	return res, err
}

func evalFor(f For, env environ.Environment[Value]) (Value, error) {
	sub := Enclosed(env)
	switch c := f.Ctrl.(type) {
	case OfCtrl:
		return evalForOf(c, f.Body, sub)
	case InCtrl:
		return evalForIn(c, f.Body, sub)
	case ForCtrl:
		return evalForClassic(c, f.Body, sub)
	default:
		return nil, ErrEval
	}
}

func evalForOf(ctrl OfCtrl, body Node, env environ.Environment[Value]) (Value, error) {
	list, err := eval(ctrl.Iter, env)
	if err != nil {
		return nil, err
	}
	it, ok := list.(Iterator)
	if !ok {
		return nil, ErrOp
	}
	var res Value
	for _, v := range it.List() {
		_ = v
		res, err = eval(body, Enclosed(env))
		if err != nil {
			if errors.Is(err, ErrBreak) || errors.Is(err, ErrThrow) {
				it.Return()
			}
			break
		}
	}
	if errors.Is(err, ErrBreak) {
		err = nil
	}
	return res, err
}

func evalForIn(ctrl InCtrl, body Node, env environ.Environment[Value]) (Value, error) {
	list, err := eval(ctrl.Iter, env)
	if err != nil {
		return nil, err
	}
	it, ok := list.(interface{ Values() []Value })
	if !ok {
		return nil, ErrOp
	}
	var res Value
	for _, v := range it.Values() {
		_ = v
		res, err = eval(body, Enclosed(env))
		if err != nil {
			break
		}
	}
	if errors.Is(err, ErrBreak) {
		err = nil
	}
	return res, err
}

func evalForClassic(ctrl ForCtrl, body Node, env environ.Environment[Value]) (Value, error) {
	return nil, nil
}

func evalIf(i If, env environ.Environment[Value]) (Value, error) {
	sub := Enclosed(env)
	res, err := eval(i.Cdt, sub)
	if err != nil {
		return nil, err
	}
	if isTrue(res) {
		return eval(i.Csq, Enclosed(sub))
	}
	if i.Alt == nil {
		return nil, nil
	}
	return eval(i.Alt, Enclosed(sub))
}

func evalSwitch(s Switch, env environ.Environment[Value]) (Value, error) {
	return nil, nil
}

func evalCall(c Call, env environ.Environment[Value]) (Value, error) {
	ident, ok := c.Ident.(Identifier)
	if !ok {
		return nil, ErrEval
	}
	value, err := env.Resolve(ident.Name)
	if err != nil {
		return nil, err
	}
	var args []Value
	for i := range c.Args {
		a, err := eval(c.Args[i], env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	if call, ok := value.(interface{ Call([]Value) (Value, error) }); ok {
		res, err := call.Call(args)
		if errors.Is(err, ErrReturn) {
			err = nil
		}
		return res, err
	}
	return nil, ErrOp
}

func evalGroup(g Group, env environ.Environment[Value]) (Value, error) {
	var (
		res Value
		err error
	)
	for _, n := range g.Nodes {
		res, err = eval(n, env)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func evalMap(a Map, env environ.Environment[Value]) (Value, error) {
	obj := createObject()
	for k, v := range a.Nodes {
		var (
			key Value
			err error
		)
		if i, ok := k.(Identifier); ok {
			key = getString(i.Name)
		} else {
			key, err = eval(k, env)
		}
		if err != nil {
			return nil, err
		}
		switch key.(type) {
		case String, Float, Bool:
		default:
			return nil, ErrEval
		}
		val, err := eval(v, env)
		if err != nil {
			return nil, err
		}
		obj.Fields[key] = val
	}
	return obj, nil
}

func evalList(a List, env environ.Environment[Value]) (Value, error) {
	arr := createArray()
	for _, n := range a.Nodes {
		v, err := eval(n, env)
		if err != nil {
			return nil, err
		}
		arr.Values = append(arr.Values, v)
	}
	return arr, nil
}

func evalAccess(a Access, env environ.Environment[Value]) (Value, error) {
	res, err := eval(a.Node, env)
	if err != nil {
		return nil, err
	}
	if i, ok := a.Ident.(Identifier); ok {
		get, ok := res.(interface{ Get(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return get.Get(getString(i.Name))
	}
	if i, ok := a.Ident.(Call); ok {
		var args []Value
		for j := range i.Args {
			a, err := eval(i.Args[j], env)
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		}
		call, ok := res.(interface {
			Call(string, []Value) (Value, error)
		})
		if !ok {
			return nil, ErrOp
		}
		ident, ok := i.Ident.(Identifier)
		if !ok {
			return nil, ErrEval
		}
		res, err := call.Call(ident.Name, args)
		if errors.Is(err, ErrReturn) {
			err = nil
		}
		return res, err
	}
	return nil, ErrOp
}

func evalIndex(i Index, env environ.Environment[Value]) (Value, error) {
	res, err := eval(i.Ident, env)
	if err != nil {
		return nil, err
	}
	expr, err := eval(i.Expr, env)
	if err != nil {
		return nil, err
	}
	at, ok := res.(interface{ At(Value) (Value, error) })
	if !ok {
		return nil, ErrOp
	}
	return at.At(expr)
}

func evalIdent(i Identifier, env environ.Environment[Value]) (Value, error) {
	v, err := env.Resolve(i.Name)
	if err != nil {
		return nil, err
	}
	if x, ok := v.(envValue); ok {
		v = x.Value
	}
	return v, nil
}

func evalAssign(a Assignment, env environ.Environment[Value]) (Value, error) {
	ident, ok := a.Ident.(Identifier)
	if !ok {
		return nil, ErrEval
	}
	if v, err := env.Resolve(ident.Name); err == nil {
		e, ok := v.(envValue)
		if ok && e.Const {
			return nil, ErrConst
		}
	}
	res, err := eval(a.Node, env)
	if err != nil {
		return nil, err
	}
	return res, env.Define(ident.Name, letValue(res))
}

func evalIncrement(i Increment, env environ.Environment[Value]) (Value, error) {
	ident, ok := i.Node.(Identifier)
	if !ok {
		return nil, ErrEval
	}
	val, err := eval(i.Node, env)
	if err != nil {
		return nil, err
	}
	incr, ok := val.(interface{ Incr() (Value, error) })
	if !ok {
		return nil, ErrOp
	}
	res, err := incr.Incr()
	if err != nil {
		return nil, err
	}
	if err := env.Define(ident.Name, res); err != nil {
		return nil, err
	}
	if !i.Post {
		val = res
	}
	return val, nil
}

func evalDecrement(i Decrement, env environ.Environment[Value]) (Value, error) {
	ident, ok := i.Node.(Identifier)
	if !ok {
		return nil, ErrEval
	}
	val, err := eval(i.Node, env)
	if err != nil {
		return nil, err
	}
	decr, ok := val.(interface{ Decr() (Value, error) })
	if !ok {
		return nil, ErrOp
	}
	res, err := decr.Decr()
	if err != nil {
		return nil, err
	}
	if err := env.Define(ident.Name, res); err != nil {
		return nil, err
	}
	if !i.Post {
		val = res
	}
	return val, nil
}

func evalUnary(u Unary, env environ.Environment[Value]) (Value, error) {
	right, err := eval(u.Node, env)
	if err != nil {
		return nil, err
	}
	switch u.Op {
	default:
		return nil, ErrEval
	case Sub:
		res, ok := right.(interface{ Rev() Value })
		if !ok {
			return nil, ErrOp
		}
		return res.Rev(), nil
	case Add:
		res, ok := right.(interface{ Float() Value })
		if !ok {
			return nil, ErrOp
		}
		return res.Float(), nil
	case Not:
		res, ok := right.(interface{ Not() Value })
		if !ok {
			return nil, ErrOp
		}
		return res.Not(), nil
	}
}

func evalBinary(b Binary, env environ.Environment[Value]) (Value, error) {
	left, err := eval(b.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := eval(b.Right, env)
	if err != nil {
		return nil, err
	}
	switch b.Op {
	default:
		return nil, ErrEval
	case And:
		return getBool(isTrue(left) && isTrue(right)), nil
	case Or:
		return getBool(isTrue(left) || isTrue(right)), nil
	case Eq:
		left, ok := left.(interface{ Equal(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Equal(right)
	case Seq:
		left, ok := left.(interface{ StrictEqual(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.StrictEqual(right)
	case Ne:
		left, ok := left.(interface{ NotEqual(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.NotEqual(right)
	case Sne:
		left, ok := left.(interface{ StrictNotEqual(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.StrictNotEqual(right)
	case Lt:
		left, ok := left.(interface{ LesserThan(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.LesserThan(right)
	case Le:
		left, ok := left.(interface{ LesserEqual(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.LesserEqual(right)
	case Gt:
		left, ok := left.(interface{ GreaterThan(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.GreaterThan(right)
	case Ge:
		left, ok := left.(interface{ GreaterEqual(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.GreaterEqual(right)
	case Add:
		left, ok := left.(interface{ Add(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Add(right)
	case Sub:
		left, ok := left.(interface{ Sub(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Sub(right)
	case Mul:
		left, ok := left.(interface{ Mul(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Mul(right)
	case Div:
		left, ok := left.(interface{ Div(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Div(right)
	case Mod:
		left, ok := left.(interface{ Mod(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Mod(right)
	case Pow:
		left, ok := left.(interface{ Pow(Value) (Value, error) })
		if !ok {
			return nil, ErrOp
		}
		return left.Pow(right)
	}
}

type Node interface{}

type Body struct {
	Nodes []Node
}

type Null struct {
	Position
}

type Undefined struct {
	Position
}

type List struct {
	Position
	Nodes []Node
}

type Map struct {
	Position
	Nodes map[Node]Node
}

type Literal[T string | float64 | bool] struct {
	Value T
	Position
}

type Group struct {
	Nodes []Node
	Position
}

type Identifier struct {
	Name string
	Position
}

type Index struct {
	Ident Node
	Expr  Node
	Position
}

type Access struct {
	Ident Node
	Node
	Position
}

type Unary struct {
	Op rune
	Node
	Position
}

type Binary struct {
	Op    rune
	Left  Node
	Right Node
	Position
}

type Assignment struct {
	Ident Node
	Node
}

type Let struct {
	Node
	Position
}

type Const struct {
	Node
	Position
}

type Increment struct {
	Node
	Post bool
	Position
}

type Decrement struct {
	Node
	Post bool
	Position
}

type If struct {
	Cdt Node
	Csq Node
	Alt Node
	Position
}

type Switch struct {
	Cdt     Node
	Cases   []Node
	Default Node
	Position
}

type Case struct {
	Value Node
	Body  Node
	Position
}

type Do struct {
	Cdt  Node
	Body Node
	Position
}

type While struct {
	Cdt  Node
	Body Node
	Position
}

type OfCtrl struct {
	Ident Node
	Iter  Node
}

type InCtrl struct {
	Ident Node
	Iter  Node
}

type ForCtrl struct {
	Init  Node
	Cdt   Node
	After Node
}

type For struct {
	Ctrl Node
	Body Node
	Position
}

type Break struct {
	Label string
	Position
}

type Continue struct {
	Label string
	Position
}

type Try struct {
	Node
	Catch   Node
	Finally Node
	Position
}

type Catch struct {
	Err  Node
	Body Node
	Position
}

type Throw struct {
	Node
	Position
}

type Return struct {
	Node
	Position
}

type Call struct {
	Ident Node
	Args  []Node
	Position
}

type Func struct {
	Ident string
	Args  []Node
	Body  Node
	Arrow bool
	Position
}

const (
	powLowest int = iota
	powComma
	powAssign
	powOr
	powAnd
	powEq
	powCmp
	powAdd
	powMul
	powPow
	powObject
	powPostfix
	powPrefix
	powAccess
	powGroup
)

var bindings = map[rune]int{
	Comma:    powComma,
	Question: powAssign,
	Assign:   powAssign,
	Colon:    powAssign,
	Keyword:  powAssign,
	Or:       powOr,
	And:      powAnd,
	Eq:       powEq,
	Ne:       powEq,
	Lt:       powCmp,
	Le:       powCmp,
	Gt:       powCmp,
	Ge:       powCmp,
	Add:      powAdd,
	Sub:      powAdd,
	Mul:      powMul,
	Div:      powMul,
	Mod:      powMul,
	Pow:      powPow,
	Lparen:   powGroup,
	Dot:      powAccess,
	Lsquare:  powAccess,
	Lcurly:   powObject,
	Incr:     powPostfix,
	Decr:     powPrefix,
}

type (
	prefixFunc func() (Node, error)
	infixFunc  func(Node) (Node, error)
)

type Parser struct {
	prefix map[rune]prefixFunc
	infix  map[rune]infixFunc

	scan *Scanner
	curr Token
	peek Token
}

func ParseReader(r io.Reader) (Node, error) {
	p := Parse(r)
	return p.Parse()
}

func Parse(r io.Reader) *Parser {
	p := Parser{
		scan:   Scan(r),
		prefix: make(map[rune]prefixFunc),
		infix:  make(map[rune]infixFunc),
	}

	p.registerPrefix(Not, p.parseNot)
	p.registerPrefix(Sub, p.parseRev)
	p.registerPrefix(Add, p.parseFloat)
	p.registerPrefix(Incr, p.parseIncrPrefix)
	p.registerPrefix(Decr, p.parseDecrPrefix)
	p.registerPrefix(Ident, p.parseIdent)
	p.registerPrefix(Text, p.parseString)
	p.registerPrefix(Number, p.parseNumber)
	p.registerPrefix(Boolean, p.parseBoolean)
	p.registerPrefix(Lparen, p.parseGroup)
	p.registerPrefix(Lsquare, p.parseList)
	p.registerPrefix(Lcurly, p.parseMap)
	p.registerPrefix(Keyword, p.parseKeyword)

	p.registerInfix(Dot, p.parseDot)
	p.registerInfix(Assign, p.parseAssign)
	p.registerInfix(Add, p.parseBinary)
	p.registerInfix(Sub, p.parseBinary)
	p.registerInfix(Mul, p.parseBinary)
	p.registerInfix(Div, p.parseBinary)
	p.registerInfix(Mod, p.parseBinary)
	p.registerInfix(Pow, p.parseBinary)
	p.registerInfix(And, p.parseBinary)
	p.registerInfix(Or, p.parseBinary)
	p.registerInfix(Eq, p.parseBinary)
	p.registerInfix(Ne, p.parseBinary)
	p.registerInfix(Lt, p.parseBinary)
	p.registerInfix(Le, p.parseBinary)
	p.registerInfix(Gt, p.parseBinary)
	p.registerInfix(Ge, p.parseBinary)
	p.registerInfix(And, p.parseBinary)
	p.registerInfix(Or, p.parseBinary)
	p.registerInfix(Incr, p.parseIncrPostfix)
	p.registerInfix(Decr, p.parseDecrPostfix)
	p.registerInfix(Arrow, p.parseArrow)
	p.registerInfix(Lparen, p.parseCall)
	p.registerInfix(Lsquare, p.parseIndex)
	p.registerInfix(Question, p.parseTernary)
	p.registerInfix(Keyword, p.parseKeywordCtrl)

	p.next()
	p.next()
	return &p
}

func (p *Parser) Parse() (Node, error) {
	var body Body
	for !p.done() {
		n, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		body.Nodes = append(body.Nodes, n)
		for p.is(EOL) {
			p.next()
		}
	}
	return body, nil
}

func (p *Parser) parseNode() (Node, error) {
	if p.is(Keyword) {
		return p.parseKeyword()
	}
	return p.parseExpression(powLowest)
}

func (p *Parser) parseKeyword() (Node, error) {
	switch p.curr.Literal {
	case "let":
		return p.parseLet()
	case "const":
		return p.parseConst()
	case "if":
		return p.parseIf()
	case "else":
		return p.parseElse()
	case "switch":
		return p.parseSwitch()
	case "for":
		return p.parseFor()
	case "do":
		return p.parseDo()
	case "while":
		return p.parseWhile()
	case "break":
		return p.parseBreak()
	case "continue":
		return p.parseContinue()
	case "return":
		return p.parseReturn()
	case "function":
		return p.parseFunction()
	case "import":
		return p.parseImport()
	case "export":
		return p.parseExport()
	case "try":
		return p.parseTry()
	case "catch":
		return p.parseCatch()
	case "finally":
		return p.parseFinally()
	case "throw":
		return p.parseThrow()
	case "null":
		return p.parseNull()
	case "undefined":
		return p.parseUndefined()
	default:
		return nil, fmt.Errorf("%s: keyword not supported/known", p.curr.Literal)
	}
}

// func (p *Parser) parseKeywordValue() (Node, error) {
// 	switch p.curr.Literal {
// 	case "null":
// 		return p.parseNull()
// 	case "undefined":
// 		return p.parseUndefined()
// 	default:
// 		return nil, fmt.Errorf("%s: keyword not supported/known", p.curr.Literal)
// 	}
// }

func (p *Parser) parseKeywordCtrl(left Node) (Node, error) {
	switch p.curr.Literal {
	case "of":
		expr := OfCtrl{
			Ident: left,
		}
		p.next()
		right, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		expr.Iter = right
		return expr, nil
	case "in":
		expr := InCtrl{
			Ident: left,
		}
		p.next()
		right, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		expr.Iter = right
		return expr, nil
	default:
		return nil, fmt.Errorf("%s: keyword not supported/known", p.curr.Literal)
	}
}

func (p *Parser) parseLet() (Node, error) {
	expr := Let{
		Position: p.curr.Position,
	}
	p.next()
	ident, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	expr.Node = ident
	if p.is(Keyword) {
		return p.parseKeywordCtrl(expr)
	}
	if !p.is(Assign) {
		expr.Node = Assignment{
			Ident: ident,
			Node:  Undefined{},
		}
		return expr, nil
	}
	p.next()
	value, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = Assignment{
		Ident: ident,
		Node:  value,
	}
	return expr, nil
}

func (p *Parser) parseConst() (Node, error) {
	expr := Const{
		Position: p.curr.Position,
	}
	p.next()
	ident, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	if !p.is(Assign) {
		return nil, p.unexpected()
	}
	p.next()
	value, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = Assignment{
		Ident: ident,
		Node:  value,
	}
	return expr, nil
}

func (p *Parser) parseIf() (Node, error) {
	expr := If{
		Position: p.curr.Position,
	}
	p.next()
	cdt, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	expr.Cdt = cdt
	if expr.Csq, err = p.parseBody(); err != nil {
		return nil, err
	}
	if p.is(Keyword) {
		expr.Alt, err = p.parseKeyword()
	}
	return expr, err
}

func (p *Parser) parseElse() (Node, error) {
	p.next()
	if p.is(Keyword) {
		return p.parseKeyword()
	}
	return p.parseBody()
}

func (p *Parser) parseSwitch() (Node, error) {
	return nil, nil
}

func (p *Parser) parseDo() (Node, error) {
	do := Do{
		Position: p.curr.Position,
	}
	p.next()
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	do.Body = body
	if do.Cdt, err = p.parseCondition(); err != nil {
		return nil, err
	}
	return do, nil
}

func (p *Parser) parseWhile() (Node, error) {
	expr := While{
		Position: p.curr.Position,
	}
	p.next()
	cdt, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	expr.Cdt = cdt
	if expr.Body, err = p.parseBody(); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *Parser) parseCondition() (Node, error) {
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return expr, nil
}

func (p *Parser) parseBody() (Node, error) {
	if !p.is(Lcurly) {
		return p.parseExpression(powLowest)
	}
	p.next()
	var b Body
	for !p.done() && !p.is(Rcurly) {
		p.skip(p.eol)
		n, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		b.Nodes = append(b.Nodes, n)
		p.skip(p.eol)
	}
	if !p.is(Rcurly) {
		return nil, p.unexpected()
	}
	p.next()
	p.skip(p.eol)
	return b, nil
}

func (p *Parser) parseFor() (Node, error) {
	loop := For{
		Position: p.curr.Position,
	}
	p.next()
	ctrl, err := p.parseForControl()
	if err != nil {
		return nil, err
	}
	loop.Ctrl = ctrl

	if loop.Body, err = p.parseBody(); err != nil {
		return nil, err
	}
	return loop, nil
}

func (p *Parser) parseForControl() (Node, error) {
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()

	var ctrl ForCtrl
	if !p.is(EOL) {
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		switch expr.(type) {
		case OfCtrl, InCtrl:
			if !p.is(Rparen) {
				return nil, p.unexpected()
			}
			p.next()
			return expr, nil
		default:
			ctrl.Init = expr
		}
	}
	if !p.is(EOL) {
		return nil, p.unexpected()
	}
	p.next()
	if !p.is(EOL) {
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		ctrl.Cdt = expr
	}
	if !p.is(EOL) {
		return nil, p.unexpected()
	}
	p.next()
	if !p.is(Rparen) {
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		ctrl.After = expr
	}

	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return ctrl, nil
}

func (p *Parser) parseBreak() (Node, error) {
	expr := Break{
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Ident) {
		expr.Label = p.curr.Literal
		p.next()
	}
	return expr, nil
}

func (p *Parser) parseContinue() (Node, error) {
	expr := Continue{
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Ident) {
		expr.Label = p.curr.Literal
		p.next()
	}
	return expr, nil
}

func (p *Parser) parseReturn() (Node, error) {
	expr := Return{
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseArrow(left Node) (Node, error) {
	return nil, nil
}

func (p *Parser) parseFunction() (Node, error) {
	fn := Func{
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Ident) {
		fn.Ident = p.curr.Literal
		p.next()
	}
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		arg, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, arg)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	return fn, nil
}

func (p *Parser) parseImport() (Node, error) {
	return nil, nil
}

func (p *Parser) parseExport() (Node, error) {
	return nil, nil
}

func (p *Parser) parseTry() (Node, error) {
	try := Try{
		Position: p.curr.Position,
	}
	p.next()
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	try.Node = body
	if p.is(Keyword) && p.curr.Literal == "catch" {
		try.Catch, err = p.parseKeyword()
		if err != nil {
			return nil, err
		}
	}
	if p.is(Keyword) && p.curr.Literal == "finally" {
		try.Finally, err = p.parseKeyword()
		if err != nil {
			return nil, err
		}
	}
	return try, nil
}

func (p *Parser) parseCatch() (Node, error) {
	catch := Catch{
		Position: p.curr.Position,
	}
	p.next()
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}
	catch.Err = ident
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	catch.Body, err = p.parseBody()
	return catch, err
}

func (p *Parser) parseFinally() (Node, error) {
	p.next()
	return p.parseBody()
}

func (p *Parser) parseThrow() (Node, error) {
	t := Throw{
		Position: p.curr.Position,
	}
	p.next()
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	t.Node = expr
	return t, nil
}

func (p *Parser) parseNull() (Node, error) {
	defer p.next()
	expr := Null{
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseUndefined() (Node, error) {
	defer p.next()
	expr := Undefined{
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseExpression(pow int) (Node, error) {
	fn, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, fmt.Errorf("unknown prefix expression %s", p.curr)
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for !p.done() && !p.eol() && pow < p.power() {
		fn, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("unknown infix expression %s", p.curr)
		}
		if left, err = fn(left); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseNot() (Node, error) {
	expr := Unary{
		Op:       Not,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseFloat() (Node, error) {
	expr := Unary{
		Op:       Add,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseRev() (Node, error) {
	expr := Unary{
		Op:       Sub,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseIncrPrefix() (Node, error) {
	incr := Increment{
		Position: p.curr.Position,
	}
	p.next()
	right, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	incr.Node = right
	return incr, nil
}

func (p *Parser) parseDecrPrefix() (Node, error) {
	decr := Decrement{
		Position: p.curr.Position,
	}
	p.next()
	right, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	decr.Node = right
	return decr, nil
}

func (p *Parser) parseIncrPostfix(left Node) (Node, error) {
	incr := Increment{
		Node:     left,
		Post:     true,
		Position: p.curr.Position,
	}
	p.next()
	return incr, nil
}

func (p *Parser) parseDecrPostfix(left Node) (Node, error) {
	decr := Decrement{
		Node:     left,
		Post:     true,
		Position: p.curr.Position,
	}
	p.next()
	return decr, nil
}

func (p *Parser) parseIdent() (Node, error) {
	defer p.next()
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	expr := Identifier{
		Name:     p.curr.Literal,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseString() (Node, error) {
	defer p.next()
	if !p.is(Text) {
		return nil, p.unexpected()
	}
	expr := Literal[string]{
		Value:    p.curr.Literal,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseNumber() (Node, error) {
	if !p.is(Number) {
		return nil, p.unexpected()
	}
	n, err := strconv.ParseFloat(p.curr.Literal, 64)
	if err != nil {
		return nil, err
	}
	defer p.next()
	expr := Literal[float64]{
		Value:    n,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseBoolean() (Node, error) {
	if !p.is(Boolean) {
		return nil, p.unexpected()
	}
	n, err := strconv.ParseBool(p.curr.Literal)
	if err != nil {
		return nil, err
	}
	defer p.next()
	expr := Literal[bool]{
		Value:    n,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseList() (Node, error) {
	if !p.is(Lsquare) {
		return nil, p.unexpected()
	}
	list := List{
		Position: p.curr.Position,
	}
	p.next()
	for !p.done() && !p.is(Rsquare) {
		p.skip(p.eol)
		n, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		list.Nodes = append(list.Nodes, n)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rsquare):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rsquare) {
		return nil, fmt.Errorf("missing ] at end of array")
	}
	p.next()
	return list, nil
}

func (p *Parser) parseKey() (Node, error) {
	if p.is(Text) {
		return p.parseString()
	}
	if p.is(Number) {
		return p.parseNumber()
	}
	if p.is(Boolean) {
		return p.parseBoolean()
	}

	if !p.is(Ident) {
		return nil, p.unexpected()
	}

	ident := Identifier{
		Name:     p.curr.Literal,
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Comma) || p.is(Colon) {
		return ident, nil
	}
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()

	fn := Func{
		Ident:    ident.Name,
		Position: ident.Position,
	}

	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		arg, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, arg)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	return fn, nil
}

func (p *Parser) parseMap() (Node, error) {
	if !p.is(Lcurly) {
		return nil, p.unexpected()
	}
	obj := Map{
		Position: p.curr.Position,
		Nodes:    make(map[Node]Node),
	}
	p.next()
	for !p.done() && !p.is(Rcurly) {
		p.skip(p.eol)
		key, err := p.parseKey()
		if err != nil {
			return nil, err
		}
		if p.is(Comma) || p.is(Rcurly) {
			val := key
			if fn, ok := key.(Func); ok {
				key = Identifier{
					Name: fn.Ident,
				}
				val = fn
			}
			obj.Nodes[key] = val
			if p.is(Comma) {
				p.next()
				p.skip(p.eol)
			}
			continue
		}
		if !p.is(Colon) {
			return nil, p.unexpected()
		}
		p.next()
		val, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		obj.Nodes[key] = val
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rcurly):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rcurly) {
		return nil, p.unexpected()
	}
	p.next()
	p.skip(p.eol)
	return obj, nil
}

func (p *Parser) parseGroup() (Node, error) {
	grp := Group{
		Position: p.curr.Position,
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		node, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		grp.Nodes = append(grp.Nodes, node)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return grp, nil
}

func (p *Parser) parseDot(left Node) (Node, error) {
	access := Access{
		Node:     left,
		Position: p.curr.Position,
	}
	p.next()
	expr, err := p.parseExpression(powAccess)
	if err != nil {
		return nil, err
	}
	access.Ident = expr
	return access, nil
}

func (p *Parser) parseAssign(left Node) (Node, error) {
	expr := Assignment{
		Ident: left,
	}
	p.next()
	right, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = right
	return expr, nil
}

func (p *Parser) parseBinary(left Node) (Node, error) {
	expr := Binary{
		Left:     left,
		Op:       p.curr.Type,
		Position: p.curr.Position,
	}
	p.next()

	right, err := p.parseExpression(bindings[expr.Op])
	if err != nil {
		return nil, err
	}
	expr.Right = right
	return expr, nil
}

func (p *Parser) parseTernary(left Node) (Node, error) {
	expr := If{
		Cdt:      left,
		Position: p.curr.Position,
	}
	p.next()
	csq, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	if !p.is(Colon) {
		return nil, p.unexpected()
	}
	p.next()

	alt, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	expr.Csq = csq
	expr.Alt = alt
	return expr, nil
}

func (p *Parser) parseIndex(left Node) (Node, error) {
	ix := Index{
		Position: p.curr.Position,
		Ident:    left,
	}
	p.next()
	x, err := p.parseExpression(powAccess)
	if err != nil {
		return nil, err
	}
	ix.Expr = x
	if !p.is(Rsquare) {
		return nil, p.unexpected()
	}
	p.next()
	return ix, nil
}

func (p *Parser) parseCall(left Node) (Node, error) {
	call := Call{
		Ident:    left,
		Position: p.curr.Position,
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		a, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, a)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return call, nil
}

func (p *Parser) registerPrefix(kind rune, fn prefixFunc) {
	p.prefix[kind] = fn
}

func (p *Parser) registerInfix(kind rune, fn infixFunc) {
	p.infix[kind] = fn
}

func (p *Parser) power() int {
	pow, ok := bindings[p.curr.Type]
	if !ok {
		return powLowest
	}
	return pow
}

func (p *Parser) skip(ok func() bool) {
	for ok() {
		p.next()
	}
}

func (p *Parser) eol() bool {
	return p.is(EOL)
}

func (p *Parser) done() bool {
	return p.is(EOF)
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) unexpected() error {
	pos := p.curr.Position
	return fmt.Errorf("%s unexpected token at %d:%d", p.curr, pos.Line, pos.Column)
}

const (
	EOF = -(iota + 1)
	EOL
	Keyword
	Ident
	Text
	Number
	Boolean
	Invalid
	Incr
	Decr
	Add
	Sub
	Mul
	Div
	Mod
	Pow
	Assign
	Not
	Eq
	Seq
	Ne
	Sne
	Lt
	Le
	Gt
	Ge
	And
	Or
	Arrow
	Dot
	Comma
	Question
	Colon
	Lparen
	Rparen
	Lsquare
	Rsquare
	Lcurly
	Rcurly
)

var keywords = []string{
	"let",
	"const",
	"break",
	"continue",
	"for",
	"in",
	"of",
	"if",
	"else",
	"switch",
	"case",
	"default",
	"while",
	"function",
	"return",
	"import",
	"export",
	"from",
	"as",
	"true",
	"false",
	"try",
	"catch",
	"finally",
	"throw",
	"null",
	"undefined",
	"instanceof",
	"typeof",
}

type Position struct {
	Line   int
	Column int
}

type Token struct {
	Literal string
	Type    rune
	Position
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case EOL:
		return "<eol>"
	case Dot:
		return "<dot>"
	case Arrow:
		return "<arrow>"
	case Comma:
		return "<comma>"
	case Lparen:
		return "<lparen>"
	case Rparen:
		return "<rparen>"
	case Lsquare:
		return "<lsquare>"
	case Rsquare:
		return "<rsquare>"
	case Lcurly:
		return "<lcurly>"
	case Rcurly:
		return "<rcurly>"
	case Incr:
		return "<incr>"
	case Decr:
		return "<decr>"
	case Add:
		return "<add>"
	case Sub:
		return "<sub>"
	case Mul:
		return "<mul>"
	case Div:
		return "<div>"
	case Mod:
		return "<mod>"
	case Pow:
		return "<pow>"
	case Assign:
		return "<assign>"
	case Not:
		return "<not>"
	case Eq:
		return "<eq>"
	case Seq:
		return "<seq>"
	case Ne:
		return "<ne>"
	case Sne:
		return "<sne>"
	case Lt:
		return "<lt>"
	case Le:
		return "<le>"
	case Gt:
		return "<gt>"
	case Ge:
		return "<ge>"
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case Question:
		return "<question>"
	case Colon:
		return "<colon>"
	case Keyword:
		prefix = "keyword"
	case Boolean:
		prefix = "boolean"
	case Ident:
		prefix = "identifier"
	case Text:
		prefix = "string"
	case Number:
		prefix = "number"
	case Invalid:
		prefix = "invalid"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

type cursor struct {
	char rune
	curr int
	next int
	Position
}

type Scanner struct {
	input []byte
	cursor
	old cursor

	quoted bool
	str    bytes.Buffer
}

func Scan(r io.Reader) *Scanner {
	buf, _ := io.ReadAll(r)
	buf, _ = bytes.CutPrefix(buf, []byte{0xef, 0xbb, 0xbf})
	s := Scanner{
		input: buf,
	}
	s.cursor.Line = 1
	s.read()
	s.skip(isBlank)
	return &s
}

func (s *Scanner) Scan() Token {
	defer s.reset()

	var tok Token
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}

	s.skip(isSpace)
	switch {
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	case isOperator(s.char):
		s.scanOperator(&tok)
	case isNL(s.char):
		s.scanNL(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	default:
		tok.Type = Invalid
	}

	return tok
}

func (s *Scanner) scanIdent(tok *Token) {
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Ident

	if slices.Contains(keywords, tok.Literal) {
		tok.Type = Keyword
		if tok.Literal == "true" || tok.Literal == "false" {
			tok.Type = Boolean
		}
	}
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for !s.done() && s.char != quote {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Text
	if !isQuote(s.char) && s.char != quote {
		tok.Type = Invalid
		return
	}
	s.read()
}

func (s *Scanner) scanNumber(tok *Token) {
	for isDigit(s.char) && !s.done() {
		s.write()
		s.read()
	}
	if s.char == dot {
		s.write()
		s.read()
		for isDigit(s.char) && !s.done() {
			s.write()
			s.read()
		}
	}
	tok.Literal = s.literal()
	tok.Type = Number
}

func (s *Scanner) scanPunct(tok *Token) {
	switch s.char {
	case lparen:
		tok.Type = Lparen
	case rparen:
		tok.Type = Rparen
	case lsquare:
		tok.Type = Lsquare
	case rsquare:
		tok.Type = Rsquare
	case lcurly:
		tok.Type = Lcurly
	case rcurly:
		tok.Type = Rcurly
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
	}
}

func (s *Scanner) scanOperator(tok *Token) {
	switch s.char {
	case plus:
		tok.Type = Add
		if s.peek() == plus {
			s.read()
			tok.Type = Incr
		}
	case minus:
		tok.Type = Sub
		if s.peek() == minus {
			s.read()
			tok.Type = Decr
		}
	case star:
		tok.Type = Mul
		if s.peek() == star {
			s.read()
			tok.Type = Pow
		}
	case slash:
		tok.Type = Div
	case percent:
		tok.Type = Mod
	case equal:
		tok.Type = Assign
		if s.peek() == rangle {
			s.read()
			tok.Type = Arrow
		}
		if s.peek() == equal {
			s.read()
			tok.Type = Eq
		}
		if s.peek() == equal && tok.Type == Eq {
			s.read()
			tok.Type = Seq
		}
	case bang:
		tok.Type = Not
		if s.peek() == equal {
			s.read()
			tok.Type = Ne
		}
		if s.peek() == equal && tok.Type == Ne {
			s.read()
			tok.Type = Sne
		}
	case langle:
		tok.Type = Lt
		if s.peek() == equal {
			s.read()
			tok.Type = Le
		}
	case rangle:
		tok.Type = Gt
		if s.peek() == equal {
			s.read()
			tok.Type = Ge
		}
	case ampersand:
		tok.Type = Invalid
		if s.peek() == ampersand {
			s.read()
			tok.Type = And
		}
	case pipe:
		tok.Type = Invalid
		if s.peek() == pipe {
			s.read()
			tok.Type = Or
		}
	case question:
		tok.Type = Question
	case colon:
		tok.Type = Colon
	case dot:
		tok.Type = Dot
	case comma:
		tok.Type = Comma
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
	}
}

func (s *Scanner) scanNL(tok *Token) {
	s.skip(isBlank)
	tok.Type = EOL
}

func (s *Scanner) done() bool {
	return s.char == utf8.RuneError || s.char == 0
}

func (s *Scanner) read() {
	if s.curr >= len(s.input) {
		s.char = utf8.RuneError
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = r
		s.next = len(s.input)
		return
	}
	s.old.Position = s.cursor.Position
	if r == nl {
		s.cursor.Line++
		s.cursor.Column = 0
	}
	s.cursor.Column++
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) reset() {
	s.str.Reset()
}

func (s *Scanner) write() {
	s.str.WriteRune(s.char)
}

func (s *Scanner) literal() string {
	return s.str.String()
}

func (s *Scanner) skip(accept func(rune) bool) {
	if s.done() {
		return
	}
	for accept(s.char) && !s.done() {
		s.read()
	}
}

func (s *Scanner) save() {
	s.old = s.cursor
}

func (s *Scanner) restore() {
	s.cursor = s.old
}

const (
	space      = ' '
	tab        = '\t'
	nl         = '\n'
	cr         = '\r'
	dot        = '.'
	squote     = '\''
	dquote     = '"'
	underscore = '_'
	pipe       = '|'
	ampersand  = '&'
	equal      = '='
	plus       = '+'
	minus      = '-'
	star       = '*'
	slash      = '/'
	percent    = '%'
	bang       = '!'
	comma      = ','
	langle     = '<'
	rangle     = '>'
	lparen     = '('
	rparen     = ')'
	lsquare    = '['
	rsquare    = ']'
	lcurly     = '{'
	rcurly     = '}'
	semi       = ';'
	question   = '?'
	colon      = ':'
)

func isDelim(r rune) bool {
	return isBlank(r) || isOperator(r) || isPunct(r)
}

func isPunct(r rune) bool {
	return r == lparen || r == rparen ||
		r == lcurly || r == rcurly ||
		r == lsquare || r == rsquare
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r) || r == underscore
}

func isQuote(r rune) bool {
	return isSingle(r) || isDouble(r)
}

func isDouble(r rune) bool {
	return r == dquote
}

func isSingle(r rune) bool {
	return r == squote
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isNL(r rune) bool {
	return r == nl || r == cr || r == semi
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}

func isOperator(r rune) bool {
	return r == equal || r == ampersand || r == pipe ||
		r == plus || r == minus || r == star || r == slash ||
		r == bang || r == langle || r == rangle || r == percent ||
		r == question || r == colon || r == dot || r == comma
}
