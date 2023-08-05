package eval

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	ErrIncompatible = errors.New("incompatible type")
	ErrOperation    = errors.New("unsupported operation")
	ErrZero         = errors.New("division by zero")
	ErrAssert       = errors.New("assertion failed")
)

type adder interface {
	Add(Object) (Object, error)
}

type suber interface {
	Sub(Object) (Object, error)
}

type muler interface {
	Mul(Object) (Object, error)
}

type diver interface {
	Div(Object) (Object, error)
}

type moder interface {
	Mod(Object) (Object, error)
}

type power interface {
	Pow(Object) (Object, error)
}

type Object interface {
	Not() (Object, error)
	True() bool
	Raw() any
}

func CreateObject(value any) (Object, error) {
	switch v := value.(type) {
	case string:
		return CreateString(v), nil
	case float64:
		return CreateReal(v), nil
	case bool:
		return CreateBool(v), nil
	default:
		return nil, fmt.Errorf("%s can not be transformed to Object")
	}
}

func leftAndRight(left, right Object) (Object, error) {
	b := left.True() && right.True()
	return CreateBool(b), nil
}

func leftOrRight(left, right Object) (Object, error) {
	b := left.True() || right.True()
	return CreateBool(b), nil
}

type real struct {
	value float64
}

func CreateReal(f float64) Object {
	return real{
		value: f,
	}
}

func (f real) Raw() any {
	return f.value
}

func (f real) Rev() (Object, error) {
	f.value = -f.value
	return f, nil
}

func (f real) Not() (Object, error) {
	return CreateBool(!f.True()), nil
}

func (f real) String() string {
	return strconv.FormatFloat(f.value, 'g', -1, 64)
}

func (f real) Add(other Object) (Object, error) {
	switch x := other.(type) {
	case real:
		f.value += x.value
	case varchar:
		s := f.String() + x.String()
		return CreateString(s), nil
	default:
		return nil, incompatibleType("addition", f, other)
	}
	return f, nil
}

func (f real) Sub(other Object) (Object, error) {
	switch x := other.(type) {
	case real:
		f.value -= x.value
	default:
		return nil, incompatibleType("subtraction", f, other)
	}
	return f, nil
}

func (f real) Div(other Object) (Object, error) {
	switch x := other.(type) {
	case real:
		if x.value == 0 {
			return nil, ErrZero
		}
		f.value /= x.value
	default:
		return nil, incompatibleType("division", f, other)
	}
	return f, nil
}

func (f real) Mul(other Object) (Object, error) {
	switch x := other.(type) {
	case real:
		f.value *= x.value
	default:
		return nil, incompatibleType("multiply", f, other)
	}
	return f, nil
}

func (f real) Mod(other Object) (Object, error) {
	switch x := other.(type) {
	case real:
		if x.value == 0 {
			return nil, ErrZero
		}
		f.value = math.Mod(f.value, x.value)
	default:
		return nil, incompatibleType("modulo", f, other)
	}
	return f, nil
}

func (f real) Pow(other Object) (Object, error) {
	switch x := other.(type) {
	case real:
		f.value = math.Pow(f.value, x.value)
	default:
		return nil, incompatibleType("power", f, other)
	}
	return f, nil
}

func (f real) True() bool {
	return f.value != 0
}

func (f real) Eq(other Object) (Object, error) {
	x, ok := other.(real)
	if !ok {
		return nil, incompatibleType("eq", f, other)
	}
	return CreateBool(f.value == x.value), nil
}

func (f real) Ne(other Object) (Object, error) {
	x, ok := other.(real)
	if !ok {
		return nil, incompatibleType("ne", f, other)
	}
	return CreateBool(f.value != x.value), nil
}

func (f real) Lt(other Object) (Object, error) {
	x, ok := other.(real)
	if !ok {
		return nil, incompatibleType("lt", f, other)
	}
	return CreateBool(f.value < x.value), nil
}

func (f real) Le(other Object) (Object, error) {
	x, ok := other.(real)
	if !ok {
		return nil, incompatibleType("le", f, other)
	}
	return CreateBool(f.value <= x.value), nil
}

func (f real) Gt(other Object) (Object, error) {
	x, ok := other.(real)
	if !ok {
		return nil, incompatibleType("gt", f, other)
	}
	return CreateBool(f.value > x.value), nil
}

func (f real) Ge(other Object) (Object, error) {
	x, ok := other.(real)
	if !ok {
		return nil, incompatibleType("ge", f, other)
	}
	return CreateBool(f.value >= x.value), nil
}

type varchar struct {
	str string
}

func CreateString(str string) Object {
	return varchar{
		str: str,
	}
}

func (s varchar) Len() int {
	return len(s.str)
}

func (s varchar) Raw() any {
	return s.str
}

func (s varchar) String() string {
	return s.str
}

func (s varchar) Rev() (Object, error) {
	return nil, unsupportedOp("reverse", s)
}

func (s varchar) Not() (Object, error) {
	return CreateBool(!s.True()), nil
}

func (s varchar) Add(other Object) (Object, error) {
	var str string
	switch x := other.(type) {
	case real:
		str = x.String()
	case varchar:
		str = x.String()
	default:
		return nil, incompatibleType("addition", s, other)
	}
	s.str += str
	return s, nil
}

func (s varchar) Sub(other Object) (Object, error) {
	var part int
	switch x := other.(type) {
	case real:
		part = int(x.value)
	default:
		return nil, incompatibleType("subtraction", s, other)
	}
	if part > len(s.str) {
		s.str = ""
		return s, nil
	}
	if part < 0 {
		s.str = s.str[-part:]
	} else {
		s.str = s.str[:part]
	}
	return s, nil
}

func (s varchar) Div(other Object) (Object, error) {
	var part int
	switch x := other.(type) {
	case real:
		part = int(x.value)
	default:
		return nil, incompatibleType("division", s, other)
	}
	if part == 0 {
		return s, nil
	}
	offset := len(s.str) / part
	s.str = s.str[:offset]
	return s, nil
}

func (s varchar) Mul(other Object) (Object, error) {
	var count int
	switch x := other.(type) {
	case real:
		count = int(x.value)
	default:
		return nil, incompatibleType("multiply", s, other)
	}
	s.str = strings.Repeat(s.str, count)
	return s, nil
}

func (s varchar) Mod(_ Object) (Object, error) {
	return nil, unsupportedOp("modulo", s)
}

func (s varchar) Pow(_ Object) (Object, error) {
	return nil, unsupportedOp("power", s)
}

func (s varchar) True() bool {
	return s.str != ""
}

func (s varchar) Eq(other Object) (Object, error) {
	x, ok := other.(varchar)
	if !ok {
		return nil, incompatibleType("eq", s, other)
	}
	return CreateBool(s.str == x.str), nil
}

func (s varchar) Ne(other Object) (Object, error) {
	x, ok := other.(varchar)
	if !ok {
		return nil, incompatibleType("ne", s, other)
	}
	return CreateBool(s.str != x.str), nil
}

func (s varchar) Lt(other Object) (Object, error) {
	x, ok := other.(varchar)
	if !ok {
		return nil, incompatibleType("lt", s, other)
	}
	return CreateBool(s.str < x.str), nil
}

func (s varchar) Le(other Object) (Object, error) {
	x, ok := other.(varchar)
	if !ok {
		return nil, incompatibleType("le", s, other)
	}
	return CreateBool(s.str <= x.str), nil
}

func (s varchar) Gt(other Object) (Object, error) {
	x, ok := other.(varchar)
	if !ok {
		return nil, incompatibleType("gt", s, other)
	}
	return CreateBool(s.str > x.str), nil
}

func (s varchar) Ge(other Object) (Object, error) {
	x, ok := other.(varchar)
	if !ok {
		return nil, incompatibleType("ge", s, other)
	}
	return CreateBool(s.str >= x.str), nil
}

type boolean struct {
	value bool
}

func CreateBool(b bool) Object {
	return boolean{
		value: b,
	}
}

func (b boolean) Raw() any {
	return b.value
}

func (b boolean) Rev() (Object, error) {
	return nil, unsupportedOp("reverse", b)
}

func (b boolean) Not() (Object, error) {
	b.value = !b.value
	return b, nil
}

func (b boolean) String() string {
	return strconv.FormatBool(b.value)
}

func (b boolean) Add(_ Object) (Object, error) {
	return nil, unsupportedOp("addition", b)
}

func (b boolean) Sub(_ Object) (Object, error) {
	return nil, unsupportedOp("subtraction", b)
}

func (b boolean) Div(_ Object) (Object, error) {
	return nil, unsupportedOp("division", b)
}

func (b boolean) Mul(_ Object) (Object, error) {
	return nil, unsupportedOp("multiply", b)
}

func (b boolean) Mod(_ Object) (Object, error) {
	return nil, unsupportedOp("modulo", b)
}

func (b boolean) Pow(_ Object) (Object, error) {
	return nil, unsupportedOp("power", b)
}

func (b boolean) True() bool {
	return b.value
}

func (b boolean) Eq(other Object) (Object, error) {
	x, ok := other.(boolean)
	if !ok {
		return nil, incompatibleType("eq", b, other)
	}
	return CreateBool(b.value == x.value), nil
}

func (b boolean) Ne(other Object) (Object, error) {
	x, ok := other.(boolean)
	if !ok {
		return nil, incompatibleType("ne", b, other)
	}
	return CreateBool(b.value != x.value), nil
}

func (b boolean) Lt(other Object) (Object, error) {
	return nil, unsupportedOp("lt", b)
}

func (b boolean) Le(other Object) (Object, error) {
	return nil, unsupportedOp("le", b)
}

func (b boolean) Gt(other Object) (Object, error) {
	return nil, unsupportedOp("gt", b)
}

func (b boolean) Ge(other Object) (Object, error) {
	return nil, unsupportedOp("ge", b)
}

type array struct {
	values []Object
}

func CreateArray(list []Object) Object {
	vs := make([]Object, len(list))
	copy(vs, list)
	return array{
		values: list,
	}
}

func (a array) String() string {
	return fmt.Sprintf("%s", a.values)
}

func (a array) Raw() any {
	var list []any
	for i := range a.values {
		list = append(list, a.values[i].Raw())
	}
	return list
}

func (a array) Len() int {
	return len(a.values)
}

func (a array) True() bool {
	return len(a.values) > 0
}

func (a array) Not() (Object, error) {
	return CreateBool(!a.True()), nil
}

func (a array) Rev() (Object, error) {
	return nil, unsupportedOp("reverse", a)
}

func (a array) Add(other Object) (Object, error) {
	switch x := other.(type) {
	case array:
		a.values = append(a.values, x.values...)
	default:
		a.values = append(a.values, other)
	}
	return a, nil
}

func (a array) Sub(other Object) (Object, error) {
	var offset int
	switch x := other.(type) {
	case real:
		offset = int(x.value)
	default:
		return nil, incompatibleType("multiply", a, other)
	}
	if offset > len(a.values) || -offset > len(a.values) {
		a.values = []Object{}
		return a, nil
	}
	if offset < 0 {
		a.values = a.values[-offset:]
	} else {
		a.values = a.values[:len(a.values)-offset]
	}
	return a, nil
}

func (a array) Div(other Object) (Object, error) {
	var offset int
	switch x := other.(type) {
	case real:
		offset = int(x.value)
	default:
		return nil, incompatibleType("multiply", a, other)
	}
	if offset <= 0 {
		return nil, fmt.Errorf("array can not be divided negative values")
	}
	if offset > len(a.values) {
		return nil, fmt.Errorf("array can not be divided by %d", offset)
	}
	var (
		arr  array
		size = len(a.values)
		step = size / offset
	)
	for i := 0; i < size && len(arr.values) < offset; i += step {
		end := i + step
		if end > size || len(arr.values) == offset-1 {
			end = size
		}
		sub := a.values[i:end]
		arr.values = append(arr.values, CreateArray(sub))
	}
	return arr, nil
}

func (a array) Mul(other Object) (Object, error) {
	var offset int
	switch x := other.(type) {
	case real:
		offset = int(x.value)
	default:
		return nil, incompatibleType("multiply", a, other)
	}
	offset--

	vs := make([]Object, len(a.values))
	copy(vs, a.values)
	for i := 0; i < offset; i++ {
		a.values = append(a.values, vs...)
	}
	return a, nil
}

func (a array) Pow(other Object) (Object, error) {
	return nil, unsupportedOp("power", a)
}

func (a array) Mod(other Object) (Object, error) {
	return nil, unsupportedOp("modulo", a)
}

func (a array) Set(ix, value Object) (Object, error) {
	x, err := a.getIndex(ix)
	if err != nil {
		return nil, err
	}
	a.values[x] = value
	return a, nil
}

func (a array) Get(ix Object) (Object, error) {
	x, err := a.getIndex(ix)
	if err != nil {
		return nil, err
	}
	return a.values[x], nil
}

func (a array) getIndex(ix Object) (int, error) {
	var x int
	switch p := ix.(type) {
	case real:
		x = int(p.value)
	default:
		return x, fmt.Errorf("%T can not be used as index", ix)
	}
	if x < 0 {
		x = len(a.values) + x
	}
	if x < 0 || x >= len(a.values) {
		return x, fmt.Errorf("index out of range")
	}
	return x, nil
}

func unsupportedOp(op string, val Object) error {
	return fmt.Errorf("%s: %w for type %s", op, typeName(val))
}

func incompatibleType(op string, left, right Object) error {
	return fmt.Errorf("%s: %w %s/%s", op, ErrIncompatible, typeName(left), typeName(right))
}

func typeName(val Object) string {
	switch val.(type) {
	case varchar:
		return "string"
	case real:
		return "number"
	case boolean:
		return "boolean"
	case array:
		return "array"
	default:
		return "?"
	}
}
