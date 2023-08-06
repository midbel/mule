package eval

import (
	"errors"
	"fmt"
	"io"

	"github.com/midbel/mule/env"
)

var (
	errBreak    = errors.New("break")
	errContinue = errors.New("continue")
	errReturn   = errors.New("return")
	errThrow    = errors.New("throw")
)

func Eval(r io.Reader) (Value, error) {
	expr, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return EvalExpr(expr, env.EmptyEnv[Value]())
}

func EvalExpr(node Expression, ev env.Env[Value]) (Value, error) {

	ev.Define("Math", Math{})
	ev.Define("Date", Date{})
	ev.Define("console", Console{})

	v, err := eval(node, ev)
	if errors.Is(err, errReturn) {
		err = nil
	}
	return v, err
}

func eval(node Expression, ev env.Env[Value]) (Value, error) {
	switch n := node.(type) {
	case Primitive[float64]:
		return evalNumber(n, ev)
	case Primitive[string]:
		return evalString(n, ev)
	case Primitive[bool]:
		return evalBool(n, ev)
	case Variable:
		return evalVariable(n, ev)
	case Function:
		return evalFunction(n, ev)
	case ArrowFunction:
		return evalArrow(n, ev)
	case Chain:
		return evalChain(n, ev)
	case Index:
		return evalIndex(n, ev)
	case Array:
		return evalArray(n, ev)
	case Hash:
		return evalHash(n, ev)
	case Call:
		return evalCall(n, ev)
	case Return:
		return evalReturn(n, ev)
	case Block:
		return evalBlock(n, ev)
	case Assignment:
		return evalAssignment(n, ev)
	case Binary:
		return evalBinary(n, ev)
	case Unary:
		return evalUnary(n, ev)
	case Let:
		return evalLet(n, ev)
	case If:
		return evalIf(n, ev)
	case Switch:
	case For:
	case While:
		return evalWhile(n, ev)
	case Break:
		return nil, errBreak
	case Continue:
		return nil, errContinue
	case Try:
		return evalTry(n, ev)
	case Throw:
	case Catch:
		return evalCatch(n, ev)
	default:
		return nil, fmt.Errorf("%T unsupported node type", node)
	}
	return nil, nil
}

func evalString(p Primitive[string], _ env.Env[Value]) (Value, error) {
	return CreateValue(p.Literal)
}

func evalNumber(p Primitive[float64], _ env.Env[Value]) (Value, error) {
	return CreateValue(p.Literal)
}

func evalBool(p Primitive[bool], _ env.Env[Value]) (Value, error) {
	return CreateValue(p.Literal)
}

func evalVariable(v Variable, ev env.Env[Value]) (Value, error) {
	return ev.Resolve(v.Ident)
}

func evalArrow(f ArrowFunction, ev env.Env[Value]) (Value, error) {
	return nil, nil
}

func evalFunction(f Function, ev env.Env[Value]) (Value, error) {
	var fn function
	fn.args = append(fn.args, f.Args...)
	fn.body = f.Body
	if f.Name != "" {
		ev.Define(f.Name, fn)
	}
	return fn, nil
}

func evalChain(c Chain, ev env.Env[Value]) (Value, error) {
	return nil, nil
}

func evalIndex(i Index, ev env.Env[Value]) (Value, error) {
	return nil, nil
}

func evalArray(a Array, ev env.Env[Value]) (Value, error) {
	var arr []Value
	for i := range a.List {
		v, err := eval(a.List[i], ev)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	return CreateArray(arr), nil
}

func evalHash(h Hash, ev env.Env[Value]) (Value, error) {
	return nil, nil
}

func evalCall(c Call, ev env.Env[Value]) (Value, error) {
	id, ok := c.Ident.(Variable)
	if !ok {
		return nil, fmt.Errorf("identifier is not a variable")
	}
	value, err := ev.Resolve(id.Ident)
	if err != nil {
		return nil, err
	}
	fn, ok := value.(function)
	if !ok {
		return nil, fmt.Errorf("value is not a callable")
	}
	if len(c.Args) != len(fn.args) {
		return nil, fmt.Errorf("invalid number of arguments given")
	}
	tmp := env.EnclosedEnv[Value](ev)
	for i, a := range fn.args {
		ag, ok := a.(Argument)
		if !ok {
			return nil, fmt.Errorf("invalid argument given")
		}
		v, err := eval(c.Args[i], ev)
		if err != nil {
			return nil, err
		}
		tmp.Define(ag.Ident, v)
	}
	res, err := eval(fn.body, tmp)
	if errors.Is(err, errReturn) {
		err = nil
	}
	return res, err
}

func evalReturn(r Return, ev env.Env[Value]) (Value, error) {
	v, err := eval(r.Expr, ev)
	if err == nil {
		err = errReturn
	}
	return v, err
}

func evalBlock(b Block, ev env.Env[Value]) (Value, error) {
	var (
		res Value
		err error
		tmp = env.EnclosedEnv[Value](ev)
	)
	for i := range b.List {
		res, err = eval(b.List[i], tmp)
		if err != nil {
			if errors.Is(err, errReturn) {
				break
			}
			return nil, err
		}
	}
	return res, err
}

func evalBinary(b Binary, ev env.Env[Value]) (Value, error) {
	left, err := eval(b.Left, ev)
	if err != nil {
		return nil, err
	}
	right, err := eval(b.Right, ev)
	if err != nil {
		return nil, err
	}
	switch b.Op {
	case Add:
		if a, ok := left.(Arithmetic); ok {
			return a.Add(right)
		}
	case Sub:
		if s, ok := left.(Arithmetic); ok {
			return s.Sub(right)
		}
	case Mul:
		if m, ok := left.(Arithmetic); ok {
			return m.Mul(right)
		}
	case Div:
		if d, ok := left.(Arithmetic); ok {
			return d.Div(right)
		}
	case Pow:
		if p, ok := left.(Arithmetic); ok {
			return p.Pow(right)
		}
	case Mod:
		if m, ok := left.(Arithmetic); ok {
			return m.Mod(right)
		}
	case Lshift:
	case Rshift:
	case Band:
	case Bor:
	case And:
		return leftAndRight(left, right)
	case Or:
		return leftOrRight(left, right)
	case Eq:
	case Ne:
	case Lt:
	case Le:
	case Gt:
	case Ge:
	default:
		return nil, fmt.Errorf("unsupported operator")
	}
	return nil, ErrOperation
}

func evalAssignment(a Assignment, ev env.Env[Value]) (Value, error) {
	ident, ok := a.Ident.(Variable)
	if !ok {
		return nil, fmt.Errorf("variable expected")
	}
	value, err := eval(a.Expr, ev)
	if err != nil {
		return nil, err
	}
	return value, ev.Assign(ident.Ident, value)
}

func evalUnary(u Unary, ev env.Env[Value]) (Value, error) {
	return nil, nil
}

func evalLet(e Let, ev env.Env[Value]) (Value, error) {
	val, err := eval(e.Expr, ev)
	if err == nil {
		ev.Define(e.Ident, val)
	}
	return val, err
}

func evalIf(i If, ev env.Env[Value]) (Value, error) {
	v, err := eval(i.Cdt, ev)
	if err != nil {
		return nil, err
	}
	tmp := env.EnclosedEnv[Value](ev)
	if v.True() {
		return eval(i.Csq, tmp)
	}
	if i.Alt != nil {
		return eval(i.Alt, tmp)
	}
	return nil, nil
}

func evalDo(w While, ev env.Env[Value]) (Value, error) {
	var (
		res Value
		err error
	)
	for i := 0; ; i++ {
		if i > 0 {
			v, err := eval(w.Cdt, ev)
			if err != nil {
				return nil, err
			}
			if !v.True() {
				break
			}
		}
		res, err = eval(w.Body, env.EnclosedEnv[Value](ev))
		if err != nil {
			if errors.Is(err, errBreak) {
				return res, err
			} else if errors.Is(err, errContinue) {
				continue
			}
			return nil, err
		}
	}
	return res, err
}

func evalWhile(w While, ev env.Env[Value]) (Value, error) {
	if w.Do {
		return evalDo(w, ev)
	}
	var (
		res Value
		err error
	)
	for {
		v, err := eval(w.Cdt, ev)
		if err != nil {
			return nil, err
		}
		if !v.True() {
			break
		}
		res, err = eval(w.Body, env.EnclosedEnv[Value](ev))
		if err != nil {
			if errors.Is(err, errBreak) {
				return res, err
			} else if errors.Is(err, errContinue) {
				continue
			}
			return nil, err
		}
	}
	return res, err
}

func evalTry(t Try, ev env.Env[Value]) (Value, error) {
	tmp := env.EnclosedEnv[Value](ev)
	v, err := eval(t.Body, tmp)
	if errors.Is(err, errThrow) && t.Catch != nil {
		v, err = eval(t.Catch, tmp)
	}
	if t.Finally != nil {
		eval(t.Finally, tmp)
	}
	return v, err
}

func evalCatch(c Catch, ev env.Env[Value]) (Value, error) {
	return nil, nil
}
