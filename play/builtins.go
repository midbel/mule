package play

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"strings"
	"time"
)

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
