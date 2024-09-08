package play

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type Void struct{}

func isUndefined(v Value) bool {
	_, ok := v.(Void)
	return ok
}

func (_ Void) Type() string {
	return "undefined"
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

func (_ Nil) Type() string {
	return "null"
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

func (_ Float) Type() string {
	return "number"
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

func (_ Bool) Type() string {
	return "boolean"
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

func (_ String) Type() string {
	return "string"
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
	var fn func([]Value) (Value, error)
	switch ident {
	case "concat":
		fn = s.concat
	case "endsWith":
		fn = s.endsWith
	case "includes":
		fn = s.includes
	case "indexOf":
		fn = s.indexOf
	case "lastIndexOf":
		fn = s.lastIndexOf
	case "padEnd":
		fn = s.padEnd
	case "padStart":
		fn = s.padStart
	case "repeat":
		fn = s.repeat
	case "replace":
	case "replaceAll":
	case "slice":
	case "split":
	case "startsWith":
		fn = s.startsWith
	case "substring":
	case "toLowerCase":
		fn = s.toLowerCase
	case "toUpperCase":
		fn = s.toUpperCase
	case "trim":
		fn = s.trim
	case "trimEnd":
		fn = s.trimEnd
	case "trimStart":
		fn = s.trimStart
	default:
		return nil, fmt.Errorf("%s: undefined function", ident)
	}
	if fn == nil {
		return nil, ErrImpl
	}
	return fn(args)
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

func (s String) concat(args []Value) (Value, error) {
	var str strings.Builder
	str.WriteString(s.value)
	for _, a := range args {
		s, ok := a.(fmt.Stringer)
		if !ok {
			continue
		}
		str.WriteString(s.String())
	}
	return getString(str.String()), nil
}

func (s String) endsWith(args []Value) (Value, error) {
	size := len(s.value)
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		size = int(x.value)
		if size >= len(s.value) {
			size = len(s.value)
		}
	}
	if len(args) >= 1 {
		str, ok := args[0].(String)
		if !ok {
			return nil, ErrOp
		}
		ok = strings.HasSuffix(s.value[:size], str.value)
		return getBool(ok), nil
	}
	return getBool(false), nil
}

func (s String) includes(args []Value) (Value, error) {
	var position int
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		position = int(x.value)
		if position < 0 {
			position = len(s.value) + position
		}
	}
	str, ok := args[0].(String)
	if !ok {
		return nil, ErrOp
	}
	ok = strings.Contains(s.value[position:], str.value)
	return getBool(ok), nil
}

func (s String) indexOf(args []Value) (Value, error) {
	if len(args) == 0 {
		return getFloat(-1), nil
	}
	var position int
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		position = int(x.value)
		if position < 0 {
			position = len(s.value) + position
		}
	}
	str, ok := args[0].(String)
	if !ok {
		return nil, ErrOp
	}
	ix := strings.Index(s.value[position:], str.value)
	return getFloat(float64(ix)), nil
}

func (s String) lastIndexOf(args []Value) (Value, error) {
	if len(args) == 0 {
		return getFloat(-1), nil
	}
	var position int
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		position = int(x.value)
		if position < 0 {
			position = len(s.value) + position
		}
	}
	str, ok := args[0].(String)
	if !ok {
		return nil, ErrOp
	}
	ix := strings.LastIndex(s.value[position:], str.value)
	return getFloat(float64(ix)), nil
}

func (s String) padEnd(args []Value) (Value, error) {
	if len(args) == 0 {
		return s, nil
	}
	var (
		char = " "
		size int
	)
	if len(args) >= 1 {
		x, ok := args[0].(Float)
		if !ok {
			return nil, ErrOp
		}
		size = int(x.value)
		if size < 0 {
			return nil, ErrEval
		}
	}
	if len(args) >= 2 {
		x, ok := args[1].(String)
		if ok {
			char = x.value
		}
	}
	if len(s.value) >= size {
		return s, nil
	}
	var (
		delta = size - len(s.value)
		prefix = strings.Repeat(char, delta)
	)
	return getString(s.value + prefix), nil
}

func (s String) padStart(args []Value) (Value, error) {
	if len(args) == 0 {
		return s, nil
	}
	var (
		char = " "
		size int
	)
	if len(args) >= 1 {
		x, ok := args[0].(Float)
		if !ok {
			return nil, ErrOp
		}
		size = int(x.value)
		if size < 0 {
			return nil, ErrEval
		}
	}
	if len(args) >= 2 {
		x, ok := args[1].(String)
		if ok {
			char = x.value
		}
	}
	if len(s.value) >= size {
		return s, nil
	}
	var (
		delta = size - len(s.value)
		prefix = strings.Repeat(char, delta)
	)
	return getString(prefix + s.value), nil
}

func (s String) repeat(args []Value) (Value, error) {
	if len(args) == 0 {
		return Void{}, nil
	}
	x, ok := args[0].(Float)
	if !ok {
		return nil, ErrEval
	}
	if x.value < 0 {
		return nil, ErrEval
	}
	return getString(strings.Repeat(s.value, int(x.value))), nil
}

func (s String) replace(args []Value) (Value, error) {
	return nil, nil
}

func (s String) replaceAll(args []Value) (Value, error) {
	return nil, nil
}

func (s String) slice(args []Value) (Value, error) {
	return nil, nil
}

func (s String) split(args []Value) (Value, error) {
	return nil, nil
}

func (s String) startsWith(args []Value) (Value, error) {
	size := len(s.value)
	if len(args) >= 2 {
		x, ok := args[1].(Float)
		if !ok {
			return nil, ErrType
		}
		size = int(x.value)
		if size >= len(s.value) {
			size = len(s.value)
		}
	}
	if len(args) >= 1 {
		str, ok := args[0].(String)
		if !ok {
			return nil, ErrOp
		}
		ok = strings.HasPrefix(s.value[:size], str.value)
		return getBool(ok), nil
	}
	return getBool(false), nil
}

func (s String) substring(args []Value) (Value, error) {
	return nil, nil
}

func (s String) toLowerCase(args []Value) (Value, error) {
	str := strings.ToLower(s.value)
	return getString(str), nil
}

func (s String) toUpperCase(args []Value) (Value, error) {
	str := strings.ToUpper(s.value)
	return getString(str), nil
}

func (s String) trim(args []Value) (Value, error) {
	str := strings.TrimSpace(s.value)
	return getString(str), nil
}

func (s String) trimEnd(args []Value) (Value, error) {
	str := strings.TrimRightFunc(s.value, unicode.IsSpace)
	return getString(str), nil
}

func (s String) trimStart(args []Value) (Value, error) {
	str := strings.TrimLeftFunc(s.value, unicode.IsSpace)
	return getString(str), nil
}
