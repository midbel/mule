package play

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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

func NewFloat(val float64) Value {
	return getFloat(val)
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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

func NewBool(val bool) Value {
	return getBool(val)
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

func NewString(str string) Value {
	return getString(str)
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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
		return nil, ErrType
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

func (s String) Call(ident string, args []Value) (Value, error) {
	switch ident {
	case "concat":
	case "endsWith":
	case "includes":
	case "indexOf":
	case "lastIndexOf":
	case "padEnd":
	case "padStart":
	case "repeat":
	case "replace":
	case "replaceAll":
	case "slice":
	case "split":
	case "startsWith":
	case "substring":
	case "toLowerCase":
	case "toUpperCase":
	case "trim":
	case "trimEnd":
	case "trimStart":
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	return nil, ErrImpl
}

func (s String) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch name := str.String(); name {
	case "length":
		n := len(s.value)
		return getFloat(float64(n)), nil
	default:
		return Void{}, fmt.Errorf("%s: undefined property", str)
	}
}

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

type Date struct {
	value time.Time
}

func (d *Date) True() Value {
	return getBool(true)
}

func (d *Date) String() string {
	return "date"
}

func (d *Date) Call(ident string, args []Value) (Value, error) {
	return Void{}, ErrImpl
}

type Url struct {
	value *url.URL
}

func NewURL(str *url.URL) Value {
	return &Url{
		value: str,
	}
}

func (u *Url) True() Value {
	return getBool(true)
}

func (u *Url) String() string {
	return u.value.String()
}

func (u *Url) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch name := str.String(); name {
	case "host", "hostname":
		return getString(u.value.Hostname()), nil
	case "port":
		return getString(u.value.Port()), nil
	case "path":
		return getString(u.value.Path), nil
	case "query":
		return getString(u.value.RawQuery), nil
	case "scheme":
		return getString(u.value.Scheme), nil
	default:
		return nil, fmt.Errorf("%s: undefined property", name)
	}
}

type Math struct{}

func (m Math) String() string {
	return "Math"
}

func (m Math) True() Value {
	return getBool(true)
}

func (m Math) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch prop := str.String(); prop {
	case "PI":
		return getFloat(math.Pi), nil
	default:
		return nil, fmt.Errorf("%s: property not known", prop)
	}
}

func (m Math) Call(ident string, args []Value) (Value, error) {
	switch ident {
	case "abs":
	case "ceil":
	case "cos":
	case "exp":
	case "floor":
	case "log":
	case "round":
	case "max":
	case "min":
	case "pow":
	case "random":
	case "sin":
	case "tan":
	case "trunc":
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	return nil, ErrImpl
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
