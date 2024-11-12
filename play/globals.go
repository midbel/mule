package play

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/midbel/mule/codecs/json"
	"github.com/midbel/mule/jwt"
)

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

func makeXml() Value {
	g := global{
		name:  "XML",
		fnset: make(map[string]Callable),
	}

	g.fnset["parse"] = asCallable(xmlParse)
	g.fnset["stringify"] = asCallable(xmlString)

	return g
}

func xmlParse(args []Value) (Value, error) {
	return nil, nil
}

func xmlString(args []Value) (Value, error) {
	return nil, nil
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
	if len(args) > 2 || len(args) == 0 {
		return Void{}, ErrArgument
	}
	str, ok := args[0].(String)
	if !ok {
		return args[0], nil
	}
	doc, err := json.Parse(strings.NewReader(str.value))
	if err != nil {
		return nil, err
	}
	if len(args) == 2 {
		q, ok := args[1].(String)
		if !ok {
			return nil, ErrArgument
		}
		query, err := json.Compile(q.value)
		if err != nil {
			return nil, err
		}
		doc, err = query.Get(doc)
		if err != nil {
			return nil, err
		}
	}
	return NativeToValues(doc)
}

func jsonString(args []Value) (Value, error) {
	if len(args) != 1 {
		return Void{}, ErrArgument
	}
	v, err := ValuesToNative(args[0])
	if err != nil {
		return nil, err
	}
	var (
		buf bytes.Buffer
		ws  = json.NewWriter(&buf)
	)
	if err := ws.Write(v); err != nil {
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
			f, ok := v.(Field)
			if !ok {
				return nil, fmt.Errorf("unexpected value type")
			}
			vv, err := ValuesToNative(f.Value)
			if err != nil {
				return nil, err
			}
			arr[fmt.Sprintf("%s", k)] = vv
		}
		return arr, nil
	case Nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("type can not be converted to json")
	}
}

func NativeToValues(obj interface{}) (Value, error) {
	if obj == nil {
		return Nil{}, nil
	}
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

type jwtGlobal struct {
	config *jwt.Config
}

func makeJWT() Value {
	config := jwt.Config{
		Secret: "",
		Alg:    jwt.HS256,
		Ttl:    time.Hour * 24,
	}
	g := jwtGlobal{
		config: &config,
	}
	return g
}

func (g jwtGlobal) Type() string {
	return "object"
}

func (g jwtGlobal) True() Value {
	return getBool(true)
}

func (g jwtGlobal) String() string {
	return jwt.JWT
}

func (g jwtGlobal) Get(prop Value) (Value, error) {
	str, ok := prop.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch name := str.String(); name {
	case "secret":
		return getString(g.config.Secret), nil
	case "alg":
		return getString(g.config.Alg), nil
	case jwt.HS256, jwt.HS384, jwt.HS512:
		return getString(name), nil
	default:
		return Void{}, fmt.Errorf("%s: undefined property", name)
	}
}

func (g jwtGlobal) Set(prop, value Value) error {
	str, ok := prop.(fmt.Stringer)
	if !ok {
		return ErrEval
	}
	switch name := str.String(); name {
	case "secret":
		str, ok := value.(String)
		if !ok {
			return ErrEval
		}
		g.config.Secret = str.String()
	case "alg":
		str, ok := value.(String)
		if !ok {
			return ErrEval
		}
		g.config.Alg = str.String()
	default:
		return fmt.Errorf("%s: undefined property", name)
	}
	return nil
}

func (g jwtGlobal) Call(ident string, args []Value) (Value, error) {
	if len(args) == 0 || len(args) > 2 {
		return Void{}, ErrArgument
	}
	cfg := g.config
	if len(args) == 2 {
		c, err := jwtConfigure(args[1])
		if err != nil {
			return Void{}, err
		}
		cfg = c
	}
	switch ident {
	case "encode":
		return jwtEncode(args[0], cfg)
	case "decode":
		return jwtDecode(args[0], cfg)
	default:
		return nil, fmt.Errorf("%s.%s: undefined function", jwt.JWT, ident)
	}
}

func jwtConfigure(arg Value) (*jwt.Config, error) {
	getter, ok := arg.(interface{ Get(Value) (Value, error) })
	if !ok {
		return nil, ErrEval
	}

	strFromGet := func(ident string) (string, error) {
		v, err := getter.Get(getString(ident))
		if err != nil {
			return "", err
		}
		str, ok := v.(fmt.Stringer)
		if !ok {
			return "", ErrEval
		}
		return str.String(), nil
	}
	var (
		cfg jwt.Config
		err error
	)
	if cfg.Alg, err = strFromGet("alg"); err != nil {
		return nil, err
	}
	if cfg.Secret, err = strFromGet("secret"); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func jwtDecode(arg Value, cfg *jwt.Config) (Value, error) {
	str, ok := arg.(String)
	if !ok {
		return Void{}, ErrEval
	}
	body, err := jwt.Decode(str.String(), cfg)
	if err != nil {
		return Void{}, err
	}
	obj := createObject()
	for k, v := range body {
		vs, err := NativeToValues(v)
		if err != nil {
			return Void{}, err
		}
		obj.Set(getString(k), vs)
	}
	return obj, nil
}

func jwtEncode(arg Value, cfg *jwt.Config) (Value, error) {
	val, err := ValuesToNative(arg)
	if err != nil {
		return nil, err
	}
	str, err := jwt.Encode(val, cfg)
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
