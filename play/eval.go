package play

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"

	"github.com/midbel/mule/environ"
)

func Default() environ.Environment[Value] {
	top := Empty()
	top.Define("console", makeConsole())
	top.Define("Math", makeMath())
	top.Define("JSON", makeJson())
	top.Define("Object", makeObject())
	top.Define("Array", makeArray())
	top.Define("parseInt", createBuiltinFunc("parseInt", execParseInt))
	top.Define("parseFloat", createBuiltinFunc("parseFloat", execParseFloat))
	top.Define("isNaN", createBuiltinFunc("isNaN", execIsNaN))

	return top
}

type Evaluable interface {
	Eval(Node) (Value, error)
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
	case Delete:
		return evalDelete(n, env)
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
	case Import:
		return evalImport(n, env)
	case Export:
		res, err := evalExport(n, env)
		if err != nil {
			return nil, ErrEval
		}
		_ = res
		return Void{}, nil
	default:
		return nil, ErrEval
	}
}

func evalImport(i Import, env environ.Environment[Value]) (Value, error) {
	u, err := url.Parse(i.From)
	if err != nil {
		return nil, err
	}
	var (
		r io.Reader
		n = path.Base(u.Path)
	)
	switch u.Scheme {
	case "http", "https":
		res, err := http.Get(i.From)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		r = res.Body
	default:
		res, err := os.Open(i.From)
		if err != nil {
			return nil, err
		}
		defer res.Close()
		r = res
	}
	mod := createModule(n)
	if _, err := EvalWithEnv(r, mod.Env); err != nil {
		return nil, err
	}
	if i.Type == nil {
		return Void{}, nil
	}
	switch i := i.Type.(type) {
	case DefaultImport:
		env.Define(i.Name, mod)
	case NamespaceImport:
		env.Define(i.Name, mod)
	case NamedImport:
		for ident, alias := range i.Names {
			if alias == "" {
				env.Define(ident, mod)
			} else {
				env.Define(alias, mod)
			}
		}
	default:
		return nil, ErrEval
	}
	return Void{}, nil
}

func evalExport(e Export, env environ.Environment[Value]) (Value, error) {
	return nil, nil
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
	_, err := eval(t.Node, Enclosed(env))
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
	return Void{}, err
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
	return Void{}, env.Define(ident.Name, letValue(res))
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
	return Void{}, env.Define(ident.Name, constValue(res))
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
		return Void{}, nil
	}
	return eval(i.Alt, Enclosed(sub))
}

func evalSwitch(s Switch, env environ.Environment[Value]) (Value, error) {
	value, err := eval(s.Cdt, env)
	if err != nil {
		return nil, err
	}
	var match bool
	for _, n := range s.Cases {
		c, ok := n.(Case)
		if !ok {
			return nil, ErrEval
		}
		sub := Enclosed(env)
		v, err := eval(c.Value, sub)
		if err != nil {
			return nil, err
		}
		if !isEqual(value, v) {
			continue
		}
		match = true
		if _, err = eval(c.Body, sub); err != nil {
			if errors.Is(err, ErrBreak) {
				break
			}
			return nil, err
		}
	}
	if !match && s.Default != nil {
		_, err = eval(s.Default, Enclosed(env))
		if err != nil {
			return nil, err
		}
	}
	return Void{}, nil
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
	if mod, ok := value.(Evaluable); ok {
		return mod.Eval(c)
	}
	var args []Value
	for i := range c.Args {
		a, err := eval(c.Args[i], env)
		if err != nil {
			return nil, err
		}
		if _, ok := c.Args[i].(Extend); ok {
			xs, ok := a.(*Array)
			if !ok {
				return nil, ErrEval
			}
			args = slices.Concat(args, xs.Values)
		} else {
			args = append(args, a)
		}
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
		if _, ok := k.(Extend); ok {
			tmp, ok := key.(*Object)
			if !ok {
				return nil, ErrEval
			}
			for k, v := range tmp.Fields {
				obj.Fields[k] = fieldByAssignment(v)
			}
			continue
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
		obj.Fields[key] = fieldByAssignment(val)
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
		if _, ok := n.(Extend); ok {
			xs, ok := v.(*Array)
			if !ok {
				return nil, ErrEval
			}
			arr.Values = slices.Concat(arr.Values, xs.Values)
		} else {
			arr.Values = append(arr.Values, v)
		}
	}
	return arr, nil
}

func evalAccess(a Access, env environ.Environment[Value]) (Value, error) {
	res, err := eval(a.Node, env)
	if err != nil {
		return nil, err
	}
	if mod, ok := res.(Evaluable); ok {
		return mod.Eval(a.Ident)
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
	res, err := eval(a.Node, env)
	if err != nil {
		return nil, err
	}
	switch ident := a.Ident.(type) {
	case Index:
		target, err := eval(ident.Ident, env)
		if err != nil {
			return nil, err
		}
		set, ok := target.(interface{ SetAt(Value, Value) error })
		if !ok {
			return nil, ErrOp
		}
		id, err := eval(ident.Expr, env)
		if err != nil {
			return nil, err
		}
		return res, set.SetAt(id, res)
	case Access:
		target, err := eval(ident.Node, env)
		if err != nil {
			return nil, err
		}
		set, ok := target.(interface{ Set(Value, Value) error })
		if !ok {
			return nil, ErrOp
		}
		id, ok := ident.Ident.(Identifier)
		if !ok {
			return nil, ErrOp
		}
		return res, set.Set(getString(id.Name), res)
	case Identifier:
		if v, err := env.Resolve(ident.Name); err == nil {
			e, ok := v.(envValue)
			if ok && e.Const {
				return nil, ErrConst
			}
		}
		return res, env.Define(ident.Name, letValue(res))
	default:
		return nil, ErrEval
	}
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

func evalDelete(d Delete, env environ.Environment[Value]) (Value, error) {
	switch n := d.Node.(type) {
	case Access:
		res, err := eval(n.Node, env)
		if err != nil {
			return nil, err
		}
		ident, ok := n.Ident.(Identifier)
		if !ok {
			return nil, ErrOp
		}
		del, ok := res.(interface{ Del(Value) error })
		if !ok {
			return nil, ErrOp
		}
		if err = del.Del(getString(ident.Name)); err != nil {
			return getBool(false), nil
		}
		return getBool(true), nil
	case Index:
		res, err := eval(n.Ident, env)
		if err != nil {
			return nil, err
		}
		expr, err := eval(n.Expr, env)
		if err != nil {
			return nil, err
		}
		del, ok := res.(interface{ DelAt(Value) error })
		if !ok {
			return nil, ErrOp
		}
		if err = del.DelAt(expr); err != nil {
			return getBool(false), nil
		}
		return getBool(true), nil
	default:
		return getBool(false), nil
	}
}

func evalUnary(u Unary, env environ.Environment[Value]) (Value, error) {
	right, err := eval(u.Node, env)
	if err != nil {
		return nil, err
	}
	switch u.Op {
	default:
		return nil, ErrEval
	case TypeOf:
		res, ok := right.(interface{ Type() string })
		if !ok {
			return nil, ErrOp
		}
		return getString(res.Type()), nil
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
	case Nullish:
		if isNull(left) || isUndefined(left) {
			return right, nil
		}
		return left, nil
	case InstanceOf:
		return nil, ErrEval
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
